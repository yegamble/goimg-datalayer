package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for comment operations.
const (
	sqlInsertComment = `
		INSERT INTO comments (id, user_id, image_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	sqlSelectCommentByID = `
		SELECT id, user_id, image_id, content, created_at, updated_at
		FROM comments
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSoftDeleteComment = `
		UPDATE comments
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectCommentsByImageID = `
		SELECT id, user_id, image_id, content, created_at, updated_at
		FROM comments
		WHERE image_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	sqlCountCommentsByImageID = `
		SELECT COUNT(*)
		FROM comments
		WHERE image_id = $1 AND deleted_at IS NULL
	`

	sqlExistsComment = `
		SELECT EXISTS(
			SELECT 1 FROM comments
			WHERE id = $1 AND deleted_at IS NULL
		)
	`
)

// commentRow represents a comment row in the database.
type commentRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ImageID   string    `db:"image_id"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// CommentRepository implements the gallery.CommentRepository interface for PostgreSQL.
type CommentRepository struct {
	db *sqlx.DB
}

// NewCommentRepository creates a new CommentRepository with the given database connection.
func NewCommentRepository(db *sqlx.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// NextID generates the next available CommentID.
func (r *CommentRepository) NextID() gallery.CommentID {
	return gallery.NewCommentID()
}

// FindByID retrieves a comment by its unique ID.
func (r *CommentRepository) FindByID(ctx context.Context, id gallery.CommentID) (*gallery.Comment, error) {
	var row commentRow
	if err := r.db.GetContext(ctx, &row, sqlSelectCommentByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to find comment by id: %w", err)
	}

	comment, err := rowToComment(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to comment: %w", err)
	}

	return comment, nil
}

// FindByImage retrieves all non-deleted comments for an image with pagination.
// Comments are ordered by creation time (oldest first).
func (r *CommentRepository) FindByImage(
	ctx context.Context,
	imageID gallery.ImageID,
	pagination shared.Pagination,
) ([]*gallery.Comment, int64, error) {
	// Get paginated comments
	var rows []commentRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectCommentsByImageID,
		imageID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find comments by image: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountCommentsByImageID, imageID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments by image: %w", err)
	}

	// Convert rows to domain entities
	comments := make([]*gallery.Comment, 0, len(rows))
	for _, row := range rows {
		comment, err := rowToComment(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to comment: %w", err)
		}
		comments = append(comments, comment)
	}

	return comments, total, nil
}

// FindByUser retrieves all non-deleted comments by a user with pagination.
func (r *CommentRepository) FindByUser(
	ctx context.Context,
	userID identity.UserID,
	pagination shared.Pagination,
) ([]*gallery.Comment, int64, error) {
	sqlSelectCommentsByUserID := `
		SELECT id, user_id, image_id, content, created_at, updated_at
		FROM comments
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountCommentsByUserID := `
		SELECT COUNT(*)
		FROM comments
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	// Get paginated comments
	var rows []commentRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectCommentsByUserID,
		userID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find comments by user: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountCommentsByUserID, userID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments by user: %w", err)
	}

	// Convert rows to domain entities
	comments := make([]*gallery.Comment, 0, len(rows))
	for _, row := range rows {
		comment, err := rowToComment(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to comment: %w", err)
		}
		comments = append(comments, comment)
	}

	return comments, total, nil
}

// CountByImage returns the number of non-deleted comments on an image.
func (r *CommentRepository) CountByImage(ctx context.Context, imageID gallery.ImageID) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, sqlCountCommentsByImageID, imageID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count comments by image: %w", err)
	}

	return count, nil
}

// Save persists a comment (insert only - comments are immutable).
func (r *CommentRepository) Save(ctx context.Context, comment *gallery.Comment) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertComment,
		comment.ID().String(),
		comment.UserID().String(),
		comment.ImageID().String(),
		comment.Content(),
		comment.CreatedAt(),
		comment.CreatedAt(), // updated_at = created_at initially
	)
	if err != nil {
		return fmt.Errorf("failed to save comment: %w", err)
	}

	return nil
}

// Delete soft-deletes a comment by setting deleted_at timestamp.
func (r *CommentRepository) Delete(ctx context.Context, id gallery.CommentID) error {
	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx, sqlSoftDeleteComment, id.String(), now)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrCommentNotFound
	}

	return nil
}

// ExistsByID checks if a comment exists and is not deleted.
func (r *CommentRepository) ExistsByID(ctx context.Context, id gallery.CommentID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsComment, id.String())
	if err != nil {
		return false, fmt.Errorf("failed to check comment existence: %w", err)
	}

	return exists, nil
}

// rowToComment converts a database row to a domain Comment entity.
func rowToComment(row commentRow) (*gallery.Comment, error) {
	// Parse IDs
	commentID, err := gallery.ParseCommentID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid comment id: %w", err)
	}

	imageID, err := gallery.ParseImageID(row.ImageID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Reconstitute comment without validation or events
	comment := gallery.ReconstructComment(
		commentID,
		imageID,
		userID,
		row.Content,
		row.CreatedAt,
	)

	return comment, nil
}
