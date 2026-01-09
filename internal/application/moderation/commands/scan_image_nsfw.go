package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/nsfw"
)

// ScanImageNSFWCommand represents the intent to scan an image for NSFW content.
// This command triggers the AI-based NSFW detection process.
type ScanImageNSFWCommand struct {
	ImageID  string // Image to scan
	ImageURL string // URL of the image to scan
	Force    bool   // If true, scan even if a recent scan exists
}

// Implement Command interface.
func (ScanImageNSFWCommand) isCommand() {}

// ScanImageNSFWResult represents the result of an NSFW scan.
type ScanImageNSFWResult struct {
	ScanID         string  `json:"scan_id"`
	Status         string  `json:"status"`
	Category       string  `json:"category"`
	Score          float64 `json:"score"`
	IsNSFW         bool    `json:"is_nsfw"`
	RequiresReview bool    `json:"requires_review"`
	Provider       string  `json:"provider"`
}

// ScanImageNSFWHandler processes NSFW scan commands.
// It orchestrates the scan process using configured providers.
type ScanImageNSFWHandler struct {
	scans          moderation.NSFWScanRepository
	images         gallery.ImageRepository
	nsfwClient     nsfw.Client
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewScanImageNSFWHandler creates a new ScanImageNSFWHandler with the given dependencies.
func NewScanImageNSFWHandler(
	scans moderation.NSFWScanRepository,
	images gallery.ImageRepository,
	nsfwClient nsfw.Client,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *ScanImageNSFWHandler {
	return &ScanImageNSFWHandler{
		scans:          scans,
		images:         images,
		nsfwClient:     nsfwClient,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the NSFW scan use case.
func (h *ScanImageNSFWHandler) Handle(
	ctx context.Context,
	cmd ScanImageNSFWCommand,
) (*ScanImageNSFWResult, error) {
	// Validate and prepare.
	imageID, err := h.validateImage(ctx, cmd)
	if err != nil {
		return nil, err
	}

	// Check for existing active scan.
	if err := h.checkActiveScan(ctx, cmd, imageID); err != nil {
		return nil, err
	}

	// Create and persist initial scan.
	scan, err := h.createAndPersistScan(ctx, imageID)
	if err != nil {
		return nil, err
	}

	// Execute NSFW detection.
	result, err := h.nsfwClient.Scan(ctx, cmd.ImageURL)
	if err != nil {
		return nil, h.handleScanFailure(ctx, scan, err)
	}

	// Complete scan with results.
	if err := h.completeScan(ctx, scan, result); err != nil {
		return nil, err
	}

	return h.buildResult(scan), nil
}

// validateImage validates the image ID and verifies the image exists.
func (h *ScanImageNSFWHandler) validateImage(
	ctx context.Context,
	cmd ScanImageNSFWCommand,
) (gallery.ImageID, error) {
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during NSFW scan")
		return gallery.ImageID{}, fmt.Errorf("invalid image id: %w", err)
	}

	_, err = h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found during NSFW scan")
		return gallery.ImageID{}, fmt.Errorf("find image: %w", err)
	}

	return imageID, nil
}

// checkActiveScan checks if an active scan already exists for the image.
func (h *ScanImageNSFWHandler) checkActiveScan(
	ctx context.Context,
	cmd ScanImageNSFWCommand,
	imageID gallery.ImageID,
) error {
	if cmd.Force {
		return nil
	}

	hasActive, err := h.scans.HasActiveScan(ctx, imageID)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to check for active scan")
		return nil // Continue despite check failure
	}

	if hasActive {
		h.logger.Debug().
			Str("image_id", imageID.String()).
			Msg("image has active NSFW scan, skipping")
		return fmt.Errorf("image already has an active scan in progress")
	}

	return nil
}

// createAndPersistScan creates a new NSFW scan and persists it.
func (h *ScanImageNSFWHandler) createAndPersistScan(
	ctx context.Context,
	imageID gallery.ImageID,
) (*moderation.NSFWScan, error) {
	scanID := h.scans.NextID()
	provider := h.nsfwClient.Provider()
	scan := moderation.NewNSFWScan(scanID, imageID, provider)

	if err := scan.MarkScanning(); err != nil {
		return nil, fmt.Errorf("mark scanning: %w", err)
	}

	if err := h.scans.Save(ctx, scan); err != nil {
		h.logger.Error().
			Err(err).
			Str("scan_id", scanID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to save initial NSFW scan")
		return nil, fmt.Errorf("save scan: %w", err)
	}

	return scan, nil
}

// handleScanFailure marks the scan as failed and returns an error.
func (h *ScanImageNSFWHandler) handleScanFailure(
	ctx context.Context,
	scan *moderation.NSFWScan,
	scanErr error,
) error {
	if failErr := scan.Fail(scanErr.Error()); failErr != nil {
		h.logger.Warn().
			Err(failErr).
			Str("scan_id", scan.ID().String()).
			Msg("failed to mark scan as failed")
	}

	if saveErr := h.scans.Save(ctx, scan); saveErr != nil {
		h.logger.Error().
			Err(saveErr).
			Str("scan_id", scan.ID().String()).
			Msg("failed to save failed scan state")
	}

	h.publishEvents(ctx, scan)
	return fmt.Errorf("NSFW scan failed: %w", scanErr)
}

// completeScan marks the scan as completed with results.
func (h *ScanImageNSFWHandler) completeScan(
	ctx context.Context,
	scan *moderation.NSFWScan,
	result *nsfw.ScanResult,
) error {
	if err := scan.Complete(result.Category, result.Score, result.Details); err != nil {
		h.logger.Error().
			Err(err).
			Str("scan_id", scan.ID().String()).
			Msg("failed to complete NSFW scan")
		return fmt.Errorf("complete scan: %w", err)
	}

	if err := h.scans.Save(ctx, scan); err != nil {
		h.logger.Error().
			Err(err).
			Str("scan_id", scan.ID().String()).
			Msg("failed to save completed NSFW scan")
		return fmt.Errorf("save completed scan: %w", err)
	}

	h.publishEvents(ctx, scan)

	h.logger.Info().
		Str("scan_id", scan.ID().String()).
		Str("image_id", scan.ImageID().String()).
		Str("category", scan.Category().String()).
		Float64("score", scan.Score()).
		Bool("is_nsfw", scan.IsNSFW()).
		Msg("NSFW scan completed successfully")

	return nil
}

// buildResult creates the result DTO from a completed scan.
func (h *ScanImageNSFWHandler) buildResult(scan *moderation.NSFWScan) *ScanImageNSFWResult {
	return &ScanImageNSFWResult{
		ScanID:         scan.ID().String(),
		Status:         scan.Status().String(),
		Category:       scan.Category().String(),
		Score:          scan.Score(),
		IsNSFW:         scan.IsNSFW(),
		RequiresReview: scan.RequiresReview(),
		Provider:       scan.Provider().String(),
	}
}

// publishEvents publishes all pending domain events for a scan.
func (h *ScanImageNSFWHandler) publishEvents(ctx context.Context, scan *moderation.NSFWScan) {
	for _, event := range scan.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("scan_id", scan.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish NSFW scan domain event")
		}
	}
	scan.ClearEvents()
}
