package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNSFWScanInitiatedEvent(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	provider := moderation.ProviderSightEngine

	event := moderation.NewNSFWScanInitiatedEvent(scanID, imageID, provider)

	// Check DomainEvent interface compliance.
	var _ shared.DomainEvent = event

	assert.Equal(t, "moderation.nsfw_scan.initiated", event.EventType())
	assert.Equal(t, scanID.String(), event.AggregateID())
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
	assert.Equal(t, scanID, event.ScanID)
	assert.Equal(t, imageID, event.ImageID)
	assert.Equal(t, provider, event.Provider)
}

func TestNSFWScanCompletedEvent(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	category := moderation.CategoryNudity
	score := 0.85
	isNSFW := true

	event := moderation.NewNSFWScanCompletedEvent(scanID, imageID, category, score, isNSFW)

	// Check DomainEvent interface compliance.
	var _ shared.DomainEvent = event

	assert.Equal(t, "moderation.nsfw_scan.completed", event.EventType())
	assert.Equal(t, scanID.String(), event.AggregateID())
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
	assert.Equal(t, scanID, event.ScanID)
	assert.Equal(t, imageID, event.ImageID)
	assert.Equal(t, category, event.Category)
	assert.Equal(t, score, event.Score)
	assert.True(t, event.IsNSFW)
}

func TestNSFWScanFailedEvent(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	errorMsg := "API timeout"

	event := moderation.NewNSFWScanFailedEvent(scanID, imageID, errorMsg)

	// Check DomainEvent interface compliance.
	var _ shared.DomainEvent = event

	assert.Equal(t, "moderation.nsfw_scan.failed", event.EventType())
	assert.Equal(t, scanID.String(), event.AggregateID())
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
	assert.Equal(t, scanID, event.ScanID)
	assert.Equal(t, imageID, event.ImageID)
	assert.Equal(t, errorMsg, event.Error)
}

func TestNSFWContentDetectedEvent(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	category := moderation.CategoryExplicit
	score := 0.95

	event := moderation.NewNSFWContentDetectedEvent(scanID, imageID, category, score)

	// Check DomainEvent interface compliance.
	var _ shared.DomainEvent = event

	assert.Equal(t, "moderation.nsfw_content.detected", event.EventType())
	assert.Equal(t, scanID.String(), event.AggregateID())
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
	assert.Equal(t, scanID, event.ScanID)
	assert.Equal(t, imageID, event.ImageID)
	assert.Equal(t, category, event.Category)
	assert.Equal(t, score, event.Score)
}
