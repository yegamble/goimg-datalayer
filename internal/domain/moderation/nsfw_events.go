package moderation

import (
	"time"

	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Ensure events implement DomainEvent interface.
var (
	_ shared.DomainEvent = (*NSFWScanInitiatedEvent)(nil)
	_ shared.DomainEvent = (*NSFWScanCompletedEvent)(nil)
	_ shared.DomainEvent = (*NSFWScanFailedEvent)(nil)
	_ shared.DomainEvent = (*NSFWContentDetectedEvent)(nil)
)

// NSFWScanInitiatedEvent is raised when an NSFW scan is initiated.
type NSFWScanInitiatedEvent struct {
	eventID   string
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Provider  NSFWProvider
	Timestamp time.Time
}

// NewNSFWScanInitiatedEvent creates a new NSFWScanInitiatedEvent.
func NewNSFWScanInitiatedEvent(
	scanID NSFWScanID,
	imageID gallery.ImageID,
	provider NSFWProvider,
) NSFWScanInitiatedEvent {
	return NSFWScanInitiatedEvent{
		eventID:   uuid.New().String(),
		ScanID:    scanID,
		ImageID:   imageID,
		Provider:  provider,
		Timestamp: time.Now().UTC(),
	}
}

// EventID returns the unique identifier for this event instance.
func (e NSFWScanInitiatedEvent) EventID() string {
	return e.eventID
}

// EventType returns the event type identifier.
func (e NSFWScanInitiatedEvent) EventType() string {
	return "moderation.nsfw_scan.initiated"
}

// OccurredAt returns when the event occurred.
func (e NSFWScanInitiatedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that emitted this event.
func (e NSFWScanInitiatedEvent) AggregateID() string {
	return e.ScanID.String()
}

// NSFWScanCompletedEvent is raised when an NSFW scan completes successfully.
type NSFWScanCompletedEvent struct {
	eventID   string
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Category  NSFWCategory
	Score     float64
	IsNSFW    bool
	Timestamp time.Time
}

// NewNSFWScanCompletedEvent creates a new NSFWScanCompletedEvent.
func NewNSFWScanCompletedEvent(
	scanID NSFWScanID,
	imageID gallery.ImageID,
	category NSFWCategory,
	score float64,
	isNSFW bool,
) NSFWScanCompletedEvent {
	return NSFWScanCompletedEvent{
		eventID:   uuid.New().String(),
		ScanID:    scanID,
		ImageID:   imageID,
		Category:  category,
		Score:     score,
		IsNSFW:    isNSFW,
		Timestamp: time.Now().UTC(),
	}
}

// EventID returns the unique identifier for this event instance.
func (e NSFWScanCompletedEvent) EventID() string {
	return e.eventID
}

// EventType returns the event type identifier.
func (e NSFWScanCompletedEvent) EventType() string {
	return "moderation.nsfw_scan.completed"
}

// OccurredAt returns when the event occurred.
func (e NSFWScanCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that emitted this event.
func (e NSFWScanCompletedEvent) AggregateID() string {
	return e.ScanID.String()
}

// NSFWScanFailedEvent is raised when an NSFW scan fails.
type NSFWScanFailedEvent struct {
	eventID   string
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Error     string
	Timestamp time.Time
}

// NewNSFWScanFailedEvent creates a new NSFWScanFailedEvent.
func NewNSFWScanFailedEvent(scanID NSFWScanID, imageID gallery.ImageID, errorMsg string) NSFWScanFailedEvent {
	return NSFWScanFailedEvent{
		eventID:   uuid.New().String(),
		ScanID:    scanID,
		ImageID:   imageID,
		Error:     errorMsg,
		Timestamp: time.Now().UTC(),
	}
}

// EventID returns the unique identifier for this event instance.
func (e NSFWScanFailedEvent) EventID() string {
	return e.eventID
}

// EventType returns the event type identifier.
func (e NSFWScanFailedEvent) EventType() string {
	return "moderation.nsfw_scan.failed"
}

// OccurredAt returns when the event occurred.
func (e NSFWScanFailedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that emitted this event.
func (e NSFWScanFailedEvent) AggregateID() string {
	return e.ScanID.String()
}

// NSFWContentDetectedEvent is raised when NSFW content is detected.
// This can trigger automatic moderation actions.
type NSFWContentDetectedEvent struct {
	eventID   string
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Category  NSFWCategory
	Score     float64
	Timestamp time.Time
}

// NewNSFWContentDetectedEvent creates a new NSFWContentDetectedEvent.
func NewNSFWContentDetectedEvent(
	scanID NSFWScanID,
	imageID gallery.ImageID,
	category NSFWCategory,
	score float64,
) NSFWContentDetectedEvent {
	return NSFWContentDetectedEvent{
		eventID:   uuid.New().String(),
		ScanID:    scanID,
		ImageID:   imageID,
		Category:  category,
		Score:     score,
		Timestamp: time.Now().UTC(),
	}
}

// EventID returns the unique identifier for this event instance.
func (e NSFWContentDetectedEvent) EventID() string {
	return e.eventID
}

// EventType returns the event type identifier.
func (e NSFWContentDetectedEvent) EventType() string {
	return "moderation.nsfw_content.detected"
}

// OccurredAt returns when the event occurred.
func (e NSFWContentDetectedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that emitted this event.
func (e NSFWContentDetectedEvent) AggregateID() string {
	return e.ScanID.String()
}
