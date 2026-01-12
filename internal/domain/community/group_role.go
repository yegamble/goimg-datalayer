package community

import "strings"

// GroupRole represents a member's role within a group.
// Roles form a hierarchy: Owner > Admin > Member.
type GroupRole int

const (
	// GroupRoleMember is the default role with basic permissions.
	// Members can share images, create albums (if allowed), and view content.
	GroupRoleMember GroupRole = iota

	// GroupRoleAdmin has elevated permissions including member management and moderation.
	// Admins can approve/reject images, invite members, and update group settings.
	GroupRoleAdmin

	// GroupRoleOwner is the highest role with full control.
	// Only the owner can transfer ownership, delete the group, or remove admins.
	GroupRoleOwner
)

// String returns the string representation of the GroupRole.
func (r GroupRole) String() string {
	switch r {
	case GroupRoleMember:
		return "member"
	case GroupRoleAdmin:
		return "admin"
	case GroupRoleOwner:
		return "owner"
	default:
		return "unknown"
	}
}

// IsValid returns true if the GroupRole is valid.
func (r GroupRole) IsValid() bool {
	return r >= GroupRoleMember && r <= GroupRoleOwner
}

// CanManageMembers returns true if this role can manage other members.
func (r GroupRole) CanManageMembers() bool {
	return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanModerateContent returns true if this role can moderate content.
func (r GroupRole) CanModerateContent() bool {
	return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanUpdateSettings returns true if this role can update group settings.
func (r GroupRole) CanUpdateSettings() bool {
	return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanDeleteGroup returns true if this role can delete the group.
func (r GroupRole) CanDeleteGroup() bool {
	return r == GroupRoleOwner
}

// IsHigherThan returns true if this role is higher in the hierarchy than the other role.
func (r GroupRole) IsHigherThan(other GroupRole) bool {
	return r > other
}

// IsOwner returns true if this role is the owner role.
func (r GroupRole) IsOwner() bool {
	return r == GroupRoleOwner
}

// IsAdmin returns true if this role is the admin role.
func (r GroupRole) IsAdmin() bool {
	return r == GroupRoleAdmin
}

// ParseGroupRole parses a string into a GroupRole.
func ParseGroupRole(s string) (GroupRole, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "member":
		return GroupRoleMember, nil
	case "admin":
		return GroupRoleAdmin, nil
	case "owner":
		return GroupRoleOwner, nil
	default:
		return 0, ErrInvalidGroupRole
	}
}
