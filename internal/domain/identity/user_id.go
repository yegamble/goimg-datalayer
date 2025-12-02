package identity

import (
	"fmt"

	"github.com/google/uuid"
)

// UserID is a value object representing a unique user identifier.
type UserID struct {
	value uuid.UUID
}

// NewUserID creates a new UserID with a generated UUID.
func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

// ParseUserID creates a UserID from a string representation.
// Returns an error if the string is not a valid UUID.
func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, fmt.Errorf("invalid user id: %w", err)
	}
	return UserID{value: id}, nil
}

// String returns the string representation of the UserID.
func (id UserID) String() string {
	return id.value.String()
}

// IsZero returns true if the UserID is the zero value.
func (id UserID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this UserID equals the other UserID.
func (id UserID) Equals(other UserID) bool {
	return id.value == other.value
}
