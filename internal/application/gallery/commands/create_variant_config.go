package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// CreateVariantConfigCommand represents the intent to create a custom variant configuration.
type CreateVariantConfigCommand struct {
	UserID    string
	Name      string
	MaxWidth  int
	MaxHeight int
	Format    string
	Quality   int
	CropMode  string
}

// CreateVariantConfigHandler processes variant config creation commands.
// It validates inputs, creates the config aggregate, and persists it.
type CreateVariantConfigHandler struct {
	configs   gallery.VariantConfigRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewCreateVariantConfigHandler creates a new CreateVariantConfigHandler with the given dependencies.
func NewCreateVariantConfigHandler(
	configs gallery.VariantConfigRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *CreateVariantConfigHandler {
	return &CreateVariantConfigHandler{
		configs:   configs,
		users:     users,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the variant config creation use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Verify user exists
//  3. Check for duplicate config name for user
//  4. Parse format and crop mode
//  5. Create VariantConfig via domain factory
//  6. Persist config via repository
//  7. Publish domain events after successful save
//  8. Return config ID
//
// Returns:
//   - VariantConfigID string on successful creation
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
func (h *CreateVariantConfigHandler) Handle(ctx context.Context, cmd CreateVariantConfigCommand) (string, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for variant config creation")
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for variant config creation")
		return "", fmt.Errorf("find user: %w", err)
	}

	// 3. Check for duplicate config name
	_, err = h.configs.FindByUserAndName(ctx, userID, cmd.Name)
	if err == nil {
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("name", cmd.Name).
			Msg("variant config name already exists for user")
		return "", fmt.Errorf("variant config with name %q already exists", cmd.Name)
	}

	// 4. Parse format
	format := gallery.OutputFormat(cmd.Format)
	if !format.IsValid() {
		h.logger.Debug().
			Str("format", cmd.Format).
			Msg("invalid output format")
		return "", gallery.ErrVariantConfigFormatInvalid
	}

	// 5. Parse crop mode
	cropMode := gallery.CropMode(cmd.CropMode)
	if !cropMode.IsValid() {
		h.logger.Debug().
			Str("crop_mode", cmd.CropMode).
			Msg("invalid crop mode")
		return "", gallery.ErrVariantConfigCropModeInvalid
	}

	// 6. Create variant config via domain factory
	config, err := gallery.NewVariantConfig(
		userID,
		cmd.Name,
		cmd.MaxWidth,
		cmd.MaxHeight,
		format,
		cmd.Quality,
		cropMode,
	)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("name", cmd.Name).
			Msg("failed to create variant config")
		return "", fmt.Errorf("create variant config: %w", err)
	}

	// 7. Persist config
	if err := h.configs.Save(ctx, config); err != nil {
		h.logger.Error().
			Err(err).
			Str("config_id", config.ID().String()).
			Str("user_id", userID.String()).
			Msg("failed to save variant config")
		return "", fmt.Errorf("save variant config: %w", err)
	}

	// 8. Publish domain events AFTER successful save
	for _, event := range config.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("config_id", config.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after variant config creation")
		}
	}
	config.ClearEvents()

	h.logger.Info().
		Str("config_id", config.ID().String()).
		Str("user_id", userID.String()).
		Str("name", config.Name()).
		Msg("variant config created successfully")

	return config.ID().String(), nil
}
