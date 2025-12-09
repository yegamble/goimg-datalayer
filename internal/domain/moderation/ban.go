package moderation

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// Maximum length for ban reason.
	maxBanReasonLength = 500
)

// Ban is the aggregate root for user bans.
// It represents a temporary or permanent restriction placed on a user.
//
// Business Rules:
//   - Reason is required and must not exceed 500 characters
//   - ExpiresAt of nil indicates a permanent ban
//   - A ban is active if: not revoked AND (permanent OR not expired)
//   - Revoked bans cannot be un-revoked
//   - Expired bans are automatically inactive
type Ban struct {
	id        BanID
	userID    identity.UserID
	bannedBy  identity.UserID
	reason    string
	expiresAt *time.Time
	createdAt time.Time
	revokedAt *time.Time
	revokedBy *identity.UserID
	events    []shared.DomainEvent
}

// NewBan creates a new Ban for the given user with a reason.
// If duration is nil, the ban is permanent.
// If duration is provided, expiresAt is set to createdAt + duration.
// Returns an error if validation fails.
func NewBan(
	userID identity.UserID,
	bannedBy identity.UserID,
	reason string,
	duration *time.Duration,
) (*Ban, error) {
	// Validate inputs
	if userID.IsZero() {
		return nil, fmt.Errorf("user id is required")
	}
	if bannedBy.IsZero() {
		return nil, fmt.Errorf("banned by user id is required")
	}
	if reason == "" {
		return nil, ErrReasonRequired
	}
	if len(reason) > maxBanReasonLength {
		return nil, ErrReasonTooLong
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if duration != nil {
		expiry := now.Add(*duration)
		expiresAt = &expiry
	}

	ban := &Ban{
		id:        NewBanID(),
		userID:    userID,
		bannedBy:  bannedBy,
		reason:    reason,
		expiresAt: expiresAt,
		createdAt: now,
		revokedAt: nil,
		revokedBy: nil,
		events:    []shared.DomainEvent{},
	}

	ban.addEvent(NewUserBanned(ban.id, userID, bannedBy, reason, duration == nil))
	return ban, nil
}

// ReconstructBan reconstitutes a Ban from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructBan(
	id BanID,
	userID identity.UserID,
	bannedBy identity.UserID,
	reason string,
	expiresAt *time.Time,
	createdAt time.Time,
	revokedAt *time.Time,
	revokedBy *identity.UserID,
) *Ban {
	return &Ban{
		id:        id,
		userID:    userID,
		bannedBy:  bannedBy,
		reason:    reason,
		expiresAt: expiresAt,
		createdAt: createdAt,
		revokedAt: revokedAt,
		revokedBy: revokedBy,
		events:    []shared.DomainEvent{},
	}
}

// ID returns the ban's unique identifier.
func (b *Ban) ID() BanID {
	return b.id
}

// UserID returns the ID of the banned user.
func (b *Ban) UserID() identity.UserID {
	return b.userID
}

// BannedBy returns the ID of the user who issued the ban.
func (b *Ban) BannedBy() identity.UserID {
	return b.bannedBy
}

// Reason returns the reason for the ban.
func (b *Ban) Reason() string {
	return b.reason
}

// ExpiresAt returns the expiration time of the ban.
// Returns nil for permanent bans.
func (b *Ban) ExpiresAt() *time.Time {
	return b.expiresAt
}

// CreatedAt returns when the ban was created.
func (b *Ban) CreatedAt() time.Time {
	return b.createdAt
}

// RevokedAt returns when the ban was revoked, if revoked.
func (b *Ban) RevokedAt() *time.Time {
	return b.revokedAt
}

// RevokedBy returns the ID of the user who revoked the ban, if revoked.
func (b *Ban) RevokedBy() *identity.UserID {
	return b.revokedBy
}

// Events returns the domain events that have occurred on this aggregate.
func (b *Ban) Events() []shared.DomainEvent {
	return b.events
}

// ClearEvents clears all domain events from this aggregate.
// This should be called after events have been dispatched.
func (b *Ban) ClearEvents() {
	b.events = []shared.DomainEvent{}
}

// IsActive returns true if the ban is currently active.
// A ban is active if it has not been revoked and is not expired.
func (b *Ban) IsActive() bool {
	// If revoked, not active
	if b.revokedAt != nil {
		return false
	}

	// If permanent (no expiry), active
	if b.expiresAt == nil {
		return true
	}

	// If temporary, check if expired
	return time.Now().UTC().Before(*b.expiresAt)
}

// IsPermanent returns true if the ban has no expiration date.
func (b *Ban) IsPermanent() bool {
	return b.expiresAt == nil
}

// IsExpired returns true if the ban has a natural expiration date that has passed.
// Returns false for permanent bans or revoked bans.
func (b *Ban) IsExpired() bool {
	// Permanent bans never expire
	if b.expiresAt == nil {
		return false
	}

	// Revoked bans are not considered "expired" in the natural sense
	if b.revokedAt != nil {
		return false
	}

	// Check if current time is past expiration
	return time.Now().UTC().After(*b.expiresAt) || time.Now().UTC().Equal(*b.expiresAt)
}

// Revoke manually revokes the ban before its natural expiration.
// Returns an error if the ban is already revoked or has expired.
func (b *Ban) Revoke(revokedBy identity.UserID) error {
	if revokedBy.IsZero() {
		return fmt.Errorf("revoked by user id is required")
	}

	if b.revokedAt != nil {
		return ErrBanAlreadyRevoked
	}

	// Check if ban has naturally expired
	if b.IsExpired() {
		return ErrBanExpired
	}

	now := time.Now().UTC()
	b.revokedAt = &now
	b.revokedBy = &revokedBy

	b.addEvent(NewBanRevoked(b.id, b.userID, revokedBy))
	return nil
}

// addEvent adds a domain event to the aggregate's event list.
func (b *Ban) addEvent(event shared.DomainEvent) {
	b.events = append(b.events, event)
}
