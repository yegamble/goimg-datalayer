package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RemoveImageFromAlbumCommand represents the intent to remove an image from an album.
// The image itself is not deleted, only the association is removed.
type RemoveImageFromAlbumCommand struct {
	AlbumID string
	ImageID string
	UserID  string
}

// RemoveImageFromAlbumHandler processes commands to remove images from albums.
// It validates ownership and removes the association.
type RemoveImageFromAlbumHandler struct {
	albums      gallery.AlbumRepository
	albumImages gallery.AlbumImageRepository
	publisher   EventPublisher
	logger      *zerolog.Logger
}

// NewRemoveImageFromAlbumHandler creates a new RemoveImageFromAlbumHandler.
func NewRemoveImageFromAlbumHandler(
	albums gallery.AlbumRepository,
	albumImages gallery.AlbumImageRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *RemoveImageFromAlbumHandler {
	return &RemoveImageFromAlbumHandler{
		albums:      albums,
		albumImages: albumImages,
		publisher:   publisher,
		logger:      logger,
	}
}

// Handle executes the remove image from album use case.
//
// Process flow:
//  1. Parse album ID, image ID, and user ID
//  2. Retrieve album and verify ownership
//  3. Remove image from album via AlbumImageRepository (idempotent)
//  4. Update album's image count
//  5. Persist album changes
//  6. Publish domain events
//
// Returns:
//   - nil on success (idempotent - no error if image wasn't in album)
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if user doesn't own the album
func (h *RemoveImageFromAlbumHandler) Handle(ctx context.Context, cmd RemoveImageFromAlbumCommand) error {
	// 1. Parse IDs
	albumID, err := gallery.ParseAlbumID(cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID).
			Msg("invalid album id for remove image")
		return fmt.Errorf("invalid album id: %w", err)
	}

	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for remove from album")
		return fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for remove image from album")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve and verify album ownership
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("album not found for remove image")
		return fmt.Errorf("find album: %w", err)
	}

	if !album.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("album_id", albumID.String()).
			Str("album_owner_id", album.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized remove image from album attempt")
		return fmt.Errorf("unauthorized: user does not own this album")
	}

	// 3. Remove image from album (idempotent)
	if err := h.albumImages.RemoveImageFromAlbum(ctx, albumID, imageID); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to remove image from album")
		return fmt.Errorf("remove image from album: %w", err)
	}

	// 4. Update album's image count
	album.DecrementImageCount()

	// 5. Persist album changes
	if err := h.albums.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("failed to save album after removing image")
		return fmt.Errorf("save album: %w", err)
	}

	// 6. Publish domain events
	for _, event := range album.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", albumID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after removing image from album")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", albumID.String()).
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Msg("image removed from album successfully")

	return nil
}
