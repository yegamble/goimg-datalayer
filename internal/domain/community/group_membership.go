package community

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// GroupMembership is an entity representing a user's membership in a group.
// It tracks the member's role (Owner, Admin, Member) and status (Active, Invited, Requested, Banned).
// This entity is owned by the Group aggregate but can be queried independently.
type GroupMembership struct {
	id        MembershipID
	groupID   GroupID
	userID    identity.UserID
	role      GroupRole
	status    MemberStatus
	invitedBy *identity.UserID
	joinedAt  time.Time
	updatedAt time.Time
	events    []shared.DomainEvent
}

// NewGroupMembership creates a new GroupMembership with the given group, user, and role.
// The status defaults to Active. For invitations or requests, use NewInvitedMembership or NewRequestedMembership.
func NewGroupMembership(groupID GroupID, userID identity.UserID, role GroupRole) (*GroupMembership, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("%w: group ID is required", shared.ErrInvalidInput)
	}

	if userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID is required", shared.ErrInvalidInput)
	}

	if !role.IsValid() {
		return nil, ErrInvalidGroupRole
	}

	now := time.Now().UTC()
	membership := &GroupMembership{
		id:        NewMembershipID(),
		groupID:   groupID,
		userID:    userID,
		role:      role,
		status:    MemberStatusActive,
		invitedBy: nil,
		joinedAt:  now,
		updatedAt: now,
		events:    []shared.DomainEvent{},
	}

	membership.addEvent(&MemberJoined{
		BaseEvent: shared.NewBaseEvent("community.member.joined", membership.groupID.String()),
		GroupID:   membership.groupID,
		UserID:    membership.userID,
		Role:      membership.role,
	})

	return membership, nil
}

// NewInvitedMembership creates a new GroupMembership in the Invited status.
// The member must accept the invitation to become active.
func NewInvitedMembership(groupID GroupID, userID identity.UserID, invitedBy identity.UserID, role GroupRole) (*GroupMembership, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("%w: group ID is required", shared.ErrInvalidInput)
	}

	if userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID is required", shared.ErrInvalidInput)
	}

	if invitedBy.IsZero() {
		return nil, fmt.Errorf("%w: inviter ID is required", shared.ErrInvalidInput)
	}

	if !role.IsValid() {
		return nil, ErrInvalidGroupRole
	}

	now := time.Now().UTC()
	membership := &GroupMembership{
		id:        NewMembershipID(),
		groupID:   groupID,
		userID:    userID,
		role:      role,
		status:    MemberStatusInvited,
		invitedBy: &invitedBy,
		joinedAt:  now,
		updatedAt: now,
		events:    []shared.DomainEvent{},
	}

	membership.addEvent(&MemberInvited{
		BaseEvent: shared.NewBaseEvent("community.member.invited", membership.groupID.String()),
		GroupID:   membership.groupID,
		UserID:    membership.userID,
		InvitedBy: invitedBy,
		Role:      membership.role,
	})

	return membership, nil
}

// NewRequestedMembership creates a new GroupMembership in the Requested status.
// The group admin/owner must approve the request to make the member active.
func NewRequestedMembership(groupID GroupID, userID identity.UserID) (*GroupMembership, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("%w: group ID is required", shared.ErrInvalidInput)
	}

	if userID.IsZero() {
		return nil, fmt.Errorf("%w: user ID is required", shared.ErrInvalidInput)
	}

	now := time.Now().UTC()
	membership := &GroupMembership{
		id:        NewMembershipID(),
		groupID:   groupID,
		userID:    userID,
		role:      GroupRoleMember, // Always start as member
		status:    MemberStatusRequested,
		invitedBy: nil,
		joinedAt:  now,
		updatedAt: now,
		events:    []shared.DomainEvent{},
	}

	membership.addEvent(&MemberRequested{
		BaseEvent: shared.NewBaseEvent("community.member.requested", membership.groupID.String()),
		GroupID:   membership.groupID,
		UserID:    membership.userID,
	})

	return membership, nil
}

