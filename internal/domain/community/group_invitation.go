package community

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// InvitationExpiryDuration is the default expiry duration for invitations (7 days).
	InvitationExpiryDuration = 7 * 24 * time.Hour
)

// GroupInvitation represents an invitation to join a group.
// Invitations can be sent to either:
// - An email address (for users who don't have an account yet)
// - A specific UserID (for existing users)
// Each invitation has a cryptographically secure token and a 7-day expiry.
type GroupInvitation struct {
	id        InvitationID
	groupID   GroupID
	invitedBy identity.UserID
	email     *string          // Email address for non-registered users (mutually exclusive with userID)
	userID    *identity.UserID // User ID for existing users (mutually exclusive with email)
	token     InvitationToken
	expiresAt time.Time
	usedAt    *time.Time
	createdAt time.Time
	events    []shared.DomainEvent
}

// NewGroupInvitation creates a new group invitation with a cryptographically secure token.
// The invitation expires in 7 days by default.
// Either email OR userID must be provided (but not both).
// Returns an error if validation fails.
func NewGroupInvitation(
	groupID GroupID,
	invitedBy identity.UserID,
	email *string,
	userID *identity.UserID,
) (*GroupInvitation, error) {
	// Validate inputs
	if groupID.IsZero() {
		return nil, fmt.Errorf("%w: group ID is required", shared.ErrInvalidInput)
	}
	if invitedBy.IsZero() {
		return nil, fmt.Errorf("%w: inviter ID is required", shared.ErrInvalidInput)
	}

	// Ensure exactly one target (email or userID)
	if email == nil && userID == nil {
		return nil, fmt.Errorf("%w: either email or user ID must be provided", shared.ErrInvalidInput)
	}
	if email != nil && userID != nil {
		return nil, fmt.Errorf("%w: cannot specify both email and user ID", shared.ErrInvalidInput)
	}

	// Validate email if provided
	if email != nil && *email == "" {
		return nil, fmt.Errorf("%w: email cannot be empty", shared.ErrInvalidInput)
	}

	// Validate userID if provided
	if userID != nil && userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID cannot be zero", shared.ErrInvalidInput)
	}

	// Generate cryptographically secure token
	token, err := NewInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("generate invitation token: %w", err)
	}

	now := time.Now().UTC()
	invitation := &GroupInvitation{
		id:        NewInvitationID(),
		groupID:   groupID,
		invitedBy: invitedBy,
		email:     email,
		userID:    userID,
		token:     token,
		expiresAt: now.Add(InvitationExpiryDuration),
		usedAt:    nil,
		createdAt: now,
		events:    []shared.DomainEvent{},
	}

	invitation.addEvent(&GroupInvitationCreated{
		BaseEvent:    shared.NewBaseEvent("community.invitation.created", invitation.id.String()),
		InvitationID: invitation.id,
		GroupID:      invitation.groupID,
		InvitedBy:    invitation.invitedBy,
		Email:        email,
		UserID:       userID,
	})

	return invitation, nil
}

// ReconstructGroupInvitation reconstitutes a GroupInvitation from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructGroupInvitation(
	id InvitationID,
	groupID GroupID,
	invitedBy identity.UserID,
	email *string,
	userID *identity.UserID,
	token InvitationToken,
	expiresAt time.Time,
	usedAt *time.Time,
	createdAt time.Time,
) *GroupInvitation {
	return &GroupInvitation{
		id:        id,
		groupID:   groupID,
		invitedBy: invitedBy,
		email:     email,
		userID:    userID,
		token:     token,
		expiresAt: expiresAt,
		usedAt:    usedAt,
		createdAt: createdAt,
		events:    []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the invitation.
func (i *GroupInvitation) ID() InvitationID {
	return i.id
}

// GroupID returns the ID of the group this invitation is for.
func (i *GroupInvitation) GroupID() GroupID {
	return i.groupID
}

// InvitedBy returns the ID of the user who created the invitation.
func (i *GroupInvitation) InvitedBy() identity.UserID {
	return i.invitedBy
}

// Email returns the email address if this invitation is for a non-registered user.
// Returns nil if the invitation is for an existing user (see UserID()).
func (i *GroupInvitation) Email() *string {
	return i.email
}

// UserID returns the user ID if this invitation is for an existing user.
// Returns nil if the invitation is for a non-registered user (see Email()).
func (i *GroupInvitation) UserID() *identity.UserID {
	return i.userID
}

// Token returns the secure invitation token.
func (i *GroupInvitation) Token() InvitationToken {
	return i.token
}

// ExpiresAt returns when the invitation expires.
func (i *GroupInvitation) ExpiresAt() time.Time {
	return i.expiresAt
}

// UsedAt returns when the invitation was used, or nil if not yet used.
func (i *GroupInvitation) UsedAt() *time.Time {
	return i.usedAt
}

// CreatedAt returns when the invitation was created.
func (i *GroupInvitation) CreatedAt() time.Time {
	return i.createdAt
}

// Events returns the domain events that have occurred.
func (i *GroupInvitation) Events() []shared.DomainEvent {
	return i.events
}

// ClearEvents clears all pending domain events.
func (i *GroupInvitation) ClearEvents() {
	i.events = []shared.DomainEvent{}
}

// Behavior Methods

// IsExpired returns true if the invitation has expired.
func (i *GroupInvitation) IsExpired() bool {
	return time.Now().UTC().After(i.expiresAt)
}

// IsUsed returns true if the invitation has already been used.
func (i *GroupInvitation) IsUsed() bool {
	return i.usedAt != nil
}

// CanBeAccepted returns true if the invitation can be accepted.
// An invitation can be accepted if it:
// - Has not expired
// - Has not already been used
func (i *GroupInvitation) CanBeAccepted() bool {
	return !i.IsExpired() && !i.IsUsed()
}

// Accept marks the invitation as used.
// Returns an error if the invitation cannot be accepted.
func (i *GroupInvitation) Accept() error {
	if i.IsUsed() {
		return ErrInvitationAlreadyUsed
	}
	if i.IsExpired() {
		return ErrInvitationExpired
	}

	now := time.Now().UTC()
	i.usedAt = &now

	i.addEvent(&GroupInvitationAccepted{
		BaseEvent:    shared.NewBaseEvent("community.invitation.accepted", i.id.String()),
		InvitationID: i.id,
		GroupID:      i.groupID,
		AcceptedAt:   now,
	})

	return nil
}

// Helper Methods

// addEvent appends a domain event to the events slice.
func (i *GroupInvitation) addEvent(event shared.DomainEvent) {
	i.events = append(i.events, event)
}
