package community

import "strings"

// MemberStatus represents the current status of a group membership.
// Status tracks the member's lifecycle from invitation through active membership or ban.
type MemberStatus int

const (
	// MemberStatusInvited means the user has been invited but hasn't accepted yet.
	MemberStatusInvited MemberStatus = iota

	// MemberStatusRequested means the user requested to join and is awaiting approval.
	// This status applies to invite-only groups where users can request membership.
	MemberStatusRequested

	// MemberStatusActive means the user is an active member of the group.
	MemberStatusActive

	// MemberStatusBanned means the user has been banned from the group.
	// Banned users cannot rejoin without being unbanned by an admin/owner.
	MemberStatusBanned
)

// String returns the string representation of the MemberStatus.
func (s MemberStatus) String() string {
	switch s {
	case MemberStatusInvited:
		return "invited"
	case MemberStatusRequested:
		return "requested"
	case MemberStatusActive:
		return "active"
	case MemberStatusBanned:
		return "banned"
	default:
		return "unknown"
	}
}

// IsValid returns true if the MemberStatus is valid.
func (s MemberStatus) IsValid() bool {
	return s >= MemberStatusInvited && s <= MemberStatusBanned
}

// IsActive returns true if the member has active access to the group.
func (s MemberStatus) IsActive() bool {
	return s == MemberStatusActive
}

// IsPending returns true if the membership is pending (invited or requested).
func (s MemberStatus) IsPending() bool {
	return s == MemberStatusInvited || s == MemberStatusRequested
}

// IsBanned returns true if the member is banned.
func (s MemberStatus) IsBanned() bool {
	return s == MemberStatusBanned
}

// CanAcceptInvitation returns true if the member can accept an invitation.
func (s MemberStatus) CanAcceptInvitation() bool {
	return s == MemberStatusInvited
}

// ParseMemberStatus parses a string into a MemberStatus.
func ParseMemberStatus(s string) (MemberStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "invited":
		return MemberStatusInvited, nil
	case "requested":
		return MemberStatusRequested, nil
	case "active":
		return MemberStatusActive, nil
	case "banned":
		return MemberStatusBanned, nil
	default:
		return 0, ErrInvalidMemberStatus
	}
}
