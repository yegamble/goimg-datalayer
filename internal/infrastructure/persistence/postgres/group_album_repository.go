package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group album operations.
const (
	sqlInsertGroupAlbum = `
		INSERT INTO group_albums (
			id, group_id, created_by, title, description, cover_image_id,
			image_count, is_public, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	sqlUpdateGroupAlbum = `
		UPDATE group_albums
		SET title = $2,
		    description = $3,
		    cover_image_id = $4,
		    image_count = $5,
		    is_public = $6,
		    updated_at = $7
		WHERE id = $1
	`

	sqlSelectGroupAlbumByID = `
		SELECT id, group_id, created_by, title, description, cover_image_id,
		       image_count, is_public, created_at, updated_at
		FROM group_albums
		WHERE id = $1
	`

	sqlSelectGroupAlbumsByGroup = `
		SELECT id, group_id, created_by, title, description, cover_image_id,
		       image_count, is_public, created_at, updated_at
		FROM group_albums
		WHERE group_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlSelectGroupAlbumsByGroupAndCreator = `
		SELECT id, group_id, created_by, title, description, cover_image_id,
		       image_count, is_public, created_at, updated_at
		FROM group_albums
		WHERE group_id = $1 AND created_by = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	sqlCountGroupAlbumsByGroup = `
		SELECT COUNT(*) FROM group_albums WHERE group_id = $1
	`

	sqlCountGroupAlbumsByGroupAndCreator = `
		SELECT COUNT(*) FROM group_albums WHERE group_id = $1 AND created_by = $2
	`

	sqlDeleteGroupAlbum = `
		DELETE FROM group_albums WHERE id = $1
	`

	sqlExistsGroupAlbum = `
		SELECT EXISTS(SELECT 1 FROM group_albums WHERE id = $1)
	`
)

// groupAlbumRow represents a group album row in the database.
type groupAlbumRow struct {
	ID           string         `db:"id"`
	GroupID      string         `db:"group_id"`
	CreatedBy    string         `db:"created_by"`
	Title        string         `db:"title"`
	Description  string         `db:"description"`
	CoverImageID sql.NullString `db:"cover_image_id"`
	ImageCount   int            `db:"image_count"`
	IsPublic     bool           `db:"is_public"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

// GroupAlbumRepository implements the community.GroupAlbumRepository interface for PostgreSQL.
type GroupAlbumRepository struct {
	db *sqlx.DB
}

// NewGroupAlbumRepository creates a new GroupAlbumRepository with the given database connection.
func NewGroupAlbumRepository(db *sqlx.DB) *GroupAlbumRepository {
	return &GroupAlbumRepository{db: db}
}

// Save persists the group album to storage.
// This handles both creation and updates.
func (r *GroupAlbumRepository) Save(ctx context.Context, album *community.GroupAlbum) error {
	// Check if album exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsGroupAlbum, album.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check group album existence: %w", err)
	}

	if exists {
		return r.update(ctx, album)
	}
	return r.insert(ctx, album)
}

// FindByID retrieves a group album by its ID.
// Returns ErrGroupAlbumNotFound if the album doesn't exist.
func (r *GroupAlbumRepository) FindByID(ctx context.Context, id community.GroupAlbumID) (*community.GroupAlbum, error) {
	var row groupAlbumRow
	if err := r.db.GetContext(ctx, &row, sqlSelectGroupAlbumByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrGroupAlbumNotFound
		}
		return nil, fmt.Errorf("failed to find group album by id: %w", err)
	}

	album, err := rowToGroupAlbum(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to group album: %w", err)
	}

	return album, nil
}

// FindByGroup retrieves albums for a specific group with pagination.
// Albums are returned in reverse chronological order (newest first).
// Returns the albums, total count, and error.
func (r *GroupAlbumRepository) FindByGroup(
	ctx context.Context,
	groupID community.GroupID,
	pagination shared.Pagination,
) ([]*community.GroupAlbum, int, error) {
	// Execute main query
	var rows []groupAlbumRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectGroupAlbumsByGroup,
		groupID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find group albums by group: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountGroupAlbumsByGroup, groupID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count group albums by group: %w", err)
	}

	// Convert rows to domain entities
	albums, err := rowsToGroupAlbums(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to group albums: %w", err)
	}

	return albums, total, nil
}

