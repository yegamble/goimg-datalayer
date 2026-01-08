package identity

import "fmt"

// UserType represents the type of user account.
type UserType string

const (
	// UserTypeRegistered is a normal registered user account.
	UserTypeRegistered UserType = "registered"
	// UserTypeGuest is a temporary guest account with limited access.
	UserTypeGuest UserType = "guest"
)

// ParseUserType creates a UserType from a string value.
// Returns an error if the string is not a valid user type.
func ParseUserType(s string) (UserType, error) {
	userType := UserType(s)
	if !userType.IsValid() {
		return "", fmt.Errorf("invalid user type: %s", s)
	}
	return userType, nil
}

// String returns the string representation of the UserType.
func (t UserType) String() string {
	return string(t)
}

// IsValid returns true if the UserType is a valid type value.
func (t UserType) IsValid() bool {
	switch t {
	case UserTypeRegistered, UserTypeGuest:
		return true
	default:
		return false
	}
}

// IsGuest returns true if this is a guest account.
func (t UserType) IsGuest() bool {
	return t == UserTypeGuest
}

// IsRegistered returns true if this is a registered account.
func (t UserType) IsRegistered() bool {
	return t == UserTypeRegistered
}
