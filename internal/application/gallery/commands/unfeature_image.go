package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// UnfeatureImageCommand represents the intent to remove an image from featured picks.
// Only admin users can unfeature images.
type UnfeatureImageCommand struct {
	ImageID string // Image to unfeature
}

// UnfeatureImageHandler processes unfeature image commands.
type UnfeatureImageHandler struct {
	picks  gallery.FeaturedPickRepository
	logger zerolog.Logger
}

// NewUnfeatureImageHandler creates a new UnfeatureImageHandler.
func NewUnfeatureImageHandler(
	picks gallery.FeaturedPickRepository,
	logger zerolog.Logger,
) *UnfeatureImageHandler {
	return &UnfeatureImageHandler{
		picks:  picks,
		logger: logger,
	}
}

// Handle executes the unfeature image use case.
func (h *UnfeatureImageHandler) Handle(ctx context.Context, cmd UnfeatureImageCommand) error {
	// 1. Parse and validate image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		return fmt.Errorf("invalid image id: %w", err)
	}

	// 2. Find active featured pick for this image
	pick, err := h.picks.FindByImageID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("find featured pick: %w", err)
	}

	// 3. Expire the featured pick via domain method
	pick.Expire()

	// 4. Persist (we could also delete, but expiring preserves audit trail)
	if err := h.picks.Save(ctx, pick); err != nil {
		return fmt.Errorf("save featured pick: %w", err)
	}

	h.logger.Info().
		Str("pick_id", pick.ID().String()).
		Str("image_id", imageID.String()).
		Msg("image unfeatured successfully")

	return nil
}
