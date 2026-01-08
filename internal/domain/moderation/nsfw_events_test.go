package moderation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNSFWScanInitiatedEvent_EventType(t *testing.T) {
	event := moderation.NSFWScanInitiatedEvent{
		ScanID:    moderation.NewNSFWScanID(),
		ImageID:   gallery.NewImageID(),
		Provider:  moderation.ProviderSightEngine,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "moderation.nsfw_scan.initiated", event.EventType())
}

func TestNSFWScanCompletedEvent_EventType(t *testing.T) {
	event := moderation.NSFWScanCompletedEvent{
		ScanID:    moderation.NewNSFWScanID(),
		ImageID:   gallery.NewImageID(),
		Category:  moderation.CategoryNudity,
		Score:     0.85,
		IsNSFW:    true,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "moderation.nsfw_scan.completed", event.EventType())
}

func TestNSFWScanFailedEvent_EventType(t *testing.T) {
	event := moderation.NSFWScanFailedEvent{
		ScanID:    moderation.NewNSFWScanID(),
		ImageID:   gallery.NewImageID(),
		Error:     "API timeout",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "moderation.nsfw_scan.failed", event.EventType())
}

func TestNSFWContentDetectedEvent_EventType(t *testing.T) {
	event := moderation.NSFWContentDetectedEvent{
		ScanID:    moderation.NewNSFWScanID(),
		ImageID:   gallery.NewImageID(),
		Category:  moderation.CategoryExplicit,
		Score:     0.95,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "moderation.nsfw_content.detected", event.EventType())
}
