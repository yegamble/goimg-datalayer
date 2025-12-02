package identity

import "fmt"

// Role represents a user's role in the system.
type Role string

const (
	// RoleUser is the default role for regular users.
	RoleUser Role = "user"
	// RoleModerator is the role for content moderators.
	RoleModerator Role = "moderator"
	// RoleAdmin is the role for system administrators.
	RoleAdmin Role = "admin"
)

// ParseRole creates a Role from a string value.
// Returns an error if the string is not a valid role.
func ParseRole(s string) (Role, error) {
	role := Role(s)
	if !role.IsValid() {
		return "", fmt.Errorf("invalid role: %s", s)
	}
	return role, nil
}

// String returns the string representation of the Role.
func (r Role) String() string {
	return string(r)
}

// IsValid returns true if the Role is a valid role value.
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleModerator, RoleAdmin:
		return true
	default:
		return false
	}
}

// CanModerate returns true if the Role has moderation permissions.
func (r Role) CanModerate() bool {
	return r == RoleModerator || r == RoleAdmin
}

// IsAdmin returns true if the Role is an administrator.
func (r Role) IsAdmin() bool {
	return r == RoleAdmin
}
