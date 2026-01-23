package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group album image operations.
const (
	sqlAddImageToAlbum = `
		INSERT INTO group_album_images (
			album_id, image_id, added_by, added_at
		) VALUES (
			$1, $2, $3, NOW()
		)
	`

	sqlRemoveImageFromAlbum = `
		DELETE FROM group_album_images
		WHERE album_id = $1 AND image_id = $2
	`

	sqlIsImageInAlbum = `
		SELECT EXISTS(
			SELECT 1 FROM group_album_images
			WHERE album_id = $1 AND image_id = $2
		)
	`

	sqlFindImagesInAlbum = `
		SELECT i.id, i.owner_id, i.title, i.description, i.width, i.height,
		       i.file_size, i.mime_type, i.storage_provider, i.storage_key,
		       i.original_filename, i.status, i.visibility, i.view_count,
		       i.scan_status,
		       i.created_at, i.updated_at
		FROM group_album_images gai
		INNER JOIN images i ON gai.image_id = i.id
		WHERE gai.album_id = $1 AND i.deleted_at IS NULL
		ORDER BY gai.added_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountGroupAlbumImages = `
		SELECT COUNT(*)
		FROM group_album_images gai
		INNER JOIN images i ON gai.image_id = i.id
		WHERE gai.album_id = $1 AND i.deleted_at IS NULL
	`

	sqlGetImageAddedBy = `
		SELECT added_by FROM group_album_images
		WHERE album_id = $1 AND image_id = $2
	`
)

// groupAlbumImageRow represents a group album image row in the database.
type groupAlbumImageRow struct {
	ID               string    `db:"id"`
	OwnerID          string    `db:"owner_id"`
	Title            string    `db:"title"`
	Description      string    `db:"description"`
	Width            int       `db:"width"`
	Height           int       `db:"height"`
	FileSize         int64     `db:"file_size"`
	MimeType         string    `db:"mime_type"`
	StorageProvider  string    `db:"storage_provider"`
	StorageKey       string    `db:"storage_key"`
	OriginalFilename string    `db:"original_filename"`
	Status           string    `db:"status"`
	Visibility       string    `db:"visibility"`
	ScanStatus       string    `db:"scan_status"`
	ViewCount        int64     `db:"view_count"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// GroupAlbumImageRepository implements the community.GroupAlbumImageRepository interface for PostgreSQL.
type GroupAlbumImageRepository struct {
	db *sqlx.DB
}

// NewGroupAlbumImageRepository creates a new GroupAlbumImageRepository with the given database connection.
func NewGroupAlbumImageRepository(db *sqlx.DB) *GroupAlbumImageRepository {
	return &GroupAlbumImageRepository{db: db}
}

// AddImageToAlbum adds an image to a group album.
// The addedBy parameter tracks who added the image.
// Returns an error if the image is already in the album.
func (r *GroupAlbumImageRepository) AddImageToAlbum(
	ctx context.Context,
	albumID community.GroupAlbumID,
	imageID gallery.ImageID,
	addedBy identity.UserID,
) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlAddImageToAlbum,
		albumID.String(),
		imageID.String(),
		addedBy.String(),
	)
	if err != nil {
		// Check for unique constraint violation (duplicate entry)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("image already in album: %w", community.ErrImageAlreadyInAlbum)
		}
		return fmt.Errorf("failed to add image to album: %w", err)
	}

	return nil
}

// RemoveImageFromAlbum removes an image from a group album.
// Returns no error if the image wasn't in the album (idempotent).
func (r *GroupAlbumImageRepository) RemoveImageFromAlbum(
	ctx context.Context,
	albumID community.GroupAlbumID,
	imageID gallery.ImageID,
) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlRemoveImageFromAlbum,
		albumID.String(),
		imageID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to remove image from album: %w", err)
	}

	// Idempotent - no error if image wasn't in album
	return nil
}

