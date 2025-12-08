package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

const (
	// TypeImageScan is the task type for malware scanning.
	TypeImageScan = "image:scan"

	// DefaultScanMaxRetry is the default number of retry attempts for malware scanning.
	DefaultScanMaxRetry = 2

	// DefaultScanTimeout is the default timeout for malware scanning.
	DefaultScanTimeout = 2 * time.Minute
)

// ImageScanPayload contains the data needed to scan an image for malware.
type ImageScanPayload struct {
	// ImageID is the unique identifier for the image.
	ImageID string `json:"image_id"`

	// StorageKey is the key where the image is stored.
	StorageKey string `json:"storage_key"`

	// OriginalFilename is the original filename of the uploaded image.
	OriginalFilename string `json:"original_filename"`

	// OwnerID is the user who uploaded the image.
	OwnerID string `json:"owner_id"`

	// EnqueuedAt is when the task was enqueued.
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// ImageScanHandler handles malware scanning tasks using ClamAV.
// It scans images for viruses, polyglot files, and other malicious content.
type ImageScanHandler struct {
	scanner clamav.Scanner
	storage Storage
	logger  zerolog.Logger
}

// NewImageScanHandler creates a new malware scanning task handler.
func NewImageScanHandler(
	scanner clamav.Scanner,
	storage Storage,
	logger zerolog.Logger,
) *ImageScanHandler {
	return &ImageScanHandler{
		scanner: scanner,
		storage: storage,
		logger:  logger,
	}
}

// ProcessTask implements asynq.Handler interface.
// It scans the image for malware using ClamAV.
func (h *ImageScanHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// Parse payload
	var payload ImageScanPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.logger.Error().
			Err(err).
			Str("task_type", t.Type()).
			Msg("failed to unmarshal image scan payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	startTime := time.Now()
	h.logger.Info().
		Str("image_id", payload.ImageID).
		Str("storage_key", payload.StorageKey).
		Str("owner_id", payload.OwnerID).
		Msg("starting malware scan")

	// Step 1: Verify ClamAV daemon is responsive
	if err := h.scanner.Ping(ctx); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", payload.ImageID).
			Msg("clamav daemon not responding")
		return fmt.Errorf("clamav ping failed: %w", err)
	}

	// Step 2: Retrieve image from storage
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
		Msg("retrieved image for scanning")

	// Step 3: Scan image for malware
	scanResult, err := h.scanner.Scan(ctx, imageData)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", payload.ImageID).
			Str("filename", payload.OriginalFilename).
			Msg("malware scan failed")
		return fmt.Errorf("scan image %s: %w", payload.ImageID, err)
	}

	duration := time.Since(startTime)

	// Step 4: Handle scan results
	if scanResult.Infected {
		// Malware detected - this is a terminal error
		h.logger.Warn().
			Str("image_id", payload.ImageID).
			Str("owner_id", payload.OwnerID).
			Str("virus", scanResult.Virus).
			Str("filename", payload.OriginalFilename).
			Dur("scan_duration_ms", duration).
			Msg("malware detected in image")

		// TODO: Update image status to "infected" in database
		// TODO: Notify user via email/notification
		// TODO: Delete infected file from storage
		// TODO: Increment user's infected file counter

		return fmt.Errorf("malware detected: %s", scanResult.Virus)
	}

	// Clean scan result
	h.logger.Info().
		Str("image_id", payload.ImageID).
		Time("scanned_at", scanResult.ScannedAt).
		Dur("scan_duration_ms", duration).
		Msg("image scan completed - no threats found")

	// TODO: Update image status to "clean" in database

	return nil
}

// NewImageScanTask creates a new malware scanning task with default options.
func NewImageScanTask(payload ImageScanPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return asynq.NewTask(
		TypeImageScan,
		payloadBytes,
		asynq.MaxRetry(DefaultScanMaxRetry),
		asynq.Timeout(DefaultScanTimeout),
		asynq.Queue("default"),
	), nil
}

// NewImageScanTaskWithOptions creates a new malware scanning task with custom options.
func NewImageScanTaskWithOptions(payload ImageScanPayload, opts ...asynq.Option) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Apply default options first
	defaultOpts := []asynq.Option{
		asynq.MaxRetry(DefaultScanMaxRetry),
		asynq.Timeout(DefaultScanTimeout),
		asynq.Queue("default"),
	}

	// Append custom options (they override defaults)
	defaultOpts = append(defaultOpts, opts...)

	return asynq.NewTask(TypeImageScan, payloadBytes, defaultOpts...), nil
}
