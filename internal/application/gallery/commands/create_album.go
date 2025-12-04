package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// CreateAlbumCommand represents the intent to create a new album.
type CreateAlbumCommand struct {
	UserID      string
	Title       string
	Description string
	Visibility  string
}

// CreateAlbumHandler processes album creation commands.
// It validates inputs, creates the album aggregate, and persists it.
type CreateAlbumHandler struct {
	albums    gallery.AlbumRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewCreateAlbumHandler creates a new CreateAlbumHandler with the given dependencies.
func NewCreateAlbumHandler(
	albums gallery.AlbumRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *CreateAlbumHandler {
	return &CreateAlbumHandler{
		albums:    albums,
		users:     users,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the album creation use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Verify user exists
//  3. Parse visibility (defaults to private if invalid)
//  4. Create Album aggregate via domain factory (validates title)
//  5. Set description and visibility if provided
//  6. Persist album via repository
//  7. Publish domain events after successful save
//  8. Return album ID
//
// Returns:
//   - AlbumID string on successful creation
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
func (h *CreateAlbumHandler) Handle(ctx context.Context, cmd CreateAlbumCommand) (string, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for album creation")
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for album creation")
		return "", fmt.Errorf("find user: %w", err)
	}

	// 3. Create album via domain factory (validates title)
	album, err := gallery.NewAlbum(userID, cmd.Title)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("title", cmd.Title).
			Msg("invalid album title")
		return "", fmt.Errorf("create album: %w", err)
	}

	// 4. Set description if provided
	if cmd.Description != "" {
		if err := album.UpdateDescription(cmd.Description); err != nil {
			h.logger.Debug().
				Err(err).
				Str("description", cmd.Description).
				Msg("invalid album description")
			return "", fmt.Errorf("set description: %w", err)
		}
	}

	// 5. Set visibility if provided (defaults to private)
	if cmd.Visibility != "" {
		visibility, err := gallery.ParseVisibility(cmd.Visibility)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("visibility", cmd.Visibility).
				Msg("invalid visibility, defaulting to private")
		} else {
			if err := album.UpdateVisibility(visibility); err != nil {
				h.logger.Debug().
					Err(err).
					Str("visibility", cmd.Visibility).
					Msg("failed to set visibility")
				return "", fmt.Errorf("set visibility: %w", err)
			}
		}
	}

	// 6. Persist album
	if err := h.albums.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", album.ID().String()).
			Str("user_id", userID.String()).
			Msg("failed to save album")
		return "", fmt.Errorf("save album: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	for _, event := range album.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", album.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after album creation")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", album.ID().String()).
		Str("user_id", userID.String()).
		Str("title", album.Title()).
		Msg("album created successfully")

	return album.ID().String(), nil
}

// EventPublisher defines the interface for publishing domain events.
// This interface should be defined once and reused across all command handlers.
type EventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}
