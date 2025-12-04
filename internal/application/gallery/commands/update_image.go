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

// UpdateImageCommand represents the intent to update image metadata.
// It encapsulates all information needed to update an image's title,
// description, visibility, and tags.
type UpdateImageCommand struct {
	ImageID     string
	UserID      string
	Title       *string  // Pointer allows nil to mean "no change"
	Description *string  // Pointer allows nil to mean "no change"
	Visibility  *string  // Pointer allows nil to mean "no change"
	Tags        []string // If non-nil, replaces all existing tags
}

// Implement Command interface
func (UpdateImageCommand) isCommand() {}

// UpdateImageResult represents the result of a successful image update.
type UpdateImageResult struct {
	ImageID    string
	UpdatedAt  string
	Message    string
	TagsAdded  int
	TagsRemove int
}

// UpdateImageHandler processes image update commands.
// It orchestrates ownership validation, metadata updates, and event publishing.
type UpdateImageHandler struct {
	images         gallery.ImageRepository
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewUpdateImageHandler creates a new UpdateImageHandler with the given dependencies.
func NewUpdateImageHandler(
	images gallery.ImageRepository,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *UpdateImageHandler {
	return &UpdateImageHandler{
		images:         images,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the image update use case.
//
// Process flow:
//  1. Parse and validate image ID and user ID
//  2. Load image from repository
//  3. Verify ownership (user must own the image)
//  4. Update metadata if provided (title, description)
//  5. Update visibility if provided
//  6. Update tags if provided (replace all existing tags)
//  7. Persist updated image
//  8. Publish domain events after successful save
//
// Returns:
//   - UpdateImageResult on successful update
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if user doesn't own image
//   - Validation errors from domain value objects
func (h *UpdateImageHandler) Handle(ctx context.Context, cmd UpdateImageCommand) (*UpdateImageResult, error) {
	// 1. Parse and validate IDs
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during update")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during image update")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during update")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for update")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Verify ownership
	if !image.IsOwnedBy(userID) {
		h.logger.Warn().
			Str("image_id", imageID.String()).
			Str("user_id", userID.String()).
			Str("owner_id", image.OwnerID().String()).
			Msg("unauthorized image update attempt")
		return nil, gallery.ErrUnauthorizedAccess
	}

	// Track changes for result
	var updatedFields []string

	// 4. Update metadata if provided
	if cmd.Title != nil || cmd.Description != nil {
		title := image.Metadata().Title()
		description := image.Metadata().Description()

		if cmd.Title != nil {
			title = *cmd.Title
			updatedFields = append(updatedFields, "title")
		}
		if cmd.Description != nil {
			description = *cmd.Description
			updatedFields = append(updatedFields, "description")
		}

		if err := image.UpdateMetadata(title, description); err != nil {
			h.logger.Debug().
				Err(err).
				Str("image_id", imageID.String()).
				Msg("invalid metadata during image update")
			return nil, fmt.Errorf("update metadata: %w", err)
		}
	}

	// 5. Update visibility if provided
	if cmd.Visibility != nil {
		visibility, err := gallery.ParseVisibility(*cmd.Visibility)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("visibility", *cmd.Visibility).
				Msg("invalid visibility during image update")
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}

		if err := image.UpdateVisibility(visibility); err != nil {
			h.logger.Debug().
				Err(err).
				Str("image_id", imageID.String()).
				Str("visibility", visibility.String()).
				Msg("failed to update visibility")
			return nil, fmt.Errorf("update visibility: %w", err)
		}
		updatedFields = append(updatedFields, "visibility")
	}

	// 6. Update tags if provided (replace all existing tags)
	var tagsAdded, tagsRemoved int
	if cmd.Tags != nil {
		// Remove all existing tags
		existingTags := image.Tags()
		for _, tag := range existingTags {
			if err := image.RemoveTag(tag); err != nil {
				h.logger.Debug().
					Err(err).
					Str("tag", tag.String()).
					Msg("failed to remove existing tag")
				// Continue removing other tags
			} else {
				tagsRemoved++
			}
		}

		// Add new tags
		for _, tagName := range cmd.Tags {
			tag, err := gallery.NewTag(tagName)
			if err != nil {
				h.logger.Debug().
					Err(err).
					Str("tag", tagName).
					Msg("invalid tag during image update")
				return nil, fmt.Errorf("invalid tag '%s': %w", tagName, err)
			}

			if err := image.AddTag(tag); err != nil {
				h.logger.Debug().
					Err(err).
					Str("tag", tagName).
					Msg("failed to add tag to image")
				return nil, fmt.Errorf("add tag '%s': %w", tagName, err)
			}
			tagsAdded++
		}

		if tagsAdded > 0 || tagsRemoved > 0 {
			updatedFields = append(updatedFields, "tags")
		}
	}

	// Check if any updates were made
	if len(updatedFields) == 0 {
		h.logger.Debug().
			Str("image_id", imageID.String()).
			Msg("no fields updated")
		return &UpdateImageResult{
			ImageID:   imageID.String(),
			UpdatedAt: image.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
			Message:   "No changes made",
		}, nil
	}

	// 7. Persist updated image
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to save updated image")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 8. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after image update")
		}
	}
	image.ClearEvents()

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("user_id", userID.String()).
		Strs("updated_fields", updatedFields).
		Msg("image updated successfully")

	return &UpdateImageResult{
		ImageID:    imageID.String(),
		UpdatedAt:  image.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Message:    fmt.Sprintf("Updated: %v", updatedFields),
		TagsAdded:  tagsAdded,
		TagsRemove: tagsRemoved,
	}, nil
}
