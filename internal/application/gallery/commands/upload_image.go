package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/rs/zerolog"

	appgallery "github.com/yegamble/goimg-datalayer/internal/application/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// UploadImageCommand represents the intent to upload a new image.
// It encapsulates all information needed for image upload including
// file content, metadata, and user information.
type UploadImageCommand struct {
	UserID      string
	FileContent io.Reader
	FileSize    int64
	Filename    string
	Title       string
	Description string
	Visibility  string
	Tags        []string
	MimeType    string
	Width       int
	Height      int
}

// UploadImageResult represents the result of a successful image upload.
type UploadImageResult struct {
	ImageID string
	Status  string
	Message string
}

// UploadImageHandler processes image upload commands.
// It orchestrates validation, storage, entity creation, and job enqueueing.
type UploadImageHandler struct {
	images         gallery.ImageRepository
	storage        storage.Storage
	jobEnqueuer    appgallery.JobEnqueuer
	eventPublisher appgallery.EventPublisher
	logger         *zerolog.Logger
}

// NewUploadImageHandler creates a new UploadImageHandler with the given dependencies.
func NewUploadImageHandler(
	images gallery.ImageRepository,
	storage storage.Storage,
	jobEnqueuer appgallery.JobEnqueuer,
	eventPublisher appgallery.EventPublisher,
	logger *zerolog.Logger,
) *UploadImageHandler {
	return &UploadImageHandler{
		images:         images,
		storage:        storage,
		jobEnqueuer:    jobEnqueuer,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the image upload use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Validate file size and MIME type
//  3. Parse visibility setting
//  4. Generate storage key and upload to storage provider
//  5. Create ImageMetadata value object
//  6. Create Image aggregate (status: processing)
//  7. Parse and add tags to image
//  8. Persist image via repository
//  9. Publish domain events after successful save
//  10. Enqueue background job for image processing
//
// Returns:
//   - UploadImageResult on successful upload
//   - Validation errors from domain value objects
//   - Storage errors if upload fails
func (h *UploadImageHandler) Handle(ctx context.Context, cmd UploadImageCommand) (*UploadImageResult, error) {
	// 1. Parse and validate user ID
	ownerID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id during image upload")
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Validate file size and MIME type
	if cmd.FileSize <= 0 {
		return nil, fmt.Errorf("invalid file size: must be positive")
	}

	if cmd.FileSize > gallery.MaxFileSize {
		return nil, fmt.Errorf("%w: %d bytes exceeds maximum of %d",
			gallery.ErrFileTooLarge, cmd.FileSize, gallery.MaxFileSize)
	}

	if !gallery.SupportedMimeTypes[cmd.MimeType] {
		return nil, fmt.Errorf("%w: '%s' is not supported",
			gallery.ErrInvalidMimeType, cmd.MimeType)
	}

	// 3. Parse visibility setting (validation only - image starts as private)
	// Image defaults to private during processing and will be updated after processing completes
	if cmd.Visibility != "" {
		_, err = gallery.ParseVisibility(cmd.Visibility)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("visibility", cmd.Visibility).
				Msg("invalid visibility during image upload")
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}
	}

	// 4. Generate image ID and storage key
	imageID := h.images.NextID()
	storageKey := fmt.Sprintf("images/%s/%s/original", ownerID.String(), imageID.String())

	// Upload to storage provider
	opts := storage.DefaultPutOptions(cmd.MimeType)
	if err := h.storage.Put(ctx, storageKey, cmd.FileContent, cmd.FileSize, opts); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("storage_key", storageKey).
			Msg("failed to upload image to storage")
		return nil, fmt.Errorf("store image: %w", err)
	}

	// 5. Create ImageMetadata value object
	metadata, err := gallery.NewImageMetadata(
		cmd.Title,
		cmd.Description,
		cmd.Filename,
		cmd.MimeType,
		cmd.Width,
		cmd.Height,
		cmd.FileSize,
		storageKey,
		h.storage.Provider(),
	)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("filename", cmd.Filename).
			Msg("invalid image metadata during upload")
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// 6. Create Image aggregate via domain factory (status: processing)
	// Use NewImageWithID to ensure the image ID matches the one used for storage key
	image, err := gallery.NewImageWithID(imageID, ownerID, metadata)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("owner_id", ownerID.String()).
			Msg("failed to create image aggregate")
		return nil, fmt.Errorf("create image: %w", err)
	}

	// Set visibility if provided (defaults to private in NewImage)
	if cmd.Visibility != "" {
		// Image starts as private; visibility will be updated after processing completes
		// For now, store the desired visibility but keep it private until active
	}

	// 7. Parse and add tags to image
	for _, tagName := range cmd.Tags {
		tag, err := gallery.NewTag(tagName)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", tagName).
				Msg("invalid tag during image upload")
			return nil, fmt.Errorf("invalid tag '%s': %w", tagName, err)
		}

		if err := image.AddTag(tag); err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", tagName).
				Msg("failed to add tag to image")
			return nil, fmt.Errorf("add tag '%s': %w", tagName, err)
		}
	}

	// 8. Persist image to repository
	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Str("owner_id", ownerID.String()).
			Msg("failed to save image")
		return nil, fmt.Errorf("save image: %w", err)
	}

	// 9. Publish domain events AFTER successful save
	for _, event := range image.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", imageID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after image upload")
		}
	}
	image.ClearEvents()

	// 10. Enqueue background job for image processing
	if err := h.jobEnqueuer.EnqueueImageProcessing(ctx, imageID.String()); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to enqueue image processing job")
		// Don't fail the upload if job enqueueing fails
	}

	h.logger.Info().
		Str("image_id", imageID.String()).
		Str("owner_id", ownerID.String()).
		Str("filename", cmd.Filename).
		Int64("size", cmd.FileSize).
		Msg("image uploaded successfully")

	return &UploadImageResult{
		ImageID: imageID.String(),
		Status:  gallery.StatusProcessing.String(),
		Message: "Image uploaded and queued for processing",
	}, nil
}
