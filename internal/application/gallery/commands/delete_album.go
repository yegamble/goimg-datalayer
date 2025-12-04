package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// DeleteAlbumCommand represents the intent to delete an album.
// This removes the album but does not delete the images (just the association).
type DeleteAlbumCommand struct {
	AlbumID string
	UserID  string
}

// DeleteAlbumHandler processes album deletion commands.
// It validates ownership and removes the album.
type DeleteAlbumHandler struct {
	albums    gallery.AlbumRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewDeleteAlbumHandler creates a new DeleteAlbumHandler with the given dependencies.
func NewDeleteAlbumHandler(
	albums gallery.AlbumRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *DeleteAlbumHandler {
	return &DeleteAlbumHandler{
		albums:    albums,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the album deletion use case.
//
// Process flow:
//  1. Parse album ID and user ID
//  2. Retrieve album from repository
//  3. Verify ownership (user must own the album)
//  4. Delete album (images remain, only association is removed)
//  5. Publish domain events
//
// Returns:
//   - nil on success (idempotent - no error if album already deleted)
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if user doesn't own the album
func (h *DeleteAlbumHandler) Handle(ctx context.Context, cmd DeleteAlbumCommand) error {
	// 1. Parse IDs
	albumID, err := gallery.ParseAlbumID(cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID).
			Msg("invalid album id for deletion")
		return fmt.Errorf("invalid album id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for album deletion")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve album
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		// Idempotent: if album not found, consider it already deleted
		if errors.Is(err, gallery.ErrAlbumNotFound) {
			h.logger.Debug().
				Str("album_id", albumID.String()).
				Msg("album already deleted or never existed")
			return nil
		}
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("failed to find album for deletion")
		return fmt.Errorf("find album: %w", err)
	}

	// 3. Verify ownership
	if !album.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("album_id", albumID.String()).
			Str("owner_id", album.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized album deletion attempt")
		return fmt.Errorf("unauthorized: user does not own this album")
	}

	// 4. Delete album (hard delete - images remain)
	if err := h.albums.Delete(ctx, albumID); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("failed to delete album")
		return fmt.Errorf("delete album: %w", err)
	}

	// 5. Publish domain events (if any were generated before deletion)
	// Note: Album entity doesn't emit events on deletion in the current implementation
	// but we keep this pattern for consistency
	for _, event := range album.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", albumID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after album deletion")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", albumID.String()).
		Str("user_id", userID.String()).
		Msg("album deleted successfully")

	return nil
}
