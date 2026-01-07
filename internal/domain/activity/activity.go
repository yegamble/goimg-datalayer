package activity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Activity represents an activity in the user's feed.
// It captures actions performed by users that are relevant to their followers.
//
// Business Rules:
// - ActorID is required (the user who performed the action)
// - ActivityType must be valid
// - TargetID is required (what the action was performed on)
// - TargetType must be valid
// - Metadata is optional but useful for displaying activity details
// - Activities are immutable once created
type Activity struct {
	id           ActivityID
	actorID      identity.UserID
	activityType ActivityType
	targetID     uuid.UUID
	targetType   TargetType
	metadata     map[string]string
	createdAt    time.Time
	events       []shared.DomainEvent
}

// NewActivity creates a new Activity entity.
// Returns an error if any required field is invalid.
func NewActivity(
	actorID identity.UserID,
	activityType ActivityType,
	targetID uuid.UUID,
	targetType TargetType,
	metadata map[string]string,
) (*Activity, error) {
	if actorID.IsZero() {
		return nil, ErrActorRequired
	}

	if !activityType.Valid() {
		return nil, ErrInvalidActivityType
	}

	if targetID == uuid.Nil {
		return nil, ErrTargetRequired
	}

	if !targetType.Valid() {
		return nil, ErrInvalidTargetType
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}

	activity := &Activity{
		id:           NewActivityID(),
		actorID:      actorID,
		activityType: activityType,
		targetID:     targetID,
		targetType:   targetType,
		metadata:     metadata,
		createdAt:    time.Now().UTC(),
		events:       []shared.DomainEvent{},
	}

	activity.addEvent(NewActivityCreated(
		activity.id,
		actorID,
		activityType,
		targetID,
		targetType,
	))

	return activity, nil
}

// ReconstructActivity reconstitutes an Activity from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructActivity(
	id ActivityID,
	actorID identity.UserID,
	activityType ActivityType,
	targetID uuid.UUID,
	targetType TargetType,
	metadata map[string]string,
	createdAt time.Time,
) *Activity {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &Activity{
		id:           id,
		actorID:      actorID,
		activityType: activityType,
		targetID:     targetID,
		targetType:   targetType,
		metadata:     metadata,
		createdAt:    createdAt,
		events:       []shared.DomainEvent{},
	}
}

// ID returns the activity's unique identifier.
func (a *Activity) ID() ActivityID {
	return a.id
}

// ActorID returns the ID of the user who performed the activity.
func (a *Activity) ActorID() identity.UserID {
	return a.actorID
}

// ActivityType returns the type of activity.
func (a *Activity) ActivityType() ActivityType {
	return a.activityType
}

// TargetID returns the ID of the target entity.
func (a *Activity) TargetID() uuid.UUID {
	return a.targetID
}

// TargetType returns the type of the target entity.
func (a *Activity) TargetType() TargetType {
	return a.targetType
}

// Metadata returns the activity's metadata.
func (a *Activity) Metadata() map[string]string {
	// Return a copy to maintain immutability
	meta := make(map[string]string, len(a.metadata))
	for k, v := range a.metadata {
		meta[k] = v
	}
	return meta
}

// CreatedAt returns when the activity was created.
func (a *Activity) CreatedAt() time.Time {
	return a.createdAt
}

// Events returns the domain events that have occurred on this entity.
func (a *Activity) Events() []shared.DomainEvent {
	return a.events
}

// ClearEvents clears all domain events from this entity.
// This should be called after events have been dispatched.
func (a *Activity) ClearEvents() {
	a.events = []shared.DomainEvent{}
}

// addEvent adds a domain event to the entity's event list.
func (a *Activity) addEvent(event shared.DomainEvent) {
	a.events = append(a.events, event)
}
