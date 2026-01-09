package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateVariantConfigCommand represents the intent to update a variant configuration.
type UpdateVariantConfigCommand struct {
	ConfigID  string
	UserID    string
	MaxWidth  int
	MaxHeight int
	Format    string
	Quality   int
	CropMode  string
}

// UpdateVariantConfigHandler processes variant config update commands.
type UpdateVariantConfigHandler struct {
	configs   gallery.VariantConfigRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewUpdateVariantConfigHandler creates a new UpdateVariantConfigHandler with the given dependencies.
func NewUpdateVariantConfigHandler(
	configs gallery.VariantConfigRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *UpdateVariantConfigHandler {
	return &UpdateVariantConfigHandler{
		configs:   configs,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the variant config update use case.
//
// Process flow:
//  1. Parse config ID and user ID
//  2. Retrieve config from repository
//  3. Verify ownership (user must own the config)
//  4. Parse format and crop mode
//  5. Update config via domain method
//  6. Persist changes
//  7. Publish domain events after successful save
//
// Returns:
//   - nil on success
//   - ErrVariantConfigNotFound if config doesn't exist
//   - Authorization error if user doesn't own the config
//   - Validation errors from domain value objects
func (h *UpdateVariantConfigHandler) Handle(ctx context.Context, cmd UpdateVariantConfigCommand) error {
	// 1. Parse IDs
	configID, err := gallery.ParseVariantConfigID(cmd.ConfigID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("config_id", cmd.ConfigID).
			Msg("invalid config id for update")
		return fmt.Errorf("invalid config id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for variant config update")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve config
	config, err := h.configs.FindByID(ctx, configID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("config_id", configID.String()).
			Msg("variant config not found for update")
		return fmt.Errorf("find variant config: %w", err)
	}

	// 3. Verify ownership
	if !config.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("config_id", configID.String()).
			Str("owner_id", config.UserID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized variant config update attempt")
		return fmt.Errorf("unauthorized: user does not own this variant config")
	}

	// 4. Parse format
	format := gallery.OutputFormat(cmd.Format)
	if !format.IsValid() {
		h.logger.Debug().
			Str("format", cmd.Format).
			Msg("invalid output format")
		return gallery.ErrVariantConfigFormatInvalid
	}

	// 5. Parse crop mode
	cropMode := gallery.CropMode(cmd.CropMode)
	if !cropMode.IsValid() {
		h.logger.Debug().
			Str("crop_mode", cmd.CropMode).
			Msg("invalid crop mode")
		return gallery.ErrVariantConfigCropModeInvalid
	}

	// 6. Update config via domain method
	if err := config.Update(cmd.MaxWidth, cmd.MaxHeight, format, cmd.Quality, cropMode); err != nil {
		h.logger.Debug().
			Err(err).
			Str("config_id", configID.String()).
			Msg("failed to update variant config")
		return fmt.Errorf("update variant config: %w", err)
	}

	// 7. Persist changes
	if err := h.configs.Save(ctx, config); err != nil {
		h.logger.Error().
			Err(err).
			Str("config_id", configID.String()).
			Msg("failed to save variant config updates")
		return fmt.Errorf("save variant config: %w", err)
	}

	// 8. Publish domain events
	for _, event := range config.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("config_id", configID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after variant config update")
		}
	}
	config.ClearEvents()

	h.logger.Info().
		Str("config_id", configID.String()).
		Str("user_id", userID.String()).
		Msg("variant config updated successfully")

	return nil
}
