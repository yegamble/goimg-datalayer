package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// FeatureImageCommand represents the intent to add an image to featured picks.
// Only admin users can feature images.
type FeatureImageCommand struct {
	ImageID       string     // Image to feature
	FeaturedBy    string     // Admin user featuring the image
	Reason        string     // Optional reason/notes
	DisplayOrder  int        // Display priority (0 = highest)
	FeaturedFrom  *time.Time // Start time (nil = now)
	FeaturedUntil *time.Time // End time (nil = no expiration)
}

// FeatureImageResult represents the result of successfully featuring an image.
type FeatureImageResult struct {
	PickID        string     `json:"pick_id"`
	ImageID       string     `json:"image_id"`
	DisplayOrder  int        `json:"display_order"`
	FeaturedFrom  time.Time  `json:"featured_from"`
	FeaturedUntil *time.Time `json:"featured_until,omitempty"`
}

// FeatureImageHandler processes feature image commands.
type FeatureImageHandler struct {
	images gallery.ImageRepository
	picks  gallery.FeaturedPickRepository
	logger zerolog.Logger
}

// NewFeatureImageHandler creates a new FeatureImageHandler.
func NewFeatureImageHandler(
	images gallery.ImageRepository,
	picks gallery.FeaturedPickRepository,
	logger zerolog.Logger,
) *FeatureImageHandler {
	return &FeatureImageHandler{
		images: images,
		picks:  picks,
		logger: logger,
	}
}

// Handle executes the feature image use case.
func (h *FeatureImageHandler) Handle(ctx context.Context, cmd FeatureImageCommand) (*FeatureImageResult, error) {
	// 1. Parse and validate image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 2. Parse and validate featured_by user ID
	featuredBy, err := identity.ParseUserID(cmd.FeaturedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid featured_by user id: %w", err)
	}

	// 3. Verify image exists and is public
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("find image: %w", err)
	}

	if image.Visibility() != gallery.VisibilityPublic {
		return nil, fmt.Errorf("cannot feature non-public image: %w", gallery.ErrInvalidVisibility)
	}

	if image.Status() != gallery.StatusActive {
		return nil, fmt.Errorf("cannot feature non-active image: %w", gallery.ErrInvalidImageStatus)
	}

	// 4. Check if image is already featured
	exists, err := h.picks.ExistsByImageID(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("check if image is featured: %w", err)
	}
	if exists {
		return nil, gallery.ErrImageAlreadyFeatured
	}

	// 5. Determine scheduling
	featuredFrom := time.Now().UTC()
	if cmd.FeaturedFrom != nil {
		featuredFrom = cmd.FeaturedFrom.UTC()
	}

	// 6. Create featured pick via domain factory
	pick, err := gallery.NewFeaturedPickWithSchedule(
		imageID,
		featuredBy,
		cmd.Reason,
		cmd.DisplayOrder,
		featuredFrom,
		cmd.FeaturedUntil,
	)
	if err != nil {
		return nil, fmt.Errorf("create featured pick: %w", err)
	}

	// 7. Persist
	if err := h.picks.Save(ctx, pick); err != nil {
		return nil, fmt.Errorf("save featured pick: %w", err)
	}

	h.logger.Info().
		Str("pick_id", pick.ID().String()).
		Str("image_id", imageID.String()).
		Str("featured_by", featuredBy.String()).
		Int("display_order", cmd.DisplayOrder).
		Msg("image featured successfully")

	return &FeatureImageResult{
		PickID:        pick.ID().String(),
		ImageID:       pick.ImageID().String(),
		DisplayOrder:  pick.DisplayOrder(),
		FeaturedFrom:  pick.FeaturedFrom(),
		FeaturedUntil: pick.FeaturedUntil(),
	}, nil
}