// ReconstructGroupMembership reconstitutes a GroupMembership from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructGroupMembership(
	id MembershipID,
	groupID GroupID,
	userID identity.UserID,
	role GroupRole,
	status MemberStatus,
	invitedBy *identity.UserID,
	joinedAt, updatedAt time.Time,
) *GroupMembership {
	return &GroupMembership{
		id:        id,
		groupID:   groupID,
		userID:    userID,
		role:      role,
		status:    status,
		invitedBy: invitedBy,
		joinedAt:  joinedAt,
		updatedAt: updatedAt,
		events:    []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the membership.
func (m *GroupMembership) ID() MembershipID {
	return m.id
}

// GroupID returns the ID of the group.
func (m *GroupMembership) GroupID() GroupID {
	return m.groupID
}

// UserID returns the ID of the user.
func (m *GroupMembership) UserID() identity.UserID {
	return m.userID
}

// Role returns the member's role.
func (m *GroupMembership) Role() GroupRole {
	return m.role
}

// Status returns the member's status.
func (m *GroupMembership) Status() MemberStatus {
	return m.status
}

// InvitedBy returns the ID of the user who invited this member, or nil if not invited.
func (m *GroupMembership) InvitedBy() *identity.UserID {
	return m.invitedBy
}

// JoinedAt returns when the member joined or was invited.
func (m *GroupMembership) JoinedAt() time.Time {
	return m.joinedAt
}

// UpdatedAt returns when the membership was last modified.
func (m *GroupMembership) UpdatedAt() time.Time {
	return m.updatedAt
}

// Events returns the domain events that have occurred.
func (m *GroupMembership) Events() []shared.DomainEvent {
	return m.events
}

// ClearEvents clears all pending domain events.
func (m *GroupMembership) ClearEvents() {
	m.events = []shared.DomainEvent{}
}

// Behavior Methods

// Activate changes the membership status to Active.
// This is used when accepting an invitation or approving a join request.
func (m *GroupMembership) Activate() error {
	if m.status.IsActive() {
		return ErrMemberAlreadyActive
	}

	if m.status.IsBanned() {
		return ErrMemberBanned
	}

	m.status = MemberStatusActive
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberActivated{
		BaseEvent: shared.NewBaseEvent("community.member.activated", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
	})

	return nil
}

// PromoteToAdmin promotes the member to Admin role.
func (m *GroupMembership) PromoteToAdmin() error {
	if m.role == GroupRoleOwner {
		return nil // Already higher
	}

	if m.role == GroupRoleAdmin {
		return nil // Already admin
	}

	oldRole := m.role
	m.role = GroupRoleAdmin
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberRoleChanged{
		BaseEvent: shared.NewBaseEvent("community.member.role_changed", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
		OldRole:   oldRole,
		NewRole:   m.role,
	})

	return nil
}

// DemoteToMember demotes the member to Member role.
func (m *GroupMembership) DemoteToMember() error {
	if m.role == GroupRoleOwner {
		return ErrCannotDemoteOwner
	}

	if m.role == GroupRoleMember {
		return nil // Already member
	}

	oldRole := m.role
	m.role = GroupRoleMember
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberRoleChanged{
		BaseEvent: shared.NewBaseEvent("community.member.role_changed", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
		OldRole:   oldRole,
		NewRole:   m.role,
	})

	return nil
}

// ChangeRole changes the member's role to the specified role.
// Validates that owners cannot be demoted.
func (m *GroupMembership) ChangeRole(newRole GroupRole) error {
	if !newRole.IsValid() {
		return ErrInvalidGroupRole
	}

	if m.role == GroupRoleOwner {
		return ErrCannotDemoteOwner
	}

	if m.role == newRole {
		return nil // No change
	}

	oldRole := m.role
	m.role = newRole
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberRoleChanged{
		BaseEvent: shared.NewBaseEvent("community.member.role_changed", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
		OldRole:   oldRole,
		NewRole:   m.role,
	})

	return nil
}

// Ban bans the member from the group.
func (m *GroupMembership) Ban(bannedBy identity.UserID, reason string) error {
	if m.role == GroupRoleOwner {
		return ErrCannotBanOwner
	}

	if m.status.IsBanned() {
		return ErrMemberAlreadyBanned
	}

	m.status = MemberStatusBanned
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberBanned{
		BaseEvent: shared.NewBaseEvent("community.member.banned", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
		BannedBy:  bannedBy,
		Reason:    reason,
	})

	return nil
}

// Unban removes the ban from the member, making them active again.
func (m *GroupMembership) Unban() error {
	if !m.status.IsBanned() {
		return nil // Not banned
	}

	m.status = MemberStatusActive
	m.updatedAt = time.Now().UTC()

	m.addEvent(&MemberUnbanned{
		BaseEvent: shared.NewBaseEvent("community.member.unbanned", m.groupID.String()),
		GroupID:   m.groupID,
		UserID:    m.userID,
	})

	return nil
}

// Helper Methods

// IsActive returns true if the member has active access to the group.
func (m *GroupMembership) IsActive() bool {
	return m.status.IsActive()
}

// IsBanned returns true if the member is banned.
func (m *GroupMembership) IsBanned() bool {
	return m.status.IsBanned()
}

// IsOwner returns true if the member is the group owner.
func (m *GroupMembership) IsOwner() bool {
	return m.role.IsOwner()
}

// IsAdmin returns true if the member is an admin.
func (m *GroupMembership) IsAdmin() bool {
	return m.role.IsAdmin()
}

// IsMember returns true if the member is a regular member.
func (m *GroupMembership) IsMember() bool {
	return m.role == GroupRoleMember
}

// CanManageMembers returns true if this member can manage other members.
func (m *GroupMembership) CanManageMembers() bool {
	return m.role.CanManageMembers() && m.status.IsActive()
}

// CanModerateContent returns true if this member can moderate content.
func (m *GroupMembership) CanModerateContent() bool {
	return m.role.CanModerateContent() && m.status.IsActive()
}

// addEvent appends a domain event to the events slice.
func (m *GroupMembership) addEvent(event shared.DomainEvent) {
	m.events = append(m.events, event)
}
