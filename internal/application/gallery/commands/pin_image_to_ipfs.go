package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appgallery "github.com/yegamble/goimg-datalayer/internal/application/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// PinImageToIPFSCommand represents the intent to pin an image to IPFS.
// This creates a decentralized backup of the image content.
type PinImageToIPFSCommand struct {
	ImageID string // ID of the image to pin
	UserID  string // ID of the user requesting the pin (must be owner)
}

// PinImageToIPFSResult contains the result of a successful IPFS pin operation.
type PinImageToIPFSResult struct {
	ImageID    string    // ID of the pinned image
	CID        string    // IPFS Content Identifier
	GatewayURL string    // Public gateway URL for the content
	PinnedAt   time.Time // When the content was pinned
}

// PinImageToIPFSHandler processes IPFS pin commands.
// It orchestrates uploading image content to IPFS and updating the domain model.
type PinImageToIPFSHandler struct {
	images         gallery.ImageRepository
	storage        appgallery.StorageProvider
	ipfs           appgallery.IPFSService
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewPinImageToIPFSHandler creates a new PinImageToIPFSHandler with the given dependencies.
func NewPinImageToIPFSHandler(
	images gallery.ImageRepository,
	storage appgallery.StorageProvider,
	ipfs appgallery.IPFSService,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *PinImageToIPFSHandler {
	return &PinImageToIPFSHandler{
		images:         images,
		storage:        storage,
		ipfs:           ipfs,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the IPFS pin use case.
//
// Process flow:
//  1. Parse and validate image ID and user ID
//  2. Load image from repository
//  3. Verify ownership (user must own the image)
//  4. Check if already pinned to IPFS
//  5. Retrieve image content from primary storage
//  6. Upload content to IPFS
//  7. Update image with IPFS metadata
//  8. Persist updated image
//  9. Publish domain events after successful save
//
// Returns:
//   - PinImageToIPFSResult on successful pin
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if user doesn't own image
//   - ErrAlreadyPinned if image is already on IPFS
func (h *PinImageToIPFSHandler) Handle(ctx context.Context, cmd PinImageToIPFSCommand) (*PinImageToIPFSResult, error) {
	// 1. Parse and validate IDs
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during IPFS pin")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during IPFS pin")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during IPFS pin")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for IPFS pin")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Verify ownership
	if !image.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Str("owner_id", image.OwnerID().String()).
			Msg("unauthorized IPFS pin attempt")
		return nil, gallery.ErrUnauthorizedAccess
	}

	// 4. Check if already pinned
	if image.HasIPFS() {
		h.logger.Debug().
			Str("image_id", imageID.String()).
			Str("existing_cid", image.IPFSMetadata().CID()).
			Msg("image already pinned to IPFS")
		return nil, fmt.Errorf("image already pinned to IPFS with CID: %s", image.IPFSMetadata().CID())
	}

	// 5. Retrieve image content from primary storage
	storageKey := image.Metadata().StorageKey()
	data, err := h.storage.GetBytes(ctx, storageKey)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("storage_key", storageKey).
			Msg("failed to retrieve image from storage for IPFS pin")
		return nil, fmt.Errorf("retrieve image content: %w", err)
	}

	// 6. Upload content to IPFS
	cid, err := h.ipfs.Add(ctx, data)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to upload image to IPFS")
		return nil, fmt.Errorf("upload to IPFS: %w", err)
	}

	// Pin the content (in case it wasn't pinned by default)
	if err := h.ipfs.Pin(ctx, cid); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("cid", cid).
			Msg("failed to pin content on IPFS")
		return nil, fmt.Errorf("pin to IPFS: %w", err)
	}

	// 7. Update image with IPFS metadata
	pinnedAt := time.Now().UTC()
	ipfsMetadata, err := gallery.NewIPFSMetadata(cid, true, &pinnedAt)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("cid", cid).
			Msg("failed to create IPFS metadata")
		return nil, fmt.Errorf("create IPFS metadata: %w", err)
	}

	if err := image.SetIPFSMetadata(ipfsMetadata); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to set IPFS metadata on image")
		return nil, fmt.Errorf("set IPFS metadata: %w", err)
	}

	// 8. Persist updated image
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to save image after IPFS pin")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 9. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after IPFS pin")
		}
	}
	image.ClearEvents()

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Str("cid", cid).
		Msg("image pinned to IPFS successfully")

	return &PinImageToIPFSResult{
		ImageID:    imageID.String(),
		CID:        cid,
		GatewayURL: h.ipfs.GatewayURL(cid),
		PinnedAt:   pinnedAt,
	}, nil
}
