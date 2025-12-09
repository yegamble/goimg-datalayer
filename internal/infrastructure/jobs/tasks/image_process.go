//go:build cgo

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/processor"
)

const (
	// TypeImageProcess is the task type for image processing.
	TypeImageProcess = "image:process"

	// DefaultMaxRetry is the default number of retry attempts for image processing.
	DefaultMaxRetry = 3

	// DefaultTimeout is the default timeout for image processing.
	DefaultTimeout = 5 * time.Minute
)

// ImageProcessPayload contains the data needed to process an image.
type ImageProcessPayload struct {
	// ImageID is the unique identifier for the image.
	ImageID string `json:"image_id"`

	// StorageKey is the key where the original image is stored.
	StorageKey string `json:"storage_key"`

	// OriginalFilename is the original filename of the uploaded image.
	OriginalFilename string `json:"original_filename"`

	// OwnerID is the user who uploaded the image.
	OwnerID string `json:"owner_id"`

	// EnqueuedAt is when the task was enqueued.
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// ImageProcessHandler handles image processing tasks.
// It generates image variants (thumbnail, small, medium, large) using bimg/libvips.
type ImageProcessHandler struct {
	processor *processor.Processor
	storage   Storage
	logger    zerolog.Logger
}

// NewImageProcessHandler creates a new image processing task handler.
func NewImageProcessHandler(
	proc *processor.Processor,
	storage Storage,
	logger zerolog.Logger,
) *ImageProcessHandler {
	return &ImageProcessHandler{
		processor: proc,
		storage:   storage,
		logger:    logger,
	}
}

// ProcessTask implements asynq.Handler interface.
// It processes the image and generates all variants.
func (h *ImageProcessHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// Parse payload
	var payload ImageProcessPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.logger.Error().
			Err(err).
			Str("task_type", t.Type()).
			Msg("failed to unmarshal image process payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	startTime := time.Now()
	h.logger.Info().
		Str("image_id", payload.ImageID).
		Str("storage_key", payload.StorageKey).
		Str("owner_id", payload.OwnerID).
		Msg("starting image processing")

	// Step 1: Retrieve original image from storage
	imageData, err := h.storage.Get(ctx, payload.StorageKey)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", payload.ImageID).
			Str("storage_key", payload.StorageKey).
			Msg("failed to retrieve image from storage")
		return fmt.Errorf("retrieve image %s: %w", payload.StorageKey, err)
	}

	h.logger.Debug().
		Str("image_id", payload.ImageID).
		Int("size_bytes", len(imageData)).
		Msg("retrieved image from storage")

	// Step 2: Process image and generate variants
	result, err := h.processor.Process(ctx, imageData, payload.OriginalFilename)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", payload.ImageID).
			Str("filename", payload.OriginalFilename).
			Msg("failed to process image")
		return fmt.Errorf("process image %s: %w", payload.ImageID, err)
	}

	h.logger.Info().
		Str("image_id", payload.ImageID).
		Int("original_width", result.OriginalWidth).
		Int("original_height", result.OriginalHeight).
		Str("format", result.OriginalFormat).
		Msg("image processed successfully")

	// Step 3: Store variants
	variants := map[string][]byte{
		"thumbnail": result.Thumbnail.Data,
		"small":     result.Small.Data,
		"medium":    result.Medium.Data,
		"large":     result.Large.Data,
		"original":  result.Original.Data,
	}

	for variantType, variantData := range variants {
		variantKey := h.buildVariantKey(payload.ImageID, variantType)

		if err := h.storage.Put(ctx, variantKey, variantData); err != nil {
			h.logger.Error().
				Err(err).
				Str("image_id", payload.ImageID).
				Str("variant", variantType).
				Str("storage_key", variantKey).
				Msg("failed to store variant")
			return fmt.Errorf("store variant %s: %w", variantType, err)
		}

		h.logger.Debug().
			Str("image_id", payload.ImageID).
			Str("variant", variantType).
			Int("size_bytes", len(variantData)).
			Msg("stored image variant")
	}

	duration := time.Since(startTime)
	h.logger.Info().
		Str("image_id", payload.ImageID).
		Dur("duration_ms", duration).
		Int("variants_count", len(variants)).
		Msg("image processing completed")

	return nil
}

// buildVariantKey constructs the storage key for an image variant.
// Format: images/{image_id}/{variant}.webp.
func (h *ImageProcessHandler) buildVariantKey(imageID, variant string) string {
	// Determine file extension based on variant
	ext := "webp" // Default for processed variants
	if variant == "original" {
		ext = "jpg" // Original might be different format, but we re-encode
	}

	return fmt.Sprintf("images/%s/%s.%s", imageID, variant, ext)
}

// NewImageProcessTask creates a new image processing task with default options.
func NewImageProcessTask(payload ImageProcessPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return asynq.NewTask(
		TypeImageProcess,
		payloadBytes,
		asynq.MaxRetry(DefaultMaxRetry),
		asynq.Timeout(DefaultTimeout),
		asynq.Queue("default"),
	), nil
}

// NewImageProcessTaskWithOptions creates a new image processing task with custom options.
func NewImageProcessTaskWithOptions(payload ImageProcessPayload, opts ...asynq.Option) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Apply default options first
	defaultOpts := []asynq.Option{
		asynq.MaxRetry(DefaultMaxRetry),
		asynq.Timeout(DefaultTimeout),
		asynq.Queue("default"),
	}

	// Append custom options (they override defaults)
	defaultOpts = append(defaultOpts, opts...)

	return asynq.NewTask(TypeImageProcess, payloadBytes, defaultOpts...), nil
}
