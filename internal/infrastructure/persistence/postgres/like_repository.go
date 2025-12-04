package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for like operations.
const (
	sqlInsertLike = `
		INSERT INTO likes (user_id, image_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, image_id) DO NOTHING
	`

	sqlDeleteLike = `
		DELETE FROM likes
		WHERE user_id = $1 AND image_id = $2
	`

	sqlCheckLikeExists = `
		SELECT EXISTS(
			SELECT 1 FROM likes
			WHERE user_id = $1 AND image_id = $2
		)
	`

	sqlCountLikesByImage = `
		SELECT COUNT(*)
		FROM likes
		WHERE image_id = $1
	`

	sqlSelectLikedImageIDsByUser = `
		SELECT image_id
		FROM likes
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountLikedImagesByUser = `
		SELECT COUNT(*)
		FROM likes
		WHERE user_id = $1
	`
)

// LikeRepository implements like operations for PostgreSQL.
type LikeRepository struct {
	db *sqlx.DB
}

// NewLikeRepository creates a new LikeRepository with the given database connection.
func NewLikeRepository(db *sqlx.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// Like creates a like relationship between a user and an image.
// Returns nil if the like already exists (idempotent).
func (r *LikeRepository) Like(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error {
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, sqlInsertLike, userID.String(), imageID.String(), now)
	if err != nil {
		return fmt.Errorf("failed to create like: %w", err)
	}

	return nil
}

// Unlike removes a like relationship between a user and an image.
// Returns nil if the like doesn't exist (idempotent).
func (r *LikeRepository) Unlike(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error {
	_, err := r.db.ExecContext(ctx, sqlDeleteLike, userID.String(), imageID.String())
	if err != nil {
		return fmt.Errorf("failed to remove like: %w", err)
	}

	return nil
}

// HasLiked checks if a user has liked an image.
func (r *LikeRepository) HasLiked(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckLikeExists, userID.String(), imageID.String())
	if err != nil {
		return false, fmt.Errorf("failed to check like existence: %w", err)
	}

	return exists, nil
}

// GetLikeCount returns the total number of likes for an image.
func (r *LikeRepository) GetLikeCount(ctx context.Context, imageID gallery.ImageID) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, sqlCountLikesByImage, imageID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count likes: %w", err)
	}

	return count, nil
}

// GetLikedImageIDs returns a paginated list of image IDs that a user has liked.
// Results are ordered by most recently liked first.
func (r *LikeRepository) GetLikedImageIDs(
	ctx context.Context,
	userID identity.UserID,
	pagination shared.Pagination,
) ([]uuid.UUID, error) {
	var imageIDs []string
	err := r.db.SelectContext(
		ctx,
		&imageIDs,
		sqlSelectLikedImageIDsByUser,
		userID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get liked image IDs: %w", err)
	}

	// Convert string IDs to UUIDs
	result := make([]uuid.UUID, 0, len(imageIDs))
	for _, idStr := range imageIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse image ID %s: %w", idStr, err)
		}
		result = append(result, id)
	}

	return result, nil
}

// CountLikedImagesByUser returns the total number of images a user has liked.
func (r *LikeRepository) CountLikedImagesByUser(ctx context.Context, userID identity.UserID) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, sqlCountLikedImagesByUser, userID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count liked images: %w", err)
	}

	return count, nil
}
