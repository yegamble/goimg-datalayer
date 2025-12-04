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

// DeleteImageCommand represents the intent to delete an image.
// It encapsulates the image ID and user ID for authorization.
type DeleteImageCommand struct {
	ImageID string
	UserID  string
	// UserRole allows moderators/admins to delete images they don't own
	UserRole string
}

// Implement Command interface
func (DeleteImageCommand) isCommand() {}

// DeleteImageResult represents the result of a successful image deletion.
type DeleteImageResult struct {
	ImageID        string
	DeletedAt      string
	VariantsCount  int
	CleanupJobID   string
	Message        string
}

// DeleteImageHandler processes image deletion commands.
// It orchestrates authorization, soft deletion, storage cleanup, and event publishing.
type DeleteImageHandler struct {
	images         gallery.ImageRepository
	jobEnqueuer    appgallery.JobEnqueuer
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewDeleteImageHandler creates a new DeleteImageHandler with the given dependencies.
func NewDeleteImageHandler(
	images gallery.ImageRepository,
	jobEnqueuer appgallery.JobEnqueuer,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *DeleteImageHandler {
	return &DeleteImageHandler{
		images:         images,
		jobEnqueuer:    jobEnqueuer,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the image deletion use case.
//
// Process flow:
//  1. Parse and validate image ID and user ID
//  2. Load image from repository
//  3. Verify authorization (owner OR admin/moderator)
//  4. Mark image as deleted via domain method
//  5. Collect storage keys for all variants
//  6. Persist soft-deleted image
//  7. Publish domain events after successful save
//  8. Enqueue cleanup job for storage deletion
//
// Returns:
//   - DeleteImageResult on successful deletion
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if user lacks permission
//   - ErrCannotDeleteFlagged if image is flagged for moderation
func (h *DeleteImageHandler) Handle(ctx context.Context, cmd DeleteImageCommand) (*DeleteImageResult, error) {
	// 1. Parse and validate IDs
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during deletion")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during image deletion")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Parse user role for authorization
	userRole := identity.RoleUser // Default role
	if cmd.UserRole != "" {
		userRole, err = identity.ParseRole(cmd.UserRole)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("role", cmd.UserRole).
				Msg("invalid user role during image deletion")
			return nil, fmt.Errorf("invalid user role: %w", err)
		}
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during deletion")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for deletion")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Verify authorization
	isOwner := image.IsOwnedBy(userID)
	isModerator := userRole == identity.RoleModerator || userRole == identity.RoleAdmin

	if !isOwner && !isModerator {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Str("owner_id", image.OwnerID().String()).
			Str("user_role", userRole.String()).
			Msg("unauthorized image deletion attempt")
		return nil, gallery.ErrUnauthorizedAccess
	}

	// 4. Mark image as deleted via domain method
	if err := image.MarkAsDeleted(); err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to mark image as deleted")
		return nil, fmt.Errorf("mark as deleted: %w", err)
	}

	// 5. Collect storage keys for all variants (for cleanup job)
	storageKeys := []string{image.Metadata().StorageKey()}
	for _, variant := range image.Variants() {
		storageKeys = append(storageKeys, variant.StorageKey())
	}

	storageProvider := image.Metadata().StorageProvider()

	// 6. Persist soft-deleted image
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to save deleted image")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after image deletion")
		}
	}
	image.ClearEvents()

	// 8. Enqueue cleanup job for storage deletion
	cleanupJobID := ""
	if err := h.jobEnqueuer.EnqueueImageCleanup(ctx, imageID.String(), storageProvider, storageKeys); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Int("storage_keys_count", len(storageKeys)).
			Msg("failed to enqueue image cleanup job")
		// Don't fail the deletion if job enqueueing fails
		cleanupJobID = "failed"
	} else {
		cleanupJobID = "enqueued"
	}

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Bool("is_owner", isOwner).
		Bool("is_moderator", isModerator).
		Int("variants_count", len(image.Variants())).
		Msg("image deleted successfully")

	return &DeleteImageResult{
		ImageID:       imageID.String(),
		DeletedAt:     image.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		VariantsCount: len(image.Variants()),
		CleanupJobID:  cleanupJobID,
		Message:       "Image deleted and cleanup job enqueued",
	}, nil
}
