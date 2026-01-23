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

// SQL queries for album operations.
const (
	sqlInsertAlbum = `
		INSERT INTO albums (
			id, owner_id, parent_id, title, description, visibility, cover_image_id,
			image_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	sqlUpdateAlbum = `
		UPDATE albums
		SET title = $2,
		    description = $3,
		    visibility = $4,
		    cover_image_id = $5,
		    image_count = $6,
		    parent_id = $7,
		    updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectAlbumByID = `
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM albums
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectAlbumsByOwner = `
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM albums
		WHERE owner_id = $1
		  AND ($4 = '' OR visibility = $4)
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountAlbumsByOwner = `
		SELECT COUNT(*)
		FROM albums
		WHERE owner_id = $1
		  AND ($2 = '' OR visibility = $2)
		  AND deleted_at IS NULL
	`

	sqlSelectPublicAlbums = `
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM albums
		WHERE visibility = 'public' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	sqlCountPublicAlbums = `
		SELECT COUNT(*)
		FROM albums
		WHERE visibility = 'public' AND deleted_at IS NULL
	`

	sqlDeleteAlbum = `
		DELETE FROM albums WHERE id = $1
	`

	sqlExistsAlbum = `
		SELECT EXISTS(SELECT 1 FROM albums WHERE id = $1 AND deleted_at IS NULL)
	`

	// Nested album queries
	sqlSelectChildAlbums = `
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM albums
		WHERE parent_id = $1 AND deleted_at IS NULL
		ORDER BY title ASC
	`

	sqlSelectRootAlbumsByOwner = `
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM albums
		WHERE owner_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	// Recursive CTE to get all ancestors (breadcrumb)
	sqlSelectAlbumAncestors = `
		WITH RECURSIVE ancestors AS (
			SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
			       image_count, created_at, updated_at, 0 as depth
			FROM albums
			WHERE id = $1 AND deleted_at IS NULL

			UNION ALL

			SELECT a.id, a.owner_id, a.parent_id, a.title, a.description, a.visibility,
			       a.cover_image_id, a.image_count, a.created_at, a.updated_at, anc.depth + 1
			FROM albums a
			INNER JOIN ancestors anc ON a.id = anc.parent_id
			WHERE a.deleted_at IS NULL
		)
		SELECT id, owner_id, parent_id, title, description, visibility, cover_image_id,
		       image_count, created_at, updated_at
		FROM ancestors
		ORDER BY depth DESC
	`
)

