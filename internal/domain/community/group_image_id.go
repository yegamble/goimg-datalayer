package community

import (
	"fmt"

	"github.com/google/uuid"
)

// GroupImageID is a value object representing a unique identifier for a group image.
type GroupImageID struct {
	value uuid.UUID
}

// NewGroupImageID creates a new GroupImageID with a generated UUID.
func NewGroupImageID() GroupImageID {
	return GroupImageID{value: uuid.New()}
}

// ParseGroupImageID creates a GroupImageID from a string.
// Returns an error if the string is not a valid UUID.
func ParseGroupImageID(s string) (GroupImageID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return GroupImageID{}, fmt.Errorf("invalid group image id: %w", err)
	}
	return GroupImageID{value: parsed}, nil
}

// MustParseGroupImageID creates a GroupImageID from a string, panicking on invalid input.
// Use this only in tests or for known-valid IDs.
func MustParseGroupImageID(s string) GroupImageID {
	id, err := ParseGroupImageID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the GroupImageID.
func (id GroupImageID) String() string {
	return id.value.String()
}

// UUID returns the underlying UUID value.
func (id GroupImageID) UUID() uuid.UUID {
	return id.value
}

// IsZero returns true if the ID is the zero value.
func (id GroupImageID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if two GroupImageIDs are equal.
func (id GroupImageID) Equals(other GroupImageID) bool {
	return id.value == other.value
}
