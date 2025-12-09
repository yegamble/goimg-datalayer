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

// SQL queries for image operations.
const (
	// Query ordering constants.
	orderByCreatedDesc = " ORDER BY i.created_at DESC"

	sqlInsertImage = `
		INSERT INTO images (
			id, owner_id, title, description, storage_provider, storage_key,
			original_filename, mime_type, file_size, width, height,
			status, visibility, scan_status, view_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	sqlUpdateImage = `
		UPDATE images
		SET title = $2,
		    description = $3,
		    status = $4,
		    visibility = $5,
		    view_count = $6,
		    updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectImageByID = `
		SELECT id, owner_id, title, description, storage_provider, storage_key,
		       original_filename, mime_type, file_size, width, height,
		       status, visibility, scan_status, view_count,
		       created_at, updated_at
		FROM images
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectImagesByOwner = `
		SELECT id, owner_id, title, description, storage_provider, storage_key,
		       original_filename, mime_type, file_size, width, height,
		       status, visibility, scan_status, view_count,
		       created_at, updated_at
		FROM images
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountImagesByOwner = `
		SELECT COUNT(*)
		FROM images
		WHERE owner_id = $1 AND deleted_at IS NULL
	`

	sqlSelectPublicImages = `
		SELECT id, owner_id, title, description, storage_provider, storage_key,
		       original_filename, mime_type, file_size, width, height,
		       status, visibility, scan_status, view_count,
		       created_at, updated_at
		FROM images
		WHERE status = 'active' AND visibility = 'public' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	sqlCountPublicImages = `
		SELECT COUNT(*)
		FROM images
		WHERE status = 'active' AND visibility = 'public' AND deleted_at IS NULL
	`

	sqlSelectImagesByTag = `
		SELECT i.id, i.owner_id, i.title, i.description, i.storage_provider, i.storage_key,
		       i.original_filename, i.mime_type, i.file_size, i.width, i.height,
		       i.status, i.visibility, i.scan_status, i.view_count,
		       i.created_at, i.updated_at
		FROM images i
		INNER JOIN image_tags it ON i.id = it.image_id
		INNER JOIN tags t ON it.tag_id = t.id
		WHERE t.slug = $1 AND i.status = 'active' AND i.visibility = 'public' AND i.deleted_at IS NULL
		ORDER BY i.created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountImagesByTag = `
		SELECT COUNT(*)
		FROM images i
		INNER JOIN image_tags it ON i.id = it.image_id
		INNER JOIN tags t ON it.tag_id = t.id
		WHERE t.slug = $1 AND i.status = 'active' AND i.visibility = 'public' AND i.deleted_at IS NULL
	`

	sqlSelectImagesByStatus = `
		SELECT id, owner_id, title, description, storage_provider, storage_key,
		       original_filename, mime_type, file_size, width, height,
		       status, visibility, scan_status, view_count,
		       created_at, updated_at
		FROM images
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountImagesByStatus = `
		SELECT COUNT(*)
		FROM images
		WHERE status = $1 AND deleted_at IS NULL
	`

	sqlDeleteImage = `
		DELETE FROM images WHERE id = $1
	`

	sqlExistsImage = `
		SELECT EXISTS(SELECT 1 FROM images WHERE id = $1 AND deleted_at IS NULL)
	`

	// Full-text search query with dynamic filtering
	// PERFORMANCE NOTE: Uses pre-computed search_vector column with GIN index
	// instead of calculating to_tsvector at query time (10-50x faster).
	// See migration 00005_add_performance_indexes.sql.
	sqlSearchImagesBase = `
		SELECT DISTINCT i.id, i.owner_id, i.title, i.description, i.storage_provider, i.storage_key,
		       i.original_filename, i.mime_type, i.file_size, i.width, i.height,
		       i.status, i.visibility, i.scan_status, i.view_count,
		       i.created_at, i.updated_at,
		       ts_rank(i.search_vector, plainto_tsquery('english', $1)) AS relevance_score
		FROM images i
	`

	sqlInsertVariant = `
		INSERT INTO image_variants (
			id, image_id, variant_type, storage_key, width, height, file_size, format, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	sqlSelectVariantsByImageID = `
		SELECT id, image_id, variant_type, storage_key, width, height, file_size, format, created_at
		FROM image_variants
		WHERE image_id = $1
		ORDER BY variant_type
	`

	sqlDeleteVariantsByImageID = `
		DELETE FROM image_variants WHERE image_id = $1
	`

	sqlInsertTag = `
		INSERT INTO tags (id, name, slug, usage_count, created_at)
		VALUES ($1, $2, $3, 0, $4)
		ON CONFLICT (slug) DO NOTHING
	`

	sqlGetTagIDBySlug = `
		SELECT id FROM tags WHERE slug = $1
	`

	sqlInsertImageTag = `
		INSERT INTO image_tags (image_id, tag_id, tagged_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (image_id, tag_id) DO NOTHING
	`

	sqlSelectTagsByImageID = `
		SELECT t.name, t.slug
		FROM tags t
		INNER JOIN image_tags it ON t.id = it.tag_id
		WHERE it.image_id = $1
		ORDER BY t.name
	`

	sqlDeleteImageTags = `
		DELETE FROM image_tags WHERE image_id = $1
	`
)

// imageRow represents an image row in the database.
type imageRow struct {
	ID               string    `db:"id"`
	OwnerID          string    `db:"owner_id"`
	Title            string    `db:"title"`
	Description      string    `db:"description"`
	StorageProvider  string    `db:"storage_provider"`
	StorageKey       string    `db:"storage_key"`
	OriginalFilename string    `db:"original_filename"`
	MimeType         string    `db:"mime_type"`
	FileSize         int64     `db:"file_size"`
	Width            int       `db:"width"`
	Height           int       `db:"height"`
	Status           string    `db:"status"`
	Visibility       string    `db:"visibility"`
	ScanStatus       string    `db:"scan_status"`
	ViewCount        int64     `db:"view_count"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// variantRow represents an image variant row in the database.
type variantRow struct {
	ID          string    `db:"id"`
	ImageID     string    `db:"image_id"`
	VariantType string    `db:"variant_type"`
	StorageKey  string    `db:"storage_key"`
	Width       int       `db:"width"`
	Height      int       `db:"height"`
	FileSize    int64     `db:"file_size"`
	Format      string    `db:"format"`
	CreatedAt   time.Time `db:"created_at"`
}

// tagRow represents a tag row in the database.
type tagRow struct {
	Name string `db:"name"`
	Slug string `db:"slug"`
}

// ImageRepository implements the gallery.ImageRepository interface for PostgreSQL.
type ImageRepository struct {
	db *sqlx.DB
}

// NewImageRepository creates a new ImageRepository with the given database connection.
func NewImageRepository(db *sqlx.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// NextID generates the next available ImageID.
func (r *ImageRepository) NextID() gallery.ImageID {
	return gallery.NewImageID()
}

// FindByID retrieves an image by its unique ID.
func (r *ImageRepository) FindByID(ctx context.Context, id gallery.ImageID) (*gallery.Image, error) {
	var row imageRow
	if err := r.db.GetContext(ctx, &row, sqlSelectImageByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to find image by id: %w", err)
	}

	// Load variants
	variants, err := r.loadVariants(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load variants: %w", err)
	}

	// Load tags
	tags, err := r.loadTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	image, err := rowToImage(row, variants, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to image: %w", err)
	}

	return image, nil
}

// FindByOwner retrieves all images owned by a user with pagination.
func (r *ImageRepository) FindByOwner(
	ctx context.Context,
	ownerID identity.UserID,
	pagination shared.Pagination,
) ([]*gallery.Image, int64, error) {
	// Get paginated images
	var rows []imageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectImagesByOwner,
		ownerID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find images by owner: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountImagesByOwner, ownerID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images by owner: %w", err)
	}

	// Convert rows to domain entities
	images, err := r.rowsToImages(ctx, rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to images: %w", err)
	}

	return images, total, nil
}

// FindPublic retrieves all public images with pagination.
func (r *ImageRepository) FindPublic(
	ctx context.Context,
	pagination shared.Pagination,
) ([]*gallery.Image, int64, error) {
	// Get paginated images
	var rows []imageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectPublicImages,
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find public images: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountPublicImages)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count public images: %w", err)
	}

	// Convert rows to domain entities
	images, err := r.rowsToImages(ctx, rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to images: %w", err)
	}

	return images, total, nil
}

// FindByTag retrieves all public images with a specific tag, with pagination.
func (r *ImageRepository) FindByTag(
	ctx context.Context,
	tag gallery.Tag,
	pagination shared.Pagination,
) ([]*gallery.Image, int64, error) {
	// Get paginated images
	var rows []imageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectImagesByTag,
		tag.Slug(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find images by tag: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountImagesByTag, tag.Slug())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images by tag: %w", err)
	}

	// Convert rows to domain entities
	images, err := r.rowsToImages(ctx, rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to images: %w", err)
	}

	return images, total, nil
}

// FindByStatus retrieves images by status with pagination.
func (r *ImageRepository) FindByStatus(
	ctx context.Context,
	status gallery.ImageStatus,
	pagination shared.Pagination,
) ([]*gallery.Image, int64, error) {
	// Get paginated images
	var rows []imageRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectImagesByStatus,
		status.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find images by status: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountImagesByStatus, status.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images by status: %w", err)
	}

	// Convert rows to domain entities
	images, err := r.rowsToImages(ctx, rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to images: %w", err)
	}

	return images, total, nil
}

// Save persists an image to the repository.
// If the image already exists, it is updated; otherwise, it is created.
// This operation includes saving variants and tags in a transaction.
func (r *ImageRepository) Save(ctx context.Context, image *gallery.Image) error {
	// Check if image exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsImage, image.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check image existence: %w", err)
	}

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if exists {
		err = r.updateInTx(ctx, tx, image)
	} else {
		err = r.insertInTx(ctx, tx, image)
	}
	if err != nil {
		return err
	}

	// Save variants (delete old ones and insert new ones)
	if err := r.saveVariantsInTx(ctx, tx, image); err != nil {
		return fmt.Errorf("failed to save variants: %w", err)
	}

	// Save tags (delete old ones and insert new ones)
	if err := r.saveTagsInTx(ctx, tx, image); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete permanently removes an image from the repository.
// This is a hard delete, not a soft delete.
func (r *ImageRepository) Delete(ctx context.Context, id gallery.ImageID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteImage, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrImageNotFound
	}

	return nil
}

// ExistsByID checks if an image exists.
func (r *ImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsImage, id.String())
	if err != nil {
		return false, fmt.Errorf("failed to check image existence: %w", err)
	}
	return exists, nil
}

// Search performs a full-text search on images with filters and sorting.
// Uses PostgreSQL's ts_vector and ts_rank for relevance scoring.
func (r *ImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	// Build dynamic query based on search parameters
	query, countQuery, args := r.buildSearchQuery(params)

	// Execute search query
	type searchRow struct {
		imageRow
		RelevanceScore float64 `db:"relevance_score"`
	}

	var rows []searchRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search images: %w", err)
	}

	// Execute count query
	var total int64
	err = r.db.GetContext(ctx, &total, countQuery, args[:len(args)-2]...) // Exclude LIMIT and OFFSET
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Convert rows to domain entities
	images := make([]*gallery.Image, 0, len(rows))
	for _, row := range rows {
		// Note: Search query doesn't fetch variants and tags, so we pass nil slices
		image, err := rowToImage(row.imageRow, nil, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to image: %w", err)
		}
		images = append(images, image)
	}

	return images, total, nil
}

// buildSearchQuery constructs a dynamic SQL query based on search parameters.
//
//nolint:cyclop // Query builder requires conditional logic for multiple search filters: query, tags, visibility, owner, and sort
func (r *ImageRepository) buildSearchQuery(params gallery.SearchParams) (string, string, []interface{}) {
	query := sqlSearchImagesBase
	countQuery := "SELECT COUNT(DISTINCT i.id) FROM images i"
	args := []interface{}{params.Query}
	paramIndex := 2

	// Build WHERE conditions
	conditions := []string{"i.deleted_at IS NULL", "i.status = 'active'"}

	// Full-text search condition (only if query is not empty)
	// Uses pre-computed search_vector column for 10-50x performance improvement
	if params.Query != "" {
		conditions = append(conditions,
			"i.search_vector @@ plainto_tsquery('english', $1)",
		)
	}

	// Filter by visibility
	if params.Visibility != nil {
		conditions = append(conditions, fmt.Sprintf("i.visibility = $%d", paramIndex))
		args = append(args, params.Visibility.String())
		paramIndex++
	} else {
		// Default to public only if no visibility specified
		conditions = append(conditions, "i.visibility = 'public'")
	}

	// Filter by owner
	if params.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("i.owner_id = $%d", paramIndex))
		args = append(args, params.OwnerID.String())
		paramIndex++
	}

	// Filter by tags (AND logic - image must have ALL specified tags)
	if len(params.Tags) > 0 {
		query += " INNER JOIN image_tags it ON i.id = it.image_id INNER JOIN tags t ON it.tag_id = t.id"
		countQuery += " INNER JOIN image_tags it ON i.id = it.image_id INNER JOIN tags t ON it.tag_id = t.id"

		tagSlugs := make([]string, len(params.Tags))
		for i, tag := range params.Tags {
			tagSlugs[i] = tag.Slug()
		}

		conditions = append(conditions, fmt.Sprintf("t.slug = ANY($%d)", paramIndex))
		args = append(args, tagSlugs)
		paramIndex++

		// For AND logic: group by image and count matching tags
		query = fmt.Sprintf("%s WHERE %s GROUP BY i.id HAVING COUNT(DISTINCT t.id) = %d",
			query, joinConditions(conditions), len(params.Tags))
		countQuery = fmt.Sprintf("%s WHERE %s GROUP BY i.id HAVING COUNT(DISTINCT t.id) = %d",
			countQuery, joinConditions(conditions), len(params.Tags))
	} else {
		query += " WHERE " + joinConditions(conditions)
		countQuery += " WHERE " + joinConditions(conditions)
	}

	// Add ORDER BY based on sort parameter
	switch params.SortBy {
	case gallery.SearchSortByRelevance:
		if params.Query != "" {
			query += " ORDER BY relevance_score DESC, i.created_at DESC"
		} else {
			query += orderByCreatedDesc
		}
	case gallery.SearchSortByCreatedAt:
		query += orderByCreatedDesc
	case gallery.SearchSortByViewCount:
		query += " ORDER BY i.view_count DESC, i.created_at DESC"
	case gallery.SearchSortByLikeCount:
		query += " ORDER BY i.view_count DESC, i.created_at DESC" // Using view_count as proxy for now
	default:
		query += orderByCreatedDesc
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	args = append(args, params.Pagination.Limit(), params.Pagination.Offset())

	return query, countQuery, args
}

// joinConditions joins SQL WHERE conditions with AND.
func joinConditions(conditions []string) string {
	result := ""
	for i, cond := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += cond
	}
	return result
}

// insertInTx creates a new image in the database within a transaction.
func (r *ImageRepository) insertInTx(ctx context.Context, tx *sqlx.Tx, image *gallery.Image) error {
	metadata := image.Metadata()
	_, err := tx.ExecContext(
		ctx,
		sqlInsertImage,
		image.ID().String(),
		image.OwnerID().String(),
		metadata.Title(),
		metadata.Description(),
		metadata.StorageProvider(),
		metadata.StorageKey(),
		metadata.OriginalFilename(),
		metadata.MimeType(),
		metadata.FileSize(),
		metadata.Width(),
		metadata.Height(),
		image.Status().String(),
		image.Visibility().String(),
		"pending", // Default scan status
		image.ViewCount(),
		image.CreatedAt(),
		image.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert image: %w", err)
	}

	return nil
}

// updateInTx updates an existing image in the database within a transaction.
func (r *ImageRepository) updateInTx(ctx context.Context, tx *sqlx.Tx, image *gallery.Image) error {
	metadata := image.Metadata()
	result, err := tx.ExecContext(
		ctx,
		sqlUpdateImage,
		image.ID().String(),
		metadata.Title(),
		metadata.Description(),
		image.Status().String(),
		image.Visibility().String(),
		image.ViewCount(),
		image.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrImageNotFound
	}

	return nil
}

// saveVariantsInTx saves image variants within a transaction.
func (r *ImageRepository) saveVariantsInTx(ctx context.Context, tx *sqlx.Tx, image *gallery.Image) error {
	// Delete existing variants
	_, err := tx.ExecContext(ctx, sqlDeleteVariantsByImageID, image.ID().String())
	if err != nil {
		return fmt.Errorf("failed to delete existing variants: %w", err)
	}

	// Insert new variants
	for _, variant := range image.Variants() {
		_, err := tx.ExecContext(
			ctx,
			sqlInsertVariant,
			uuid.New().String(),
			image.ID().String(),
			variant.VariantType().String(),
			variant.StorageKey(),
			variant.Width(),
			variant.Height(),
			variant.FileSize(),
			variant.Format(),
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert variant: %w", err)
		}
	}

	return nil
}

// saveTagsInTx saves image tags within a transaction.
func (r *ImageRepository) saveTagsInTx(ctx context.Context, tx *sqlx.Tx, image *gallery.Image) error {
	// Delete existing image-tag associations
	_, err := tx.ExecContext(ctx, sqlDeleteImageTags, image.ID().String())
	if err != nil {
		return fmt.Errorf("failed to delete existing image tags: %w", err)
	}

	// Insert tags and create associations
	now := time.Now().UTC()
	for _, tag := range image.Tags() {
		// Insert tag if it doesn't exist (upsert)
		_, err := tx.ExecContext(
			ctx,
			sqlInsertTag,
			uuid.New().String(),
			tag.Name(),
			tag.Slug(),
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert tag: %w", err)
		}

		// Get tag ID
		var tagID string
		err = tx.GetContext(ctx, &tagID, sqlGetTagIDBySlug, tag.Slug())
		if err != nil {
			return fmt.Errorf("failed to get tag id: %w", err)
		}

		// Create image-tag association
		_, err = tx.ExecContext(
			ctx,
			sqlInsertImageTag,
			image.ID().String(),
			tagID,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert image tag: %w", err)
		}
	}

	return nil
}

// loadVariants loads all variants for an image.
func (r *ImageRepository) loadVariants(ctx context.Context, imageID gallery.ImageID) ([]gallery.ImageVariant, error) {
	var rows []variantRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectVariantsByImageID, imageID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to load variants: %w", err)
	}

	variants := make([]gallery.ImageVariant, 0, len(rows))
	for _, row := range rows {
		variantType, err := gallery.ParseVariantType(row.VariantType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse variant type: %w", err)
		}

		variant, err := gallery.NewImageVariant(
			variantType,
			row.StorageKey,
			row.Width,
			row.Height,
			row.FileSize,
			row.Format,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create variant: %w", err)
		}

		variants = append(variants, variant)
	}

	return variants, nil
}

// loadTags loads all tags for an image.
func (r *ImageRepository) loadTags(ctx context.Context, imageID gallery.ImageID) ([]gallery.Tag, error) {
	var rows []tagRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectTagsByImageID, imageID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	tags := make([]gallery.Tag, 0, len(rows))
	for _, row := range rows {
		tag, err := gallery.NewTag(row.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// rowsToImages converts multiple image rows to domain entities using batch loading.
// This method uses batch loading to avoid N+1 query problems.
// For 50 images, this reduces queries from 101 (1 + 50 + 50) to 3 (1 + 1 + 1).
//
// PERFORMANCE NOTE: This is a critical optimization for list queries.
// See docs/performance-analysis-sprint8.md for detailed analysis.
func (r *ImageRepository) rowsToImages(ctx context.Context, rows []imageRow) ([]*gallery.Image, error) {
	// Use batch loading helper to eliminate N+1 queries
	return rowsToImagesWithBatchLoading(ctx, r.db, rows)
}

// rowToImage converts a database row to a domain Image entity.
func rowToImage(row imageRow, variants []gallery.ImageVariant, tags []gallery.Tag) (*gallery.Image, error) {
	// Parse IDs
	imageID, err := gallery.ParseImageID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	ownerID, err := identity.ParseUserID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	// Parse enums
	status, err := gallery.ParseImageStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	visibility, err := gallery.ParseVisibility(row.Visibility)
	if err != nil {
		return nil, fmt.Errorf("invalid visibility: %w", err)
	}

	// Create metadata
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
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// Reconstitute image without validation or events
	image := gallery.ReconstructImage(
		imageID,
		ownerID,
		metadata,
		visibility,
		status,
		variants,
		tags,
		row.ViewCount,
		0, // likeCount - not stored in images table yet
		0, // commentCount - not stored in images table yet
		row.CreatedAt,
		row.UpdatedAt,
	)

	return image, nil
}
