package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// DismissReportCommand represents the intent to dismiss a report as invalid or unfounded.
// This transitions the report to dismissed status.
type DismissReportCommand struct {
	ReportID   string // Report to dismiss
	ResolverID string // Moderator dismissing the report
}

// Implement Command interface
func (DismissReportCommand) isCommand() {}

// DismissReportResult represents the result of successfully dismissing a report.
type DismissReportResult struct {
	ReportID string
	Status   string
}

// DismissReportHandler processes report dismissal commands.
// It orchestrates the workflow of dismissing an invalid or unfounded report.
type DismissReportHandler struct {
	reports        moderation.ReportRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewDismissReportHandler creates a new DismissReportHandler with the given dependencies.
func NewDismissReportHandler(
	reports moderation.ReportRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *DismissReportHandler {
	return &DismissReportHandler{
		reports:        reports,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the dismiss report use case.
//
// Process flow:
//  1. Parse and validate report ID
//  2. Parse and validate resolver ID
//  3. Load report aggregate from repository
//  4. Call domain method to dismiss
//  5. Persist updated report
//  6. Publish domain events after successful save
//
// Returns:
//   - DismissReportResult on successful dismissal
//   - ErrReportNotFound if the report does not exist
//   - ErrReportAlreadyDismissed if the report is already dismissed
//   - ErrReportInTerminalState if the report is already resolved
func (h *DismissReportHandler) Handle(ctx context.Context, cmd DismissReportCommand) (*DismissReportResult, error) {
	// 1. Parse and validate report ID
	reportID, err := moderation.ParseReportID(cmd.ReportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", cmd.ReportID).
			Msg("invalid report id during dismiss")
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	// 2. Parse and validate resolver ID
	resolverID, err := identity.ParseUserID(cmd.ResolverID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("resolver_id", cmd.ResolverID).
			Msg("invalid resolver id during dismiss")
		return nil, fmt.Errorf("invalid resolver id: %w", err)
	}

	// 3. Load report aggregate
	report, err := h.reports.FindByID(ctx, reportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Msg("report not found during dismiss")
		return nil, fmt.Errorf("find report: %w", err)
	}

	// 4. Dismiss via domain method
	if err := report.Dismiss(resolverID); err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Str("resolver_id", resolverID.String()).
			Str("current_status", report.Status().String()).
			Msg("failed to dismiss report")
		return nil, fmt.Errorf("dismiss report: %w", err)
	}

	// 5. Persist updated report
	if err := h.reports.Save(ctx, report); err != nil {
		h.logger.Error().
			Err(err).
			Str("report_id", reportID.String()).
			Str("resolver_id", resolverID.String()).
			Msg("failed to save report after dismissal")
		return nil, fmt.Errorf("save report: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	for _, event := range report.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("report_id", reportID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after report dismissal")
		}
	}
	report.ClearEvents()

	h.logger.Info().
		Str("report_id", reportID.String()).
		Str("resolver_id", resolverID.String()).
		Msg("report dismissed successfully")

	return &DismissReportResult{
		ReportID: reportID.String(),
		Status:   report.Status().String(),
	}, nil
}
