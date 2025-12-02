package shared

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents an event that occurred in the domain.
// Domain events are immutable facts that have happened and can be used for:
//   - Event sourcing
//   - Cross-context communication
//   - Audit logging
//   - Triggering side effects
//
// Each event has a unique identifier (EventID) to enable deduplication and idempotent processing.
type DomainEvent interface {
	// EventID returns the unique identifier for this event instance.
	EventID() string

	// EventType returns a unique identifier for the event type (e.g., "user.registered").
	EventType() string

	// OccurredAt returns when the event occurred (always UTC).
	OccurredAt() time.Time

	// AggregateID returns the ID of the aggregate that emitted this event.
	AggregateID() string
}

// BaseEvent provides a common implementation of the DomainEvent interface.
// Concrete events should embed BaseEvent and add their specific payload fields.
//
// Example:
//
//	type UserRegistered struct {
//	    shared.BaseEvent
//	    Email string
//	    Username string
//	}
type BaseEvent struct {
	eventID     string
	eventType   string
	occurredAt  time.Time
	aggregateID string
}

// NewBaseEvent creates a new BaseEvent with the given type and aggregate ID.
// EventID is automatically generated as a UUID, and OccurredAt is set to the current UTC time.
func NewBaseEvent(eventType, aggregateID string) BaseEvent {
	return BaseEvent{
		eventID:     uuid.New().String(),
		eventType:   eventType,
		occurredAt:  Now(),
		aggregateID: aggregateID,
	}
}

// EventID returns the unique identifier for this event.
func (e BaseEvent) EventID() string {
	return e.eventID
}

// EventType returns the type of the event.
func (e BaseEvent) EventType() string {
	return e.eventType
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the ID of the aggregate that emitted the event.
func (e BaseEvent) AggregateID() string {
	return e.aggregateID
}
