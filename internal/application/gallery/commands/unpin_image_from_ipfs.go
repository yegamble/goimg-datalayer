package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	appgallery "github.com/yegamble/goimg-datalayer/internal/application/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UnpinImageFromIPFSCommand represents the intent to unpin an image from IPFS.
// This removes the decentralized backup but keeps the primary storage copy.
type UnpinImageFromIPFSCommand struct {
	ImageID string // ID of the image to unpin
	UserID  string // ID of the user requesting the unpin (must be owner)
}

// UnpinImageFromIPFSResult contains the result of a successful IPFS unpin operation.
type UnpinImageFromIPFSResult struct {
	ImageID     string // ID of the unpinned image
	UnpinnedCID string // The CID that was unpinned
	Message     string // Success message
}

// UnpinImageFromIPFSHandler processes IPFS unpin commands.
// It orchestrates removing content from IPFS and updating the domain model.
type UnpinImageFromIPFSHandler struct {
	images         gallery.ImageRepository
	ipfs           appgallery.IPFSService
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewUnpinImageFromIPFSHandler creates a new UnpinImageFromIPFSHandler with the given dependencies.
func NewUnpinImageFromIPFSHandler(
	images gallery.ImageRepository,
	ipfs appgallery.IPFSService,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *UnpinImageFromIPFSHandler {
	return &UnpinImageFromIPFSHandler{
		images:         images,
		ipfs:           ipfs,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the IPFS unpin use case.
//
// Process flow:
//  1. Parse and validate image ID and user ID
//  2. Load image from repository
//  3. Verify ownership (user must own the image)
//  4. Check if image has IPFS metadata
//  5. Unpin content from IPFS node
//  6. Clear IPFS metadata from image
//  7. Persist updated image
//  8. Publish domain events after successful save
//
// Returns:
//   - UnpinImageFromIPFSResult on successful unpin
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if user doesn't own image
//   - Error if image is not pinned to IPFS
func (h *UnpinImageFromIPFSHandler) Handle(ctx context.Context, cmd UnpinImageFromIPFSCommand) (*UnpinImageFromIPFSResult, error) {
	// 1. Parse and validate IDs
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during IPFS unpin")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during IPFS unpin")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during IPFS unpin")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for IPFS unpin")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Verify ownership
	if !image.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Str("owner_id", image.OwnerID().String()).
			Msg("unauthorized IPFS unpin attempt")
		return nil, gallery.ErrUnauthorizedAccess
	}

	// 4. Check if image has IPFS metadata
	if !image.HasIPFS() {
		h.logger.Debug().
			Str("image_id", imageID.String()).
			Msg("image is not pinned to IPFS")
		return nil, fmt.Errorf("image is not pinned to IPFS")
	}

	cid := image.IPFSMetadata().CID()

	// 5. Unpin content from IPFS node
	if err := h.ipfs.Unpin(ctx, cid); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("cid", cid).
			Msg("failed to unpin content from IPFS")
		return nil, fmt.Errorf("unpin from IPFS: %w", err)
	}

	// 6. Clear IPFS metadata from image
	if err := image.ClearIPFSMetadata(); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to clear IPFS metadata from image")
		return nil, fmt.Errorf("clear IPFS metadata: %w", err)
	}

	// 7. Persist updated image
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to save image after IPFS unpin")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 8. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after IPFS unpin")
		}
	}
	image.ClearEvents()

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Str("cid", cid).
		Msg("image unpinned from IPFS successfully")

	return &UnpinImageFromIPFSResult{
		ImageID:     imageID.String(),
		UnpinnedCID: cid,
		Message:     "Image successfully unpinned from IPFS",
	}, nil
}
