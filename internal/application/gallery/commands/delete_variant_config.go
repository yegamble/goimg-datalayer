package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// DeleteVariantConfigCommand represents the intent to delete a variant configuration.
type DeleteVariantConfigCommand struct {
	ConfigID string
	UserID   string
}

// DeleteVariantConfigHandler processes variant config deletion commands.
type DeleteVariantConfigHandler struct {
	configs   gallery.VariantConfigRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewDeleteVariantConfigHandler creates a new DeleteVariantConfigHandler with the given dependencies.
func NewDeleteVariantConfigHandler(
	configs gallery.VariantConfigRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *DeleteVariantConfigHandler {
	return &DeleteVariantConfigHandler{
		configs:   configs,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the variant config deletion use case.
//
// Process flow:
//  1. Parse config ID and user ID
//  2. Retrieve config from repository
//  3. Verify ownership (user must own the config)
//  4. Delete config via repository
//  5. Publish domain events after successful deletion
//
// Returns:
//   - nil on success
//   - ErrVariantConfigNotFound if config doesn't exist
//   - Authorization error if user doesn't own the config
func (h *DeleteVariantConfigHandler) Handle(ctx context.Context, cmd DeleteVariantConfigCommand) error {
	// 1. Parse IDs
	configID, err := gallery.ParseVariantConfigID(cmd.ConfigID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("config_id", cmd.ConfigID).
			Msg("invalid config id for deletion")
		return fmt.Errorf("invalid config id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for variant config deletion")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve config
	config, err := h.configs.FindByID(ctx, configID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("config_id", configID.String()).
			Msg("variant config not found for deletion")
		return fmt.Errorf("find variant config: %w", err)
	}

	// 3. Verify ownership
	if !config.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("config_id", configID.String()).
			Str("owner_id", config.UserID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized variant config deletion attempt")
		return fmt.Errorf("unauthorized: user does not own this variant config")
	}

	// 4. Cannot delete system presets
	if config.IsPreset() {
		h.logger.Warn().
			Str("config_id", configID.String()).
			Str("user_id", userID.String()).
			Msg("cannot delete system preset variant config")
		return fmt.Errorf("cannot delete system preset variant configs")
	}

	// 5. Delete config
	if err := h.configs.Delete(ctx, configID); err != nil {
		h.logger.Error().
			Err(err).
			Str("config_id", configID.String()).
			Msg("failed to delete variant config")
		return fmt.Errorf("delete variant config: %w", err)
	}

	// 6. Publish domain event
	event := &gallery.VariantConfigDeleted{
		VariantConfigID: configID,
	}
	if err := h.publisher.Publish(ctx, event); err != nil {
		h.logger.Error().
			Err(err).
			Str("config_id", configID.String()).
			Str("event_type", event.EventType()).
			Msg("failed to publish domain event after variant config deletion")
	}

	h.logger.Info().
		Str("config_id", configID.String()).
		Str("user_id", userID.String()).
		Msg("variant config deleted successfully")

	return nil
}
