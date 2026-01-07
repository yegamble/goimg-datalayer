package identity

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Follow represents a follow relationship between two users.
// It is an entity in the Identity bounded context that captures social connections.
//
// Business Rules:
// - A user cannot follow themselves
// - Each follow relationship is unique (follower + followed)
// - Follow relationships are immutable once created (can only be created or deleted)
type Follow struct {
	followerID UserID
	followedID UserID
	createdAt  time.Time
	events     []shared.DomainEvent
}

// NewFollow creates a new follow relationship between two users.
// Returns an error if a user tries to follow themselves.
// Emits a UserFollowed event.
func NewFollow(followerID, followedID UserID) (*Follow, error) {
	if followerID.IsZero() {
		return nil, fmt.Errorf("follower ID is required")
	}

	if followedID.IsZero() {
		return nil, fmt.Errorf("followed ID is required")
	}

	if followerID.Equals(followedID) {
		return nil, ErrCannotFollowSelf
	}

	now := time.Now().UTC()
	follow := &Follow{
		followerID: followerID,
		followedID: followedID,
		createdAt:  now,
		events:     []shared.DomainEvent{},
	}

	follow.addEvent(NewUserFollowed(followerID, followedID))
	return follow, nil
}

// ReconstructFollow reconstitutes a Follow from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructFollow(
	followerID, followedID UserID,
	createdAt time.Time,
) *Follow {
	return &Follow{
		followerID: followerID,
		followedID: followedID,
		createdAt:  createdAt,
		events:     []shared.DomainEvent{},
	}
}

// FollowerID returns the ID of the user who is following.
func (f *Follow) FollowerID() UserID {
	return f.followerID
}

// FollowedID returns the ID of the user being followed.
func (f *Follow) FollowedID() UserID {
	return f.followedID
}

// CreatedAt returns when the follow relationship was created.
func (f *Follow) CreatedAt() time.Time {
	return f.createdAt
}

// Events returns the domain events that have occurred on this entity.
func (f *Follow) Events() []shared.DomainEvent {
	return f.events
}

// ClearEvents clears all domain events from this entity.
// This should be called after events have been dispatched.
func (f *Follow) ClearEvents() {
	f.events = []shared.DomainEvent{}
}

// addEvent adds a domain event to the entity's event list.
func (f *Follow) addEvent(event shared.DomainEvent) {
	f.events = append(f.events, event)
}

// Matches returns whether this follow relationship matches the given follower and followed IDs.
// This is useful for comparing follow relationships.
func (f *Follow) Matches(followerID, followedID UserID) bool {
	return f.followerID.Equals(followerID) && f.followedID.Equals(followedID)
}
