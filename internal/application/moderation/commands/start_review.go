package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appmoderation "github.com/yegamble/goimg-datalayer/internal/application/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// StartReviewCommand represents the intent to start reviewing a pending report.
// This transitions the report from pending to reviewing status.
type StartReviewCommand struct {
	ReportID string // Report to start reviewing
}

// Implement Command interface
func (StartReviewCommand) isCommand() {}

// StartReviewResult represents the result of successfully starting a review.
type StartReviewResult struct {
	ReportID string
	Status   string
}

// StartReviewHandler processes start review commands.
// It orchestrates the workflow of transitioning a report to reviewing status.
type StartReviewHandler struct {
	reports        moderation.ReportRepository
	eventPublisher appmoderation.EventPublisher
	logger         *zerolog.Logger
}

// NewStartReviewHandler creates a new StartReviewHandler with the given dependencies.
func NewStartReviewHandler(
	reports moderation.ReportRepository,
	eventPublisher appmoderation.EventPublisher,
	logger *zerolog.Logger,
) *StartReviewHandler {
	return &StartReviewHandler{
		reports:        reports,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the start review use case.
//
// Process flow:
//  1. Parse and validate report ID
//  2. Load report aggregate from repository
//  3. Call domain method to start review
//  4. Persist updated report
//  5. Publish domain events after successful save
//
// Returns:
//   - StartReviewResult on successful status transition
//   - ErrReportNotFound if the report does not exist
//   - ErrReportInTerminalState if the report is already resolved/dismissed
func (h *StartReviewHandler) Handle(ctx context.Context, cmd StartReviewCommand) (*StartReviewResult, error) {
	// 1. Parse and validate report ID
	reportID, err := moderation.ParseReportID(cmd.ReportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", cmd.ReportID).
			Msg("invalid report id during start review")
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	// 2. Load report aggregate
	report, err := h.reports.FindByID(ctx, reportID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Msg("report not found during start review")
		return nil, fmt.Errorf("find report: %w", err)
	}

	// 3. Start review via domain method
	if err := report.StartReview(); err != nil {
		h.logger.Debug().
			Err(err).
			Str("report_id", reportID.String()).
			Str("current_status", report.Status().String()).
			Msg("failed to start review")
		return nil, fmt.Errorf("start review: %w", err)
	}

	// 4. Persist updated report
	if err := h.reports.Save(ctx, report); err != nil {
		h.logger.Error().
			Err(err).
			Str("report_id", reportID.String()).
			Msg("failed to save report after starting review")
		return nil, fmt.Errorf("save report: %w", err)
	}

	// 5. Publish domain events AFTER successful save
	for _, event := range report.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("report_id", reportID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after starting review")
		}
	}
	report.ClearEvents()

	h.logger.Info().
		Str("report_id", reportID.String()).
		Msg("review started successfully")

	return &StartReviewResult{
		ReportID: reportID.String(),
		Status:   report.Status().String(),
	}, nil
}
