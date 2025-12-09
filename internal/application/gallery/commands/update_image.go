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
//
//nolint:cyclop // Command handler requires sequential validation: image ID, user ID, ownership, and multiple field updates
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
	if err := h.updateMetadata(image, cmd, &updatedFields); err != nil {
		return nil, err
	}

	// 5. Update visibility if provided
	if err := h.updateVisibility(image, imageID, cmd, &updatedFields); err != nil {
		return nil, err
	}

	// 6. Update tags if provided (replace all existing tags)
	tagsAdded, tagsRemoved, err := h.updateImageTags(image, cmd.Tags)
	if err != nil {
		return nil, err
	}
	if tagsAdded > 0 || tagsRemoved > 0 {
		updatedFields = append(updatedFields, "tags")
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

// updateMetadata updates the image metadata (title and description) if provided.
func (h *UpdateImageHandler) updateMetadata(image *gallery.Image, cmd UpdateImageCommand, updatedFields *[]string) error {
	if cmd.Title == nil && cmd.Description == nil {
		return nil
	}

	title := image.Metadata().Title()
	description := image.Metadata().Description()

	if cmd.Title != nil {
		title = *cmd.Title
		*updatedFields = append(*updatedFields, "title")
	}
	if cmd.Description != nil {
		description = *cmd.Description
		*updatedFields = append(*updatedFields, "description")
	}

	if err := image.UpdateMetadata(title, description); err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", image.ID().String()).
			Msg("invalid metadata during image update")
		return fmt.Errorf("update metadata: %w", err)
	}

	return nil
}

// updateVisibility updates the image visibility if provided.
func (h *UpdateImageHandler) updateVisibility(image *gallery.Image, imageID gallery.ImageID, cmd UpdateImageCommand, updatedFields *[]string) error {
	if cmd.Visibility == nil {
		return nil
	}

	visibility, err := gallery.ParseVisibility(*cmd.Visibility)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("visibility", *cmd.Visibility).
			Msg("invalid visibility during image update")
		return fmt.Errorf("invalid visibility: %w", err)
	}

	if err := image.UpdateVisibility(visibility); err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Str("visibility", visibility.String()).
			Msg("failed to update visibility")
		return fmt.Errorf("update visibility: %w", err)
	}

	*updatedFields = append(*updatedFields, "visibility")
	return nil
}

// updateImageTags replaces all tags on an image with new tags.
// Returns the number of tags added and removed.
func (h *UpdateImageHandler) updateImageTags(image *gallery.Image, newTags []string) (int, int, error) {
	if newTags == nil {
		return 0, 0, nil
	}

	// Remove all existing tags
	tagsRemoved := h.removeAllTags(image)

	// Add new tags
	tagsAdded, err := h.addNewTags(image, newTags)
	if err != nil {
		return 0, tagsRemoved, err
	}

	return tagsAdded, tagsRemoved, nil
}

// removeAllTags removes all existing tags from an image.
func (h *UpdateImageHandler) removeAllTags(image *gallery.Image) int {
	existingTags := image.Tags()
	tagsRemoved := 0

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

	return tagsRemoved
}

// addNewTags adds new tags to an image.
func (h *UpdateImageHandler) addNewTags(image *gallery.Image, tagNames []string) (int, error) {
	tagsAdded := 0

	for _, tagName := range tagNames {
		tag, err := gallery.NewTag(tagName)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", tagName).
				Msg("invalid tag during image update")
			return tagsAdded, fmt.Errorf("invalid tag '%s': %w", tagName, err)
		}

		if err := image.AddTag(tag); err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", tagName).
				Msg("failed to add tag to image")
			return tagsAdded, fmt.Errorf("add tag '%s': %w", tagName, err)
		}
		tagsAdded++
	}

	return tagsAdded, nil
}
