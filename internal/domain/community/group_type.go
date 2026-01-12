package community

import "strings"

// GroupType represents the access level and discoverability of a group.
type GroupType int

const (
	// GroupTypePublic allows anyone to join without approval.
	// Public groups are discoverable via search and browsing.
	GroupTypePublic GroupType = iota

	// GroupTypePrivate is invite-only and not discoverable.
	// Only members can see the group and only the owner/admin can invite.
	GroupTypePrivate

	// GroupTypeInviteOnly is discoverable but requires an invitation or approval to join.
	// Anyone can find the group and request to join, but owner/admin must approve.
	GroupTypeInviteOnly
)

// String returns the string representation of the GroupType.
func (t GroupType) String() string {
	switch t {
	case GroupTypePublic:
		return "public"
	case GroupTypePrivate:
		return "private"
	case GroupTypeInviteOnly:
		return "invite_only"
	default:
		return "unknown"
	}
}

// IsValid returns true if the GroupType is valid.
func (t GroupType) IsValid() bool {
	return t >= GroupTypePublic && t <= GroupTypeInviteOnly
}

// IsDiscoverable returns true if the group type allows public discovery.
// Public and InviteOnly groups are discoverable; Private groups are not.
func (t GroupType) IsDiscoverable() bool {
	return t == GroupTypePublic || t == GroupTypeInviteOnly
}

// AllowsInstantJoin returns true if users can join without approval.
func (t GroupType) AllowsInstantJoin() bool {
	return t == GroupTypePublic
}

// ParseGroupType parses a string into a GroupType.
func ParseGroupType(s string) (GroupType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "public":
		return GroupTypePublic, nil
	case "private":
		return GroupTypePrivate, nil
	case "invite_only", "inviteonly", "invite-only":
		return GroupTypeInviteOnly, nil
	default:
		return 0, ErrInvalidGroupType
	}
}
