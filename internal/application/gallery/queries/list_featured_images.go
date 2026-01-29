package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

const (
	defaultFeaturedLimit = 10
	maxFeaturedLimit     = 50
)

// ListFeaturedImagesQuery retrieves currently active featured images.
type ListFeaturedImagesQuery struct {
	// Limit is the maximum number of featured images to return (1-50, default 10).
	Limit int
}

// FeaturedImageDTO represents a featured image with metadata for API responses.
type FeaturedImageDTO struct {
	PickID        string             `json:"pick_id"`
	ImageID       string             `json:"image_id"`
	Image         *FeaturedImageInfo `json:"image"`
	DisplayOrder  int                `json:"display_order"`
	FeaturedSince string             `json:"featured_since"` // ISO 8601
	Reason        string             `json:"reason,omitempty"`
}

// FeaturedImageInfo represents a minimal image response for featured picks.
type FeaturedImageInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"owner_id"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
}

// ListFeaturedImagesResult represents the result of a featured images query.
type ListFeaturedImagesResult struct {
	FeaturedImages []FeaturedImageDTO `json:"featured_images"`
	TotalCount     int                `json:"total_count"`
}

// ListFeaturedImagesHandler processes ListFeaturedImagesQuery requests.
type ListFeaturedImagesHandler struct {
	picks  gallery.FeaturedPickRepository
	images gallery.ImageRepository
	logger zerolog.Logger
}

// NewListFeaturedImagesHandler creates a new ListFeaturedImagesHandler.
func NewListFeaturedImagesHandler(
	picks gallery.FeaturedPickRepository,
	images gallery.ImageRepository,
	logger zerolog.Logger,
) *ListFeaturedImagesHandler {
	return &ListFeaturedImagesHandler{
		picks:  picks,
		images: images,
		logger: logger,
	}
}

// Handle executes the ListFeaturedImagesQuery and returns featured images.
func (h *ListFeaturedImagesHandler) Handle(ctx context.Context, query ListFeaturedImagesQuery) (*ListFeaturedImagesResult, error) {
	// Validate and normalize limit
	limit := query.Limit
	if limit < 1 {
		limit = defaultFeaturedLimit
	}
	if limit > maxFeaturedLimit {
		limit = maxFeaturedLimit
	}

	h.logger.Debug().
		Int("limit", limit).
		Msg("fetching featured images")

	// Fetch active featured picks
	picks, err := h.picks.ListActive(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list active featured picks: %w", err)
	}

	// Collect all image IDs
	imageIDs := make([]gallery.ImageID, len(picks))
	for i, pick := range picks {
		imageIDs[i] = pick.ImageID()
	}

	// Fetch all images in one batch
	images, err := h.images.FindByIDs(ctx, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch featured images: %w", err)
	}

	// Index images by ID for O(1) lookup
	imageMap := make(map[string]*gallery.Image, len(images))
	for _, img := range images {
		imageMap[img.ID().String()] = img
	}

	// Build DTOs maintaining pick order
	featuredImages := make([]FeaturedImageDTO, 0, len(picks))
	for _, pick := range picks {
		image, found := imageMap[pick.ImageID().String()]
		if !found {
			h.logger.Warn().
				Str("image_id", pick.ImageID().String()).
				Msg("featured image not found, skipping")
			continue
		}

		// Build DTO with full image details
		featuredImages = append(featuredImages, FeaturedImageDTO{
			PickID:        pick.ID().String(),
			ImageID:       pick.ImageID().String(),
			Image:         mapImageToFeaturedInfo(image),
			DisplayOrder:  pick.DisplayOrder(),
			FeaturedSince: pick.FeaturedFrom().Format(time.RFC3339),
			Reason:        pick.Reason(),
		})
	}

	return &ListFeaturedImagesResult{
		FeaturedImages: featuredImages,
		TotalCount:     len(featuredImages),
	}, nil
}

// mapImageToFeaturedInfo converts a domain Image to a minimal FeaturedImageInfo.
func mapImageToFeaturedInfo(img *gallery.Image) *FeaturedImageInfo {
	return &FeaturedImageInfo{
		ID:          img.ID().String(),
		Title:       img.Metadata().Title(),
		Description: img.Metadata().Description(),
		OwnerID:     img.OwnerID().String(),
		Visibility:  img.Visibility().String(),
		CreatedAt:   img.CreatedAt().Format(time.RFC3339),
	}
}
