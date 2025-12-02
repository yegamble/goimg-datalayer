package identity

import (
	"regexp"
	"strings"
)

// Username is a value object representing a validated username.
type Username struct {
	value string
}

var (
	// usernameRegex validates alphanumeric characters and underscores only.
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// reservedUsernames contains usernames that cannot be used.
	// In production, this should be loaded from configuration or database.
	reservedUsernames = map[string]bool{
		"admin":         true,
		"root":          true,
		"system":        true,
		"administrator": true,
		"moderator":     true,
		"mod":           true,
		"support":       true,
		"help":          true,
		"api":           true,
		"www":           true,
		"ftp":           true,
		"mail":          true,
		"postmaster":    true,
		"hostmaster":    true,
		"webmaster":     true,
		"abuse":         true,
		"noreply":       true,
		"no-reply":      true,
		"security":      true,
		"info":          true,
		"marketing":     true,
		"sales":         true,
		"billing":       true,
		"legal":         true,
		"privacy":       true,
		"terms":         true,
		"guest":         true,
		"anonymous":     true,
		"null":          true,
		"undefined":     true,
		"test":          true,
		"demo":          true,
	}
)

// NewUsername creates a new Username value object after validating the input.
// Username is normalized: trimmed, but preserves case.
func NewUsername(value string) (Username, error) {
	// Normalize: trim whitespace
	value = strings.TrimSpace(value)

	// Validate
	if value == "" {
		return Username{}, ErrUsernameEmpty
	}

	if len(value) < 3 {
		return Username{}, ErrUsernameTooShort
	}

	if len(value) > 32 {
		return Username{}, ErrUsernameTooLong
	}

	if !usernameRegex.MatchString(value) {
		return Username{}, ErrUsernameInvalid
	}

	// Check for reserved usernames (case-insensitive)
	if reservedUsernames[strings.ToLower(value)] {
		return Username{}, ErrUsernameReserved
	}

	return Username{value: value}, nil
}

// String returns the string representation of the username.
func (u Username) String() string {
	return u.value
}

// IsEmpty returns true if the username is the zero value.
func (u Username) IsEmpty() bool {
	return u.value == ""
}

// Equals returns true if this Username equals the other Username.
func (u Username) Equals(other Username) bool {
	return u.value == other.value
}
