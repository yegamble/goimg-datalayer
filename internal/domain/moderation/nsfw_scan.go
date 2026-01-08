package moderation

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// NSFWScan represents an AI-powered NSFW content scan for an image.
// It is an entity that tracks the scan status, results, and provider details.
type NSFWScan struct {
	id        NSFWScanID
	imageID   gallery.ImageID
	provider  NSFWProvider
	status    NSFWScanStatus
	category  NSFWCategory
	score     float64
	details   NSFWDetails
	errorMsg  string
	scannedAt time.Time
	createdAt time.Time
	updatedAt time.Time

	// Domain events that occurred during this aggregate's lifecycle.
	events []shared.DomainEvent
}

// NewNSFWScan creates a new NSFW scan for an image.
func NewNSFWScan(
	id NSFWScanID,
	imageID gallery.ImageID,
	provider NSFWProvider,
) *NSFWScan {
	now := time.Now().UTC()
	scan := &NSFWScan{
		id:        id,
		imageID:   imageID,
		provider:  provider,
		status:    ScanStatusPending,
		category:  CategoryUnknown,
		score:     0,
		details:   NSFWDetails{},
		createdAt: now,
		updatedAt: now,
		events:    []shared.DomainEvent{},
	}

	scan.addEvent(NewNSFWScanInitiatedEvent(id, imageID, provider))

	return scan
}

// ReconstructNSFWScan reconstructs an NSFWScan from persistence.
// This bypasses validation and should only be used by repositories.
func ReconstructNSFWScan(
	id NSFWScanID,
	imageID gallery.ImageID,
	provider NSFWProvider,
	status NSFWScanStatus,
	category NSFWCategory,
	score float64,
	details NSFWDetails,
	errorMsg string,
	scannedAt time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *NSFWScan {
	return &NSFWScan{
		id:        id,
		imageID:   imageID,
		provider:  provider,
		status:    status,
		category:  category,
		score:     score,
		details:   details,
		errorMsg:  errorMsg,
		scannedAt: scannedAt,
		createdAt: createdAt,
		updatedAt: updatedAt,
		events:    []shared.DomainEvent{},
	}
}

// ID returns the scan's unique identifier.
func (s *NSFWScan) ID() NSFWScanID {
	return s.id
}

// ImageID returns the ID of the scanned image.
func (s *NSFWScan) ImageID() gallery.ImageID {
	return s.imageID
}

// Provider returns the NSFW detection provider used.
func (s *NSFWScan) Provider() NSFWProvider {
	return s.provider
}

// Status returns the current scan status.
func (s *NSFWScan) Status() NSFWScanStatus {
	return s.status
}

// Category returns the detected NSFW category.
func (s *NSFWScan) Category() NSFWCategory {
	return s.category
}

// Score returns the overall NSFW score (0.0 to 1.0).
func (s *NSFWScan) Score() float64 {
	return s.score
}

// Details returns the detailed detection results.
func (s *NSFWScan) Details() NSFWDetails {
	return s.details
}

// ErrorMessage returns the error message if scan failed.
func (s *NSFWScan) ErrorMessage() string {
	return s.errorMsg
}

// ScannedAt returns when the scan completed (or zero if not completed).
func (s *NSFWScan) ScannedAt() time.Time {
	return s.scannedAt
}

// CreatedAt returns when the scan was created.
func (s *NSFWScan) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns when the scan was last updated.
func (s *NSFWScan) UpdatedAt() time.Time {
	return s.updatedAt
}

// Events returns all pending domain events.
func (s *NSFWScan) Events() []shared.DomainEvent {
	return s.events
}

// ClearEvents clears all domain events from this aggregate.
// This should be called after events have been dispatched.
func (s *NSFWScan) ClearEvents() {
	s.events = []shared.DomainEvent{}
}

// MarkScanning transitions the scan to scanning status.
func (s *NSFWScan) MarkScanning() error {
	if s.status.IsTerminal() {
		if s.status == ScanStatusCompleted {
			return ErrNSFWScanAlreadyCompleted
		}
		return ErrNSFWScanAlreadyFailed
	}

	s.status = ScanStatusScanning
	s.updatedAt = time.Now().UTC()
	return nil
}

// Complete marks the scan as successfully completed with results.
func (s *NSFWScan) Complete(category NSFWCategory, score float64, details NSFWDetails) error {
	if s.status.IsTerminal() {
		if s.status == ScanStatusCompleted {
			return ErrNSFWScanAlreadyCompleted
		}
		return ErrNSFWScanAlreadyFailed
	}

	if score < 0 || score > 1 {
		return ErrNSFWScoreOutOfRange
	}

	now := time.Now().UTC()
	s.status = ScanStatusCompleted
	s.category = category
	s.score = clampScore(score)
	s.details = details
	s.scannedAt = now
	s.updatedAt = now

	s.addEvent(NewNSFWScanCompletedEvent(s.id, s.imageID, category, score, category.IsNSFW()))

	return nil
}

// Fail marks the scan as failed with an error message.
func (s *NSFWScan) Fail(errorMsg string) error {
	if s.status.IsTerminal() {
		if s.status == ScanStatusCompleted {
			return ErrNSFWScanAlreadyCompleted
		}
		return ErrNSFWScanAlreadyFailed
	}

	now := time.Now().UTC()
	s.status = ScanStatusFailed
	s.errorMsg = errorMsg
	s.updatedAt = now

	s.addEvent(NewNSFWScanFailedEvent(s.id, s.imageID, errorMsg))

	return nil
}

// IsNSFW returns true if the content was detected as NSFW.
func (s *NSFWScan) IsNSFW() bool {
	return s.status == ScanStatusCompleted && s.category.IsNSFW()
}

// RequiresReview returns true if the content should be manually reviewed.
func (s *NSFWScan) RequiresReview() bool {
	return s.status == ScanStatusCompleted && s.category.RequiresReview()
}

// IsPending returns true if the scan has not started yet.
func (s *NSFWScan) IsPending() bool {
	return s.status == ScanStatusPending
}

// IsCompleted returns true if the scan completed successfully.
func (s *NSFWScan) IsCompleted() bool {
	return s.status == ScanStatusCompleted
}

// IsFailed returns true if the scan failed.
func (s *NSFWScan) IsFailed() bool {
	return s.status == ScanStatusFailed
}

// addEvent adds a domain event to the pending events list.
func (s *NSFWScan) addEvent(event shared.DomainEvent) {
	s.events = append(s.events, event)
}
