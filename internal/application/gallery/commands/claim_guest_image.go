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

// ClaimGuestImageCommand represents the intent to claim an image uploaded
// during a guest session. This transfers ownership from a guest user
// to the authenticated registered user.
type ClaimGuestImageCommand struct {
	ImageID     string // The image to claim
	RequesterID string // The authenticated user claiming the image
	GuestUserID string // The guest user ID that uploaded the image (for verification)
}

// Ensure ClaimGuestImageCommand implements Command interface.
func (c ClaimGuestImageCommand) isCommand() {}

// ClaimGuestImageResult represents the result of a successful image claim.
type ClaimGuestImageResult struct {
	ImageID       string
	NewOwnerID    string
	PreviousOwner string
	TransferredAt string
	Message       string
}

// ClaimGuestImageHandler processes claim guest image commands.
// It orchestrates ownership validation, transfer, and event publishing.
type ClaimGuestImageHandler struct {
	images         gallery.ImageRepository
	users          identity.UserRepository
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewClaimGuestImageHandler creates a new ClaimGuestImageHandler with the given dependencies.
func NewClaimGuestImageHandler(
	images gallery.ImageRepository,
	users identity.UserRepository,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *ClaimGuestImageHandler {
	return &ClaimGuestImageHandler{
		images:         images,
		users:          users,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the claim guest image use case.
//
// Process flow:
//  1. Parse and validate image ID, requester ID, and guest user ID
//  2. Verify the requester is a registered (non-guest) user
//  3. Load image from repository
//  4. Verify the image is owned by the specified guest user
//  5. Verify the guest user is actually a guest account
//  6. Transfer ownership using Image.ChangeOwner()
//  7. Persist updated image
//  8. Publish domain events after successful save
//
// Returns:
//   - ClaimGuestImageResult on successful claim
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if user cannot claim this image
//   - Validation errors from domain value objects
//
//nolint:funlen // Sequential validation of IDs, user types, ownership, and transfer.
func (h *ClaimGuestImageHandler) Handle(ctx context.Context, cmd ClaimGuestImageCommand) (*ClaimGuestImageResult, error) {
	// 1. Parse and validate IDs
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during claim")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	requesterID, err := identity.ParseUserID(cmd.RequesterID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("requester_id", cmd.RequesterID).
			Msg("invalid requester id during claim")
		return nil, fmt.Errorf("invalid requester id: %w", err)
	}

	guestUserID, err := identity.ParseUserID(cmd.GuestUserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("guest_user_id", cmd.GuestUserID).
			Msg("invalid guest user id during claim")
		return nil, fmt.Errorf("invalid guest user id: %w", err)
	}

	// 2. Verify the requester is a registered user
	requester, err := h.users.FindByID(ctx, requesterID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			h.logger.Debug().
				Str("requester_id", requesterID.String()).
				Msg("requester not found during claim")
			return nil, fmt.Errorf("find requester: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("requester_id", requesterID.String()).
			Msg("failed to load requester for claim")
		return nil, fmt.Errorf("find requester: %w", err)
	}

	// Requester must be a registered user to claim images
	if requester.IsGuest() {
		h.logger.Warn().
			Str("requester_id", requesterID.String()).
			Msg("guest user attempted to claim image")
		return nil, fmt.Errorf("%w: only registered users can claim images", gallery.ErrUnauthorizedAccess)
	}

	// 3. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during claim")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for claim")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 4. Verify the image is owned by the specified guest user
	if image.OwnerID() != guestUserID {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("expected_owner", guestUserID.String()).
			Str("actual_owner", image.OwnerID().String()).
			Msg("image not owned by specified guest user")
		return nil, fmt.Errorf("%w: image not owned by specified guest", gallery.ErrUnauthorizedAccess)
	}

	// 5. Verify the current owner is actually a guest
	currentOwner, err := h.users.FindByID(ctx, image.OwnerID())
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			// Owner deleted - allow claim
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Str("owner_id", image.OwnerID().String()).
				Msg("image owner not found, allowing claim")
		} else {
			h.logger.Error().
				Err(err).
				Str("owner_id", image.OwnerID().String()).
				Msg("failed to load image owner for claim verification")
			return nil, fmt.Errorf("find image owner: %w", err)
		}
	} else if !currentOwner.IsGuest() {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("owner_id", image.OwnerID().String()).
			Msg("cannot claim image from non-guest user")
		return nil, fmt.Errorf("%w: cannot claim image from registered user", gallery.ErrUnauthorizedAccess)
	}

	// 6. Transfer ownership
	previousOwner := image.OwnerID()
	if err := image.ChangeOwner(requesterID); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("new_owner", requesterID.String()).
			Msg("failed to change image owner")
		return nil, fmt.Errorf("change owner: %w", err)
	}

	// 7. Persist updated image
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to save claimed image")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 8. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after image claim")
		}
	}
	image.ClearEvents()

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("previous_owner", previousOwner.String()).
		Str("new_owner", requesterID.String()).
		Msg("image claimed successfully")

	return &ClaimGuestImageResult{
		ImageID:       imageID.String(),
		NewOwnerID:    requesterID.String(),
		PreviousOwner: previousOwner.String(),
		TransferredAt: image.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Message:       "Image claimed successfully",
	}, nil
}
