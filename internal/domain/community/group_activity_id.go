//nolint:dupl // ID types are intentionally similar for type safety in DDD
package community

import (
	"fmt"

	"github.com/google/uuid"
)

// GroupActivityID is a value object representing a unique group activity identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type GroupActivityID struct {
	value uuid.UUID
}

// NewGroupActivityID creates a new GroupActivityID with a generated UUID.
func NewGroupActivityID() GroupActivityID {
	return GroupActivityID{value: uuid.New()}
}

// ParseGroupActivityID parses a string into a GroupActivityID.
// Returns an error if the string is not a valid UUID.
func ParseGroupActivityID(s string) (GroupActivityID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return GroupActivityID{}, fmt.Errorf("invalid group activity id: %w", err)
	}
	return GroupActivityID{value: id}, nil
}

// MustParseGroupActivityID parses a string into a GroupActivityID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseGroupActivityID(s string) GroupActivityID {
	id, err := ParseGroupActivityID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the GroupActivityID.
func (id GroupActivityID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id GroupActivityID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this GroupActivityID equals the other GroupActivityID.
func (id GroupActivityID) Equals(other GroupActivityID) bool {
	return id.value == other.value
}
