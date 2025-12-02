package shared

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		eventType   string
		aggregateID string
	}{
		{
			name:        "creates valid base event",
			eventType:   "UserCreated",
			aggregateID: "user-123",
		},
		{
			name:        "handles empty aggregate ID",
			eventType:   "SystemEvent",
			aggregateID: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now().UTC()
			event := NewBaseEvent(tt.eventType, tt.aggregateID)
			after := time.Now().UTC()

			assert.NotEmpty(t, event.EventID())
			assert.Equal(t, tt.eventType, event.EventType())
			assert.Equal(t, tt.aggregateID, event.AggregateID())

			occurredAt := event.OccurredAt()
			assert.True(t, occurredAt.After(before) || occurredAt.Equal(before))
			assert.True(t, occurredAt.Before(after) || occurredAt.Equal(after))
		})
	}
}

func TestBaseEvent_Getters(t *testing.T) {
	t.Parallel()

	event := NewBaseEvent("TestEvent", "aggregate-456")

	t.Run("EventID returns non-empty UUID", func(t *testing.T) {
		t.Parallel()
		id := event.EventID()
		require.NotEmpty(t, id)
		assert.Len(t, id, 36) // UUID string length
	})

	t.Run("EventType returns correct type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "TestEvent", event.EventType())
	})

	t.Run("AggregateID returns correct ID", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "aggregate-456", event.AggregateID())
	})

	t.Run("OccurredAt returns UTC time", func(t *testing.T) {
		t.Parallel()
		occurredAt := event.OccurredAt()
		assert.Equal(t, time.UTC, occurredAt.Location())
	})
}

func TestBaseEvent_ImplementsDomainEvent(t *testing.T) {
	t.Parallel()

	var _ DomainEvent = BaseEvent{}
}

func TestBaseEvent_UniqueEventIDs(t *testing.T) {
	t.Parallel()

	// Create multiple events and verify each has a unique EventID
	const numEvents = 100
	eventIDs := make(map[string]bool)

	for i := 0; i < numEvents; i++ {
		event := NewBaseEvent("test.event", "agg-1")
		id := event.EventID()

		if eventIDs[id] {
			t.Errorf("Duplicate EventID generated: %s", id)
		}
		eventIDs[id] = true
	}

	assert.Len(t, eventIDs, numEvents)
}

func TestBaseEvent_EmbeddingInConcreteEvents(t *testing.T) {
	t.Parallel()

	// Example of embedding BaseEvent in a concrete event
	type UserRegistered struct {
		BaseEvent
		Email    string
		Username string
	}

	event := UserRegistered{
		BaseEvent: NewBaseEvent("user.registered", "user-123"),
		Email:     "test@example.com",
		Username:  "testuser",
	}

	// Verify it implements DomainEvent
	var _ DomainEvent = event

	// Verify base event methods work
	assert.Equal(t, "user.registered", event.EventType())
	assert.Equal(t, "user-123", event.AggregateID())
	assert.NotEmpty(t, event.EventID())

	// Verify custom fields
	assert.Equal(t, "test@example.com", event.Email)
	assert.Equal(t, "testuser", event.Username)
}

func TestBaseEvent_ImmutableFields(t *testing.T) {
	t.Parallel()

	event := NewBaseEvent("test.event", "agg-123")

	// Store original values
	originalID := event.EventID()
	originalType := event.EventType()
	originalAggID := event.AggregateID()
	originalTime := event.OccurredAt()

	// Verify values don't change on subsequent calls
	assert.Equal(t, originalID, event.EventID())
	assert.Equal(t, originalType, event.EventType())
	assert.Equal(t, originalAggID, event.AggregateID())
	assert.Equal(t, originalTime, event.OccurredAt())
}

func TestBaseEvent_MultipleEventsHaveDistinctTimestamps(t *testing.T) {
	t.Parallel()

	// Create multiple events in quick succession
	event1 := NewBaseEvent("test.event", "agg-1")
	event2 := NewBaseEvent("test.event", "agg-2")

	// They should have different EventIDs
	assert.NotEqual(t, event1.EventID(), event2.EventID())

	// Second event should not occur before the first
	assert.False(t, event2.OccurredAt().Before(event1.OccurredAt()))
}

func TestBaseEvent_EmptyStringsAllowed(t *testing.T) {
	t.Parallel()

	// Empty strings should be allowed (validation is up to the concrete event type)
	event := NewBaseEvent("", "")

	assert.Empty(t, event.EventType())
	assert.Empty(t, event.AggregateID())

	// EventID should still be generated
	assert.NotEmpty(t, event.EventID())

	// OccurredAt should still be set
	assert.False(t, event.OccurredAt().IsZero())
}