// IsImageInAlbum checks if an image is in a group album.
func (r *GroupAlbumImageRepository) IsImageInAlbum(
	ctx context.Context,
	albumID community.GroupAlbumID,
	imageID gallery.ImageID,
) (bool, error) {
	var exists bool
	err := r.db.GetContext(
		ctx,
		&exists,
		sqlIsImageInAlbum,
		albumID.String(),
		imageID.String(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to check if image is in album: %w", err)
	}

	return exists, nil
}

// FindImagesInAlbum retrieves all images in a group album with pagination.
// Returns the images, total count, and any error.
func (r *GroupAlbumImageRepository) FindImagesInAlbum(
	ctx context.Context,
	albumID community.GroupAlbumID,
	pagination shared.Pagination,
) ([]*gallery.Image, int, error) {
	// Execute main query
	var rows []groupAlbumImageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlFindImagesInAlbum,
		albumID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find images in album: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountGroupAlbumImages, albumID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images in album: %w", err)
	}

	// Convert rows to domain entities
	images := make([]*gallery.Image, 0, len(rows))
	for _, row := range rows {
		image, err := rowToImageFromAlbum(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to image: %w", err)
		}
		images = append(images, image)
	}

	return images, total, nil
}

// GetImageAddedBy returns the user who added a specific image to an album.
// Returns ErrGroupImageNotFound if the association doesn't exist.
func (r *GroupAlbumImageRepository) GetImageAddedBy(
	ctx context.Context,
	albumID community.GroupAlbumID,
	imageID gallery.ImageID,
) (identity.UserID, error) {
	var addedByStr string
	err := r.db.GetContext(
		ctx,
		&addedByStr,
		sqlGetImageAddedBy,
		albumID.String(),
		imageID.String(),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return identity.UserID{}, community.ErrGroupImageNotFound
		}
		return identity.UserID{}, fmt.Errorf("failed to get image added by: %w", err)
	}

	addedBy, err := identity.ParseUserID(addedByStr)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("invalid user id: %w", err)
	}

	return addedBy, nil
}

// rowToImageFromAlbum converts a database row to an Image domain entity.
// This function is used specifically for images retrieved from album queries.
func rowToImageFromAlbum(row groupAlbumImageRow) (*gallery.Image, error) {
	// Parse IDs
	imageID, err := gallery.ParseImageID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	ownerID, err := identity.ParseUserID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	// Parse visibility
	visibility, err := gallery.ParseVisibility(row.Visibility)
	if err != nil {
		return nil, fmt.Errorf("invalid visibility: %w", err)
	}

	// Parse status
	status, err := gallery.ParseImageStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Parse scan status
	scanStatus, err := gallery.ParseScanStatus(row.ScanStatus)
	if err != nil {
		if row.ScanStatus == "" {
			scanStatus = gallery.ScanStatusPending
		} else {
			return nil, fmt.Errorf("invalid scan status: %w", err)
		}
	}

	// Create image metadata using constructor
	metadata, err := gallery.NewImageMetadata(
		row.Title,
		row.Description,
		row.OriginalFilename,
		row.MimeType,
		row.Width,
		row.Height,
		row.FileSize,
		row.StorageKey,
		row.StorageProvider,
	)
	if err != nil {
		return nil, fmt.Errorf("create image metadata: %w", err)
	}

	// Reconstruct image without validation or events
	// Note: variants and tags are not loaded in this query for performance
	// IPFS metadata is nil since it's not needed for album listing
	image := gallery.ReconstructImage(
		imageID,
		ownerID,
		metadata,
		visibility,
		status,
		scanStatus,
		[]gallery.ImageVariant{}, // Empty variants - not needed for album listing
		[]gallery.Tag{},          // Empty tags - not needed for album listing
		nil,                      // No IPFS metadata for album listing
		row.ViewCount,
		0, // Like count - would need to join with likes table
		0, // Comment count - would need to join with comments table
		row.CreatedAt,
		row.UpdatedAt,
	)

	return image, nil
}