// FindByGroupAndCreator retrieves albums created by a specific user within a group.
// Useful for filtering by album creator.
// Returns the albums, total count, and error.
func (r *GroupAlbumRepository) FindByGroupAndCreator(
	ctx context.Context,
	groupID community.GroupID,
	creatorID identity.UserID,
	pagination shared.Pagination,
) ([]*community.GroupAlbum, int, error) {
	// Execute main query
	var rows []groupAlbumRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectGroupAlbumsByGroupAndCreator,
		groupID.String(),
		creatorID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find group albums by group and creator: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountGroupAlbumsByGroupAndCreator, groupID.String(), creatorID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count group albums by group and creator: %w", err)
	}

	// Convert rows to domain entities
	albums, err := rowsToGroupAlbums(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to group albums: %w", err)
	}

	return albums, total, nil
}

// Delete removes the group album from storage.
// This is a hard delete since albums are just organizational containers.
func (r *GroupAlbumRepository) Delete(ctx context.Context, id community.GroupAlbumID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteGroupAlbum, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete group album: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrGroupAlbumNotFound
	}

	return nil
}

// insert creates a new group album in the database.
func (r *GroupAlbumRepository) insert(ctx context.Context, album *community.GroupAlbum) error {
	var coverImageID sql.NullString
	if album.CoverImageID() != nil {
		coverImageID = sql.NullString{String: album.CoverImageID().String(), Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertGroupAlbum,
		album.ID().String(),
		album.GroupID().String(),
		album.CreatedBy().String(),
		album.Title(),
		album.Description(),
		coverImageID,
		album.ImageCount(),
		album.IsPublic(),
		album.CreatedAt(),
		album.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert group album: %w", err)
	}

	return nil
}

// update updates an existing group album in the database.
func (r *GroupAlbumRepository) update(ctx context.Context, album *community.GroupAlbum) error {
	var coverImageID sql.NullString
	if album.CoverImageID() != nil {
		coverImageID = sql.NullString{String: album.CoverImageID().String(), Valid: true}
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateGroupAlbum,
		album.ID().String(),
		album.Title(),
		album.Description(),
		coverImageID,
		album.ImageCount(),
		album.IsPublic(),
		album.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update group album: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrGroupAlbumNotFound
	}

	return nil
}

// rowToGroupAlbum converts a database row to a GroupAlbum domain entity.
func rowToGroupAlbum(row groupAlbumRow) (*community.GroupAlbum, error) {
	// Parse IDs
	albumID, err := community.ParseGroupAlbumID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group album id: %w", err)
	}

	groupID, err := community.ParseGroupID(row.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}

	createdBy, err := identity.ParseUserID(row.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid creator id: %w", err)
	}

	// Parse optional cover image ID
	var coverImageID *gallery.ImageID
	if row.CoverImageID.Valid {
		imageID, err := gallery.ParseImageID(row.CoverImageID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid cover image id: %w", err)
		}
		coverImageID = &imageID
	}

	// Reconstitute album without validation or events
	album := community.ReconstructGroupAlbum(
		albumID,
		groupID,
		createdBy,
		row.Title,
		row.Description,
		coverImageID,
		row.ImageCount,
		row.IsPublic,
		row.CreatedAt,
		row.UpdatedAt,
	)

	return album, nil
}

// rowsToGroupAlbums converts multiple group album rows to domain entities.
func rowsToGroupAlbums(rows []groupAlbumRow) ([]*community.GroupAlbum, error) {
	albums := make([]*community.GroupAlbum, 0, len(rows))
	for _, row := range rows {
		album, err := rowToGroupAlbum(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row: %w", err)
		}
		albums = append(albums, album)
	}
	return albums, nil
}
