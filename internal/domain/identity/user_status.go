package identity

import "fmt"

// UserStatus represents the status of a user account.
type UserStatus string

const (
	// StatusActive indicates the user account is active and can log in.
	StatusActive UserStatus = "active"
	// StatusPending indicates the user account is pending email verification.
	StatusPending UserStatus = "pending"
	// StatusSuspended indicates the user account has been suspended by a moderator.
	StatusSuspended UserStatus = "suspended"
	// StatusDeleted indicates the user account has been soft-deleted.
	StatusDeleted UserStatus = "deleted"
)

// ParseUserStatus creates a UserStatus from a string value.
// Returns an error if the string is not a valid status.
func ParseUserStatus(s string) (UserStatus, error) {
	status := UserStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid user status: %s", s)
	}
	return status, nil
}

// String returns the string representation of the UserStatus.
func (s UserStatus) String() string {
	return string(s)
}

// IsValid returns true if the UserStatus is a valid status value.
func (s UserStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusPending, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// CanLogin returns true if the user with this status can log in.
func (s UserStatus) CanLogin() bool {
	return s == StatusActive
}
