package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// batchLoadVariants loads variants for multiple images in a single query.
// This eliminates the N+1 query problem when loading image lists.
//
// Example: Loading 50 images would normally require 50 separate queries.
// This function reduces it to 1 query by using WHERE image_id = ANY($1).
//
// Performance impact: 50x reduction in database round trips.
func batchLoadVariants(
	ctx context.Context, db *sqlx.DB, imageIDs []gallery.ImageID,
) (map[string][]gallery.ImageVariant, error) {
	if len(imageIDs) == 0 {
		return make(map[string][]gallery.ImageVariant), nil
	}

	// Convert ImageIDs to string array for PostgreSQL ANY query
	idStrings := make([]string, len(imageIDs))
	for i, id := range imageIDs {
		idStrings[i] = id.String()
	}

	// Single query to fetch all variants for all images
	query := `
		SELECT id, image_id, variant_type, storage_key, width, height, file_size, format, created_at
		FROM image_variants
		WHERE image_id = ANY($1)
		ORDER BY image_id, variant_type
	`

	var rows []variantRow
	err := db.SelectContext(ctx, &rows, query, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("batch load variants: %w", err)
	}

	// Group variants by image_id
	variantsByImageID := make(map[string][]gallery.ImageVariant)
	for _, row := range rows {
		variantType, err := gallery.ParseVariantType(row.VariantType)
		if err != nil {
			return nil, fmt.Errorf("parse variant type: %w", err)
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
			return nil, fmt.Errorf("create variant: %w", err)
		}

		variantsByImageID[row.ImageID] = append(variantsByImageID[row.ImageID], variant)
	}

	return variantsByImageID, nil
}

// batchLoadTags loads tags for multiple images in a single query.
// This eliminates the N+1 query problem when loading image lists.
//
// Example: Loading 50 images would normally require 50 separate queries.
// This function reduces it to 1 query using a JOIN with ANY filter.
//
// Performance impact: 50x reduction in database round trips.
func batchLoadTags(ctx context.Context, db *sqlx.DB, imageIDs []gallery.ImageID) (map[string][]gallery.Tag, error) {
	if len(imageIDs) == 0 {
		return make(map[string][]gallery.Tag), nil
	}

	// Convert ImageIDs to string array for PostgreSQL ANY query
	idStrings := make([]string, len(imageIDs))
	for i, id := range imageIDs {
		idStrings[i] = id.String()
	}

	// Single query to fetch all tags for all images
	query := `
		SELECT it.image_id, t.name, t.slug
		FROM image_tags it
		INNER JOIN tags t ON it.tag_id = t.id
		WHERE it.image_id = ANY($1)
		ORDER BY it.image_id, t.name
	`

	type tagWithImageID struct {
		ImageID string `db:"image_id"`
		Name    string `db:"name"`
		Slug    string `db:"slug"`
	}

	var rows []tagWithImageID
	err := db.SelectContext(ctx, &rows, query, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("batch load tags: %w", err)
	}

	// Group tags by image_id
	tagsByImageID := make(map[string][]gallery.Tag)
	for _, row := range rows {
		tag, err := gallery.NewTag(row.Name)
		if err != nil {
			return nil, fmt.Errorf("create tag: %w", err)
		}
		tagsByImageID[row.ImageID] = append(tagsByImageID[row.ImageID], tag)
	}

	return tagsByImageID, nil
}

// rowsToImagesWithBatchLoading converts multiple image rows to domain entities
// using batch loading for variants and tags to avoid N+1 queries.
//
// This is the optimized version of rowsToImages that should be used for all
// list queries (FindByOwner, FindPublic, FindByTag, FindByStatus).
//
// Performance comparison:
//   - Old approach: 1 + N + N = 101 queries for 50 images
//   - New approach: 1 + 1 + 1 = 3 queries for 50 images
//   - Improvement: 33x fewer queries
func rowsToImagesWithBatchLoading(ctx context.Context, db *sqlx.DB, rows []imageRow) ([]*gallery.Image, error) {
	if len(rows) == 0 {
		return []*gallery.Image{}, nil
	}

	// Extract all image IDs
	imageIDs := make([]gallery.ImageID, len(rows))
	for i, row := range rows {
		imageID, err := gallery.ParseImageID(row.ID)
		if err != nil {
			return nil, fmt.Errorf("parse image id %s: %w", row.ID, err)
		}
		imageIDs[i] = imageID
	}

	// Batch load variants and tags for all images (2 queries total instead of 2N)
	variantsByImageID, err := batchLoadVariants(ctx, db, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("batch load variants: %w", err)
	}

	tagsByImageID, err := batchLoadTags(ctx, db, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("batch load tags: %w", err)
	}

	// Convert rows to domain entities with pre-loaded variants and tags
	images := make([]*gallery.Image, 0, len(rows))
	for _, row := range rows {
		variants := variantsByImageID[row.ID]
		tags := tagsByImageID[row.ID]

		image, err := rowToImage(row, variants, tags)
		if err != nil {
			return nil, fmt.Errorf("convert row to image: %w", err)
		}

		images = append(images, image)
	}

	return images, nil
}
