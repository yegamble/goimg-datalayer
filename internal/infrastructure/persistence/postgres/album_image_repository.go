package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for album-image relationship operations.
const (
	sqlInsertAlbumImage = `
		INSERT INTO album_images (album_id, image_id, position, added_at)
		VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM album_images WHERE album_id = $1), 0), NOW())
	`

	sqlDeleteAlbumImage = `
		DELETE FROM album_images
		WHERE album_id = $1 AND image_id = $2
	`

	sqlSelectImagesInAlbum = `
		SELECT i.id, i.owner_id, i.title, i.description, i.storage_provider, i.storage_key,
		       i.original_filename, i.mime_type, i.file_size, i.width, i.height,
		       i.status, i.visibility, i.scan_status, i.view_count,
		       i.created_at, i.updated_at
		FROM images i
		INNER JOIN album_images ai ON i.id = ai.image_id
		WHERE ai.album_id = $1 AND i.deleted_at IS NULL
		ORDER BY ai.position, ai.added_at
		LIMIT $2 OFFSET $3
	`

	sqlCountImagesInAlbum = `
		SELECT COUNT(*)
		FROM album_images ai
		INNER JOIN images i ON ai.image_id = i.id
		WHERE ai.album_id = $1 AND i.deleted_at IS NULL
	`

	sqlSelectAlbumsForImage = `
		SELECT a.id, a.owner_id, a.title, a.description, a.visibility, a.cover_image_id,
		       a.image_count, a.created_at, a.updated_at
		FROM albums a
		INNER JOIN album_images ai ON a.id = ai.album_id
		WHERE ai.image_id = $1 AND a.deleted_at IS NULL
		ORDER BY ai.added_at DESC
	`

	sqlCheckImageInAlbum = `
		SELECT EXISTS(
			SELECT 1
			FROM album_images
			WHERE album_id = $1 AND image_id = $2
		)
	`
)

// AlbumImageRepository implements the gallery.AlbumImageRepository interface.
// It manages the many-to-many relationship between albums and images.
type AlbumImageRepository struct {
	db *sqlx.DB
}

// NewAlbumImageRepository creates a new AlbumImageRepository.
func NewAlbumImageRepository(db *sqlx.DB) *AlbumImageRepository {
	return &AlbumImageRepository{db: db}
}

// AddImageToAlbum adds an image to an album.
// Returns an error if the image is already in the album.
func (r *AlbumImageRepository) AddImageToAlbum(
	ctx context.Context,
	albumID gallery.AlbumID,
	imageID gallery.ImageID,
) error {
	_, err := r.db.ExecContext(ctx, sqlInsertAlbumImage, albumID.String(), imageID.String())
	if err != nil {
		// Check for unique constraint violation (image already in album)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("image already in album")
			}
		}
		return fmt.Errorf("failed to add image to album: %w", err)
	}

	return nil
}

// RemoveImageFromAlbum removes an image from an album.
// Returns no error if the image wasn't in the album (idempotent).
func (r *AlbumImageRepository) RemoveImageFromAlbum(
	ctx context.Context,
	albumID gallery.AlbumID,
	imageID gallery.ImageID,
) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteAlbumImage, albumID.String(), imageID.String())
	if err != nil {
		return fmt.Errorf("failed to remove image from album: %w", err)
	}

	// Check if any rows were affected (idempotent - no error if not found)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log if no rows were deleted, but don't error (idempotent operation)
	_ = rowsAffected

	return nil
}

// FindImagesInAlbum retrieves all images in an album with pagination.
// Returns the images, total count, and any error.
func (r *AlbumImageRepository) FindImagesInAlbum(
	ctx context.Context,
	albumID gallery.AlbumID,
	pagination shared.Pagination,
) ([]*gallery.Image, int64, error) {
	// Get paginated images
	var rows []imageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectImagesInAlbum,
		albumID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find images in album: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountImagesInAlbum, albumID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images in album: %w", err)
	}

	// Convert rows to domain entities
	images := make([]*gallery.Image, 0, len(rows))
	for _, row := range rows {
		// Note: This query doesn't fetch variants and tags, so we pass nil slices
		image, err := rowToImage(row, nil, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to image: %w", err)
		}
		images = append(images, image)
	}

	return images, total, nil
}

// FindAlbumsForImage retrieves all albums containing an image.
// Used to show which albums an image is in.
func (r *AlbumImageRepository) FindAlbumsForImage(
	ctx context.Context,
	imageID gallery.ImageID,
) ([]*gallery.Album, error) {
	var rows []albumRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectAlbumsForImage, imageID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find albums for image: %w", err)
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

// IsImageInAlbum checks if an image is in an album.
func (r *AlbumImageRepository) IsImageInAlbum(
	ctx context.Context,
	albumID gallery.AlbumID,
	imageID gallery.ImageID,
) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckImageInAlbum, albumID.String(), imageID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if image is in album: %w", err)
	}

	return exists, nil
}

// CountImagesInAlbum returns the number of images in an album.
// Used to update the image count on the album entity.
func (r *AlbumImageRepository) CountImagesInAlbum(
	ctx context.Context,
	albumID gallery.AlbumID,
) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountImagesInAlbum, albumID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count images in album: %w", err)
	}

	return count, nil
}