// albumRow represents an album row in the database.
type albumRow struct {
	ID           string    `db:"id"`
	OwnerID      string    `db:"owner_id"`
	ParentID     *string   `db:"parent_id"`
	Title        string    `db:"title"`
	Description  string    `db:"description"`
	Visibility   string    `db:"visibility"`
	CoverImageID *string   `db:"cover_image_id"`
	ImageCount   int       `db:"image_count"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// AlbumRepository implements the gallery.AlbumRepository interface for PostgreSQL.
type AlbumRepository struct {
	db *sqlx.DB
}

// NewAlbumRepository creates a new AlbumRepository with the given database connection.
func NewAlbumRepository(db *sqlx.DB) *AlbumRepository {
	return &AlbumRepository{db: db}
}

// NextID generates the next available AlbumID.
func (r *AlbumRepository) NextID() gallery.AlbumID {
	return gallery.NewAlbumID()
}

// FindByID retrieves an album by its unique ID.
func (r *AlbumRepository) FindByID(ctx context.Context, id gallery.AlbumID) (*gallery.Album, error) {
	var row albumRow
	if err := r.db.GetContext(ctx, &row, sqlSelectAlbumByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrAlbumNotFound
		}
		return nil, fmt.Errorf("failed to find album by id: %w", err)
	}

	album, err := rowToAlbum(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to album: %w", err)
	}

	return album, nil
}

// FindByOwner retrieves all albums owned by a user with pagination.
func (r *AlbumRepository) FindByOwner(
	ctx context.Context,
	ownerID identity.UserID,
	pagination shared.Pagination,
	visibility *gallery.Visibility,
) ([]*gallery.Album, int64, error) {
	visStr := ""
	if visibility != nil {
		visStr = visibility.String()
	}

	var rows []albumRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectAlbumsByOwner,
		ownerID.String(),
		pagination.Limit(),
		pagination.Offset(),
		visStr,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find albums by owner: %w", err)
	}

	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountAlbumsByOwner, ownerID.String(), visStr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count albums by owner: %w", err)
	}

	albums := make([]*gallery.Album, 0, len(rows))
	for _, row := range rows {
		album, err := rowToAlbum(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, total, nil
}

// FindPublic retrieves all public albums with pagination.
func (r *AlbumRepository) FindPublic(
	ctx context.Context,
	pagination shared.Pagination,
) ([]*gallery.Album, int64, error) {
	// Get paginated albums
	var rows []albumRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectPublicAlbums,
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find public albums: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountPublicAlbums)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count public albums: %w", err)
	}

	// Convert rows to domain entities
	albums := make([]*gallery.Album, 0, len(rows))
	for _, row := range rows {
		album, err := rowToAlbum(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, total, nil
}

// Save persists an album to the repository.
// If the album already exists, it is updated; otherwise, it is created.
func (r *AlbumRepository) Save(ctx context.Context, album *gallery.Album) error {
	// Check if album exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsAlbum, album.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check album existence: %w", err)
	}

	if exists {
		return r.update(ctx, album)
	}
	return r.insert(ctx, album)
}

// Delete permanently removes an album from the repository.
// This is a hard delete, not a soft delete.
func (r *AlbumRepository) Delete(ctx context.Context, id gallery.AlbumID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteAlbum, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete album: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrAlbumNotFound
	}

	return nil
}

// ExistsByID checks if an album exists.
func (r *AlbumRepository) ExistsByID(ctx context.Context, id gallery.AlbumID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsAlbum, id.String())
	if err != nil {
		return false, fmt.Errorf("failed to check album existence: %w", err)
	}
	return exists, nil
}

// insert creates a new album in the database.
func (r *AlbumRepository) insert(ctx context.Context, album *gallery.Album) error {
	var coverImageID *string
	if album.CoverImageID() != nil {
		id := album.CoverImageID().String()
		coverImageID = &id
	}

	var parentID *string
	if album.ParentID() != nil {
		id := album.ParentID().String()
		parentID = &id
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertAlbum,
		album.ID().String(),
		album.OwnerID().String(),
		parentID,
		album.Title(),
		album.Description(),
		album.Visibility().String(),
		coverImageID,
		album.ImageCount(),
		album.CreatedAt(),
		album.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert album: %w", err)
	}

	return nil
}

// update updates an existing album in the database.
func (r *AlbumRepository) update(ctx context.Context, album *gallery.Album) error {
	var coverImageID *string
	if album.CoverImageID() != nil {
		id := album.CoverImageID().String()
		coverImageID = &id
	}

	var parentID *string
	if album.ParentID() != nil {
		id := album.ParentID().String()
		parentID = &id
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateAlbum,
		album.ID().String(),
		album.Title(),
		album.Description(),
		album.Visibility().String(),
		coverImageID,
		album.ImageCount(),
		parentID,
		album.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update album: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrAlbumNotFound
	}

	return nil
}

// rowToAlbum converts a database row to a domain Album entity.
func rowToAlbum(row albumRow) (*gallery.Album, error) {
	// Parse IDs
	albumID, err := gallery.ParseAlbumID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}

	ownerID, err := identity.ParseUserID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	// Parse parent ID if present
	var parentID *gallery.AlbumID
	if row.ParentID != nil {
		pID, err := gallery.ParseAlbumID(*row.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent album id: %w", err)
		}
		parentID = &pID
	}

	// Parse cover image ID if present
	var coverImageID *gallery.ImageID
	if row.CoverImageID != nil {
		imageID, err := gallery.ParseImageID(*row.CoverImageID)
		if err != nil {
			return nil, fmt.Errorf("invalid cover image id: %w", err)
		}
		coverImageID = &imageID
	}

	// Parse visibility
	visibility, err := gallery.ParseVisibility(row.Visibility)
	if err != nil {
		return nil, fmt.Errorf("invalid visibility: %w", err)
	}

	// Reconstitute album without validation or events
	album := gallery.ReconstructAlbum(
		albumID,
		ownerID,
		parentID,
		row.Title,
		row.Description,
		visibility,
		coverImageID,
		row.ImageCount,
		row.CreatedAt,
		row.UpdatedAt,
	)

	return album, nil
}

// FindChildren retrieves all direct child albums of a parent album.
func (r *AlbumRepository) FindChildren(ctx context.Context, parentID gallery.AlbumID) ([]*gallery.Album, error) {
	var rows []albumRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectChildAlbums, parentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find child albums: %w", err)
	}

	albums := make([]*gallery.Album, 0, len(rows))
	for _, row := range rows {
		album, err := rowToAlbum(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, nil
}

// FindRootAlbumsByOwner retrieves all root albums (no parent) for a user.
func (r *AlbumRepository) FindRootAlbumsByOwner(ctx context.Context, ownerID identity.UserID) ([]*gallery.Album, error) {
	var rows []albumRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectRootAlbumsByOwner, ownerID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find root albums by owner: %w", err)
	}

	albums := make([]*gallery.Album, 0, len(rows))
	for _, row := range rows {
		album, err := rowToAlbum(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, nil
}

// FindAncestors retrieves all ancestors of an album (breadcrumb path).
// Returns albums from root to the given album (inclusive).
func (r *AlbumRepository) FindAncestors(ctx context.Context, albumID gallery.AlbumID) ([]*gallery.Album, error) {
	var rows []albumRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectAlbumAncestors, albumID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find album ancestors: %w", err)
	}

	if len(rows) == 0 {
		return nil, gallery.ErrAlbumNotFound
	}

	albums := make([]*gallery.Album, 0, len(rows))
	for _, row := range rows {
		album, err := rowToAlbum(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, nil
}
