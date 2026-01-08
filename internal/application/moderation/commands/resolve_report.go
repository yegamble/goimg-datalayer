package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// ResolveReportCommand represents the intent to resolve a report with a moderator action.
// This transitions the report to resolved status and records the action taken.
type ResolveReportCommand struct {
	ReportID   string // Report to resolve
	ResolverID string // Moderator resolving the report
	Resolution string // Resolution notes explaining the action taken
}

// Implement Command interface
func (ResolveReportCommand) isCommand() {}

// ResolveReportResult represents the result of successfully resolving a report.
type ResolveReportResult struct {
	ReportID   string
	Status     string
	Resolution string
}

// ResolveReportHandler processes report resolution commands.
// It orchestrates the workflow of resolving a report with moderator action.
type ResolveReportHandler struct {
	reports        moderation.ReportRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewResolveReportHandler creates a new ResolveReportHandler with the given dependencies.
func NewResolveReportHandler(
	reports moderation.ReportRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *ResolveReportHandler {
	return &ResolveReportHandler{
		reports:        reports,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the resolve report use case.
//
// Process flow:
//  1. Parse and validate report ID
//  2. Parse and validate resolver ID
//  3. Load report aggregate from repository
//  4. Call domain method to resolve with action
//  5. Persist updated report
//  6. Publish domain events after successful save
//
// Returns:
//   - ResolveReportResult on successful resolution
//   - ErrReportNotFound if the report does not exist
//   - ErrReportAlreadyResolved if the report is already resolved
//   - ErrReportInTerminalState if the report is dismissed
//   - ErrResolutionRequired if resolution notes are empty
func (h *ResolveReportHandler) Handle(ctx context.Context, cmd ResolveReportCommand) (*ResolveReportResult, error) {
	// 1. Parse and validate report ID
	reportID, err := moderation.ParseReportID(cmd.ReportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", cmd.ReportID).
			Msg("invalid report id during resolve")
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	// 2. Parse and validate resolver ID
	resolverID, err := identity.ParseUserID(cmd.ResolverID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("resolver_id", cmd.ResolverID).
			Msg("invalid resolver id during resolve")
		return nil, fmt.Errorf("invalid resolver id: %w", err)
	}

	// 3. Load report aggregate
	report, err := h.reports.FindByID(ctx, reportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Msg("report not found during resolve")
		return nil, fmt.Errorf("find report: %w", err)
	}

	// 4. Resolve via domain method
	if err := report.Resolve(resolverID, cmd.Resolution); err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Str("resolver_id", resolverID.String()).
			Str("current_status", report.Status().String()).
			Msg("failed to resolve report")
		return nil, fmt.Errorf("resolve report: %w", err)
	}

	// 5. Persist updated report
	if err := h.reports.Save(ctx, report); err != nil {
		h.logger.Error().
			Err(err).
			Str("report_id", reportID.String()).
			Str("resolver_id", resolverID.String()).
			Msg("failed to save report after resolution")
		return nil, fmt.Errorf("save report: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	for _, event := range report.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("report_id", reportID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after report resolution")
		}
	}
	report.ClearEvents()

	h.logger.Info().
		Str("report_id", reportID.String()).
		Str("resolver_id", resolverID.String()).
		Msg("report resolved successfully")

	return &ResolveReportResult{
		ReportID:   reportID.String(),
		Status:     report.Status().String(),
		Resolution: report.Resolution(),
	}, nil
}
