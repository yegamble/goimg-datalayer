package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// AddImageToAlbumCommand represents the intent to add an image to an album.
type AddImageToAlbumCommand struct {
	AlbumID string
	ImageID string
	UserID  string
}

// AddImageToAlbumHandler processes commands to add images to albums.
// It validates that the user owns both the album and the image before adding.
type AddImageToAlbumHandler struct {
	albums      gallery.AlbumRepository
	images      gallery.ImageRepository
	albumImages gallery.AlbumImageRepository
	publisher   EventPublisher
	logger      *zerolog.Logger
}

// NewAddImageToAlbumHandler creates a new AddImageToAlbumHandler.
func NewAddImageToAlbumHandler(
	albums gallery.AlbumRepository,
	images gallery.ImageRepository,
	albumImages gallery.AlbumImageRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *AddImageToAlbumHandler {
	return &AddImageToAlbumHandler{
		albums:      albums,
		images:      images,
		albumImages: albumImages,
		publisher:   publisher,
		logger:      logger,
	}
}

// Handle executes the add image to album use case.
//
// Process flow:
//  1. Parse album ID, image ID, and user ID
//  2. Retrieve album and verify ownership
//  3. Retrieve image and verify ownership
//  4. Add image to album via AlbumImageRepository
//  5. Update album's image count
//  6. Persist album changes
//  7. Publish domain events
//
// Returns:
//   - nil on success
//   - ErrAlbumNotFound if album doesn't exist
//   - ErrImageNotFound if image doesn't exist
//   - Authorization error if user doesn't own the album or image
//   - Error if image is already in the album
//
//nolint:cyclop // Command handler requires sequential validation: album ID, image ID, ownership checks, and persistence
func (h *AddImageToAlbumHandler) Handle(ctx context.Context, cmd AddImageToAlbumCommand) error {
	// 1. Parse IDs
	albumID, err := gallery.ParseAlbumID(cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID).
			Msg("invalid album id for add image")
		return fmt.Errorf("invalid album id: %w", err)
	}

	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for add to album")
		return fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for add image to album")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve and verify album ownership
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("album not found for add image")
		return fmt.Errorf("find album: %w", err)
	}

	if !album.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("album_id", albumID.String()).
			Str("album_owner_id", album.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized add image to album attempt (album not owned)")
		return fmt.Errorf("unauthorized: user does not own this album")
	}

	// 3. Retrieve and verify image ownership
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found for add to album")
		return fmt.Errorf("find image: %w", err)
	}

	if !image.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("image_owner_id", image.OwnerID().String()).
			Str("user_id", userID.String()).
			Msg("unauthorized add image to album attempt (image not owned)")
		return fmt.Errorf("unauthorized: user does not own this image")
	}

	// 4. Add image to album
	if err := h.albumImages.AddImageToAlbum(ctx, albumID, imageID); err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", albumID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to add image to album")
		return fmt.Errorf("add image to album: %w", err)
	}

	// 5. Update album's image count
	album.IncrementImageCount()

	// 6. Persist album changes
	if err := h.albums.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", albumID.String()).
			Msg("failed to save album after adding image")
		return fmt.Errorf("save album: %w", err)
	}

	// 7. Publish domain events
	for _, event := range album.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", albumID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after adding image to album")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", albumID.String()).
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Msg("image added to album successfully")

	return nil
}
