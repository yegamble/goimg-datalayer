package moderation

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// Maximum length for report description and resolution.
	maxReportTextLength = 1000
)

// Report is the aggregate root for content reports.
// It represents a user's report of inappropriate or problematic content.
//
// Business Rules:
//   - Description is required and must not exceed 1000 characters
//   - Status follows a state machine: pending -> reviewing -> (resolved | dismissed)
//   - Terminal states (resolved, dismissed) cannot transition to other states
//   - Users cannot report their own content (enforced at application layer)
//
// State Machine:
//
//	pending -> reviewing (via StartReview)
//	reviewing -> resolved (via Resolve)
//	reviewing -> dismissed (via Dismiss)
type Report struct {
	id          ReportID
	reporterID  identity.UserID
	imageID     gallery.ImageID
	reason      ReportReason
	description string
	status      ReportStatus
	resolvedBy  *identity.UserID
	resolvedAt  *time.Time
	resolution  string
	createdAt   time.Time
	events      []shared.DomainEvent
}

// NewReport creates a new Report for the given image with a reason and description.
// Returns an error if validation fails.
func NewReport(
	reporterID identity.UserID,
	imageID gallery.ImageID,
	reason ReportReason,
	description string,
) (*Report, error) {
	// Validate inputs
	if reporterID.IsZero() {
		return nil, fmt.Errorf("reporter id is required")
	}
	if imageID.IsZero() {
		return nil, fmt.Errorf("image id is required")
	}
	if !reason.IsValid() {
		return nil, ErrInvalidReportReason
	}
	if description == "" {
		return nil, ErrDescriptionEmpty
	}
	if len(description) > maxReportTextLength {
		return nil, ErrDescriptionTooLong
	}

	now := time.Now().UTC()
	report := &Report{
		id:          NewReportID(),
		reporterID:  reporterID,
		imageID:     imageID,
		reason:      reason,
		description: description,
		status:      StatusPending,
		resolvedBy:  nil,
		resolvedAt:  nil,
		resolution:  "",
		createdAt:   now,
		events:      []shared.DomainEvent{},
	}

	report.addEvent(NewReportCreated(report.id, reporterID, imageID, reason, description))
	return report, nil
}

// ReconstructReport reconstitutes a Report from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructReport(
	id ReportID,
	reporterID identity.UserID,
	imageID gallery.ImageID,
	reason ReportReason,
	description string,
	status ReportStatus,
	resolvedBy *identity.UserID,
	resolvedAt *time.Time,
	resolution string,
	createdAt time.Time,
) *Report {
	return &Report{
		id:          id,
		reporterID:  reporterID,
		imageID:     imageID,
		reason:      reason,
		description: description,
		status:      status,
		resolvedBy:  resolvedBy,
		resolvedAt:  resolvedAt,
		resolution:  resolution,
		createdAt:   createdAt,
		events:      []shared.DomainEvent{},
	}
}

// ID returns the report's unique identifier.
func (r *Report) ID() ReportID {
	return r.id
}

// ReporterID returns the ID of the user who created the report.
func (r *Report) ReporterID() identity.UserID {
	return r.reporterID
}

// ImageID returns the ID of the reported image.
func (r *Report) ImageID() gallery.ImageID {
	return r.imageID
}

// Reason returns the reason for the report.
func (r *Report) Reason() ReportReason {
	return r.reason
}

// Description returns the description of the report.
func (r *Report) Description() string {
	return r.description
}

// Status returns the current status of the report.
func (r *Report) Status() ReportStatus {
	return r.status
}

// ResolvedBy returns the ID of the user who resolved the report, if resolved.
func (r *Report) ResolvedBy() *identity.UserID {
	return r.resolvedBy
}

// ResolvedAt returns the time the report was resolved, if resolved.
func (r *Report) ResolvedAt() *time.Time {
	return r.resolvedAt
}

// Resolution returns the resolution notes, if resolved.
func (r *Report) Resolution() string {
	return r.resolution
}

// CreatedAt returns when the report was created.
func (r *Report) CreatedAt() time.Time {
	return r.createdAt
}

// Events returns the domain events that have occurred on this aggregate.
func (r *Report) Events() []shared.DomainEvent {
	return r.events
}

// ClearEvents clears all domain events from this aggregate.
// This should be called after events have been dispatched.
func (r *Report) ClearEvents() {
	r.events = []shared.DomainEvent{}
}

// StartReview transitions the report from pending to reviewing status.
// Returns an error if the report is not in pending status or is in a terminal state.
func (r *Report) StartReview() error {
	if r.status.IsTerminal() {
		return ErrReportInTerminalState
	}

	if r.status != StatusPending {
		// Already reviewing or in another state
		return nil
	}

	r.status = StatusReviewing
	r.addEvent(NewReportReviewStarted(r.id))
	return nil
}

// Resolve marks the report as resolved with the given resolution notes.
// The resolverID identifies the moderator who resolved the report.
// Returns an error if the report is not in reviewing status or is already resolved.
func (r *Report) Resolve(resolverID identity.UserID, resolution string) error {
	if r.status.IsTerminal() {
		if r.status == StatusResolved {
			return ErrReportAlreadyResolved
		}
		return ErrReportInTerminalState
	}

	if resolverID.IsZero() {
		return fmt.Errorf("resolver id is required")
	}
	if resolution == "" {
		return ErrResolutionRequired
	}
	if len(resolution) > maxReportTextLength {
		return ErrResolutionTooLong
	}

	now := time.Now().UTC()
	r.status = StatusResolved
	r.resolvedBy = &resolverID
	r.resolvedAt = &now
	r.resolution = resolution

	r.addEvent(NewReportResolved(r.id, resolverID, resolution))
	return nil
}

// Dismiss marks the report as dismissed (invalid or unfounded).
// The resolverID identifies the moderator who dismissed the report.
// Returns an error if the report is not in reviewing status or is already dismissed.
func (r *Report) Dismiss(resolverID identity.UserID) error {
	if r.status.IsTerminal() {
		if r.status == StatusDismissed {
			return ErrReportAlreadyDismissed
		}
		return ErrReportInTerminalState
	}

	if resolverID.IsZero() {
		return fmt.Errorf("resolver id is required")
	}

	now := time.Now().UTC()
	r.status = StatusDismissed
	r.resolvedBy = &resolverID
	r.resolvedAt = &now
	r.resolution = "Report dismissed"

	r.addEvent(NewReportDismissed(r.id, resolverID))
	return nil
}

// addEvent adds a domain event to the aggregate's event list.
func (r *Report) addEvent(event shared.DomainEvent) {
	r.events = append(r.events, event)
}
