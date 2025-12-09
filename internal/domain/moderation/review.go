package moderation

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	// Maximum length for review notes.
	maxReviewNotesLength = 2000
)

// Review is an entity representing an audit trail of moderation decisions.
// It records the action taken by a moderator when reviewing a report.
//
// Business Rules:
//   - Notes must not exceed 2000 characters
//   - Review is immutable once created (no update methods)
//   - Each review is associated with a specific report
type Review struct {
	id         ReviewID
	reportID   ReportID
	reviewerID identity.UserID
	action     ReviewAction
	notes      string
	createdAt  time.Time
}

// NewReview creates a new Review for the given report with the moderator's decision.
// Returns an error if validation fails.
func NewReview(
	reportID ReportID,
	reviewerID identity.UserID,
	action ReviewAction,
	notes string,
) (*Review, error) {
	// Validate inputs
	if reportID.IsZero() {
		return nil, fmt.Errorf("report id is required")
	}
	if reviewerID.IsZero() {
		return nil, fmt.Errorf("reviewer id is required")
	}
	if !action.IsValid() {
		return nil, ErrInvalidReviewAction
	}
	if len(notes) > maxReviewNotesLength {
		return nil, ErrNotesTooLong
	}

	now := time.Now().UTC()
	review := &Review{
		id:         NewReviewID(),
		reportID:   reportID,
		reviewerID: reviewerID,
		action:     action,
		notes:      notes,
		createdAt:  now,
	}

	return review, nil
}

// ReconstructReview reconstitutes a Review from persistence without validation.
// This should only be used by the repository layer when loading from storage.
func ReconstructReview(
	id ReviewID,
	reportID ReportID,
	reviewerID identity.UserID,
	action ReviewAction,
	notes string,
	createdAt time.Time,
) *Review {
	return &Review{
		id:         id,
		reportID:   reportID,
		reviewerID: reviewerID,
		action:     action,
		notes:      notes,
		createdAt:  createdAt,
	}
}

// ID returns the review's unique identifier.
func (r *Review) ID() ReviewID {
	return r.id
}

// ReportID returns the ID of the report this review is for.
func (r *Review) ReportID() ReportID {
	return r.reportID
}

// ReviewerID returns the ID of the moderator who performed the review.
func (r *Review) ReviewerID() identity.UserID {
	return r.reviewerID
}

// Action returns the action taken by the moderator.
func (r *Review) Action() ReviewAction {
	return r.action
}

// Notes returns the review notes.
func (r *Review) Notes() string {
	return r.notes
}

// CreatedAt returns when the review was created.
func (r *Review) CreatedAt() time.Time {
	return r.createdAt
}
