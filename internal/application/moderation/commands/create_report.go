package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// CreateReportCommand represents the intent to create a new abuse report.
// It encapsulates all information needed to report inappropriate content.
type CreateReportCommand struct {
	ReporterID  string // User submitting the report
	ImageID     string // Image being reported
	Reason      string // Reason for the report (e.g., "spam", "inappropriate")
	Description string // Detailed description of the issue
}

// Implement Command interface
func (CreateReportCommand) isCommand() {}

// CreateReportResult represents the result of a successful report creation.
type CreateReportResult struct {
	ReportID string
	Status   string
}

// CreateReportHandler processes report creation commands.
// It orchestrates validation, report creation, and event publishing.
type CreateReportHandler struct {
	reports        moderation.ReportRepository
	images         gallery.ImageRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewCreateReportHandler creates a new CreateReportHandler with the given dependencies.
func NewCreateReportHandler(
	reports moderation.ReportRepository,
	images gallery.ImageRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *CreateReportHandler {
	return &CreateReportHandler{
		reports:        reports,
		images:         images,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the report creation use case.
//
// Process flow:
//  1. Parse and validate reporter ID
//  2. Parse and validate image ID
//  3. Verify image exists
//  4. Parse and validate report reason
//  5. Create Report aggregate via domain factory
//  6. Persist report via repository
//  7. Publish domain events after successful save
//
// Returns:
//   - CreateReportResult on successful creation
//   - Validation errors from domain value objects
//   - ErrImageNotFound if the image does not exist
func (h *CreateReportHandler) Handle(ctx context.Context, cmd CreateReportCommand) (*CreateReportResult, error) {
	// 1. Parse and validate reporter ID
	reporterID, err := identity.ParseUserID(cmd.ReporterID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("reporter_id", cmd.ReporterID).
			Msg("invalid reporter id during report creation")
		return nil, fmt.Errorf("invalid reporter id: %w", err)
	}

	// 2. Parse and validate image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id during report creation")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 3. Verify image exists
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found during report creation")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// Business rule: Users cannot report their own content
	if image.OwnerID().Equals(reporterID) {
		h.logger.Debug().
			Str("reporter_id", reporterID.String()).
			Str("image_id", imageID.String()).
			Msg("user attempted to report their own content")
		return nil, fmt.Errorf("cannot report your own content")
	}

	// 4. Parse and validate report reason
	reason, err := moderation.ParseReportReason(cmd.Reason)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("reason", cmd.Reason).
			Msg("invalid report reason during report creation")
		return nil, fmt.Errorf("invalid report reason: %w", err)
	}

	// 5. Create Report aggregate via domain factory
	report, err := moderation.NewReport(reporterID, imageID, reason, cmd.Description)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("reporter_id", reporterID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to create report aggregate")
		return nil, fmt.Errorf("create report: %w", err)
	}

	// 6. Persist report to repository
	if err := h.reports.Save(ctx, report); err != nil {
		h.logger.Error().
			Err(err).
			Str("report_id", report.ID().String()).
			Str("reporter_id", reporterID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to save report")
		return nil, fmt.Errorf("save report: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	for _, event := range report.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("report_id", report.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after report creation")
		}
	}
	report.ClearEvents()

	h.logger.Info().
		Str("report_id", report.ID().String()).
		Str("reporter_id", reporterID.String()).
		Str("image_id", imageID.String()).
		Str("reason", reason.String()).
		Msg("report created successfully")

	return &CreateReportResult{
		ReportID: report.ID().String(),
		Status:   report.Status().String(),
	}, nil
}
