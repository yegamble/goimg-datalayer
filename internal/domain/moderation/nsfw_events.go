package moderation

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// NSFWScanInitiatedEvent is raised when an NSFW scan is initiated.
type NSFWScanInitiatedEvent struct {
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Provider  NSFWProvider
	Timestamp time.Time
}

// EventType returns the event type identifier.
func (e NSFWScanInitiatedEvent) EventType() string {
	return "moderation.nsfw_scan.initiated"
}

// NSFWScanCompletedEvent is raised when an NSFW scan completes successfully.
type NSFWScanCompletedEvent struct {
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Category  NSFWCategory
	Score     float64
	IsNSFW    bool
	Timestamp time.Time
}

// EventType returns the event type identifier.
func (e NSFWScanCompletedEvent) EventType() string {
	return "moderation.nsfw_scan.completed"
}

// NSFWScanFailedEvent is raised when an NSFW scan fails.
type NSFWScanFailedEvent struct {
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Error     string
	Timestamp time.Time
}

// EventType returns the event type identifier.
func (e NSFWScanFailedEvent) EventType() string {
	return "moderation.nsfw_scan.failed"
}

// NSFWContentDetectedEvent is raised when NSFW content is detected.
// This can trigger automatic moderation actions.
type NSFWContentDetectedEvent struct {
	ScanID    NSFWScanID
	ImageID   gallery.ImageID
	Category  NSFWCategory
	Score     float64
	Timestamp time.Time
}

// EventType returns the event type identifier.
func (e NSFWContentDetectedEvent) EventType() string {
	return "moderation.nsfw_content.detected"
}
