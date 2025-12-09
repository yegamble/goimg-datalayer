package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateAlbumCommand represents the intent to update an album's metadata.
// Optional fields use pointers (nil = no change).
type UpdateAlbumCommand struct {
	AlbumID     string
	UserID      string
	Title       *string // Optional: nil means no change
	Description *string // Optional: nil means no change
	Visibility  *string // Optional: nil means no change
}

// UpdateAlbumHandler processes album update commands.
// It validates ownership, applies updates, and persists changes.
type UpdateAlbumHandler struct {
	albums    gallery.AlbumRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewUpdateAlbumHandler creates a new UpdateAlbumHandler with the given dependencies.
func NewUpdateAlbumHandler(
	albums gallery.AlbumRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *UpdateAlbumHandler {
	return &UpdateAlbumHandler{
		albums:    albums,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the album update use case.
//
// Process flow:
//  1. Parse album ID and user ID
//  2. Retrieve album from repository
//  3. Verify ownership (user must own the album)
//  4. Apply updates via domain methods (if provided)
//  5. Persist changes if any updates were made
//  6. Publish domain events after successful save
//
// Returns:
//   - nil on success
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if user doesn't own the album
//   - Validation errors from domain value objects
//
//nolint:funlen // Command/query handler with sequential validation and business logic
func (h *UpdateAlbumHandler) Handle(ctx context.Context, cmd UpdateAlbumCommand) error {
	// 1. Parse IDs
	albumID, err := gallery.ParseAlbumID(cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID).
			Msg("invalid album id for update")
		return fmt.Errorf("invalid album id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for album update")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve album
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("album not found for update")
		return fmt.Errorf("find album: %w", err)
	}

	// 3. Verify ownership
	if !album.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("album_id", albumID.String()).
			Str("owner_id", album.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized album update attempt")
		return fmt.Errorf("unauthorized: user does not own this album")
	}

	// 4. Apply updates via domain methods
	updateNeeded, err := h.applyAlbumUpdates(album, cmd)
	if err != nil {
		return err
	}

	// 5. If no changes, return early
	if !updateNeeded {
		h.logger.Debug().
			Str("album_id", albumID.String()).
			Msg("no album updates needed")
		return nil
	}

	// 6. Persist changes
	if err := h.albums.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("failed to save album updates")
		return fmt.Errorf("save album: %w", err)
	}

	// 7. Publish domain events
	for _, event := range album.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", albumID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after album update")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", albumID.String()).
		Str("user_id", userID.String()).
		Msg("album updated successfully")

	return nil
}

// applyAlbumUpdates applies all update fields to the album.
// Returns true if any updates were made, false otherwise.
func (h *UpdateAlbumHandler) applyAlbumUpdates(album *gallery.Album, cmd UpdateAlbumCommand) (bool, error) {
	updateNeeded := false

	// Update title if provided and different
	if cmd.Title != nil && *cmd.Title != album.Title() {
		if err := album.UpdateTitle(*cmd.Title); err != nil {
			h.logger.Debug().
				Err(err).
				Str("title", *cmd.Title).
				Msg("invalid album title for update")
			return false, fmt.Errorf("update title: %w", err)
		}
		updateNeeded = true
	}

	// Update description if provided and different
	if cmd.Description != nil && *cmd.Description != album.Description() {
		if err := album.UpdateDescription(*cmd.Description); err != nil {
			h.logger.Debug().
				Err(err).
				Str("description", *cmd.Description).
				Msg("invalid album description for update")
			return false, fmt.Errorf("update description: %w", err)
		}
		updateNeeded = true
	}

	// Update visibility if provided
	if cmd.Visibility != nil {
		updated, err := h.updateAlbumVisibility(album, *cmd.Visibility)
		if err != nil {
			return false, err
		}
		if updated {
			updateNeeded = true
		}
	}

	return updateNeeded, nil
}

// updateAlbumVisibility updates the album visibility if it has changed.
// Returns true if the visibility was updated, false otherwise.
func (h *UpdateAlbumHandler) updateAlbumVisibility(album *gallery.Album, visibilityStr string) (bool, error) {
	visibility, err := gallery.ParseVisibility(visibilityStr)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("visibility", visibilityStr).
			Msg("invalid visibility for album update")
		return false, fmt.Errorf("invalid visibility: %w", err)
	}

	if visibility == album.Visibility() {
		return false, nil
	}

	if err := album.UpdateVisibility(visibility); err != nil {
		h.logger.Debug().
			Err(err).
			Str("visibility", visibilityStr).
			Msg("failed to update album visibility")
		return false, fmt.Errorf("update visibility: %w", err)
	}

	return true, nil
}
