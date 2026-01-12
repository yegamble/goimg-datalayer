//nolint:dupl // ID types are intentionally similar for type safety in DDD
package community

import (
	"fmt"

	"github.com/google/uuid"
)

// GroupID is a value object representing a unique group identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type GroupID struct {
	value uuid.UUID
}

// NewGroupID creates a new GroupID with a generated UUID.
func NewGroupID() GroupID {
	return GroupID{value: uuid.New()}
}

// ParseGroupID parses a string into a GroupID.
// Returns an error if the string is not a valid UUID.
func ParseGroupID(s string) (GroupID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return GroupID{}, fmt.Errorf("invalid group id: %w", err)
	}
	return GroupID{value: id}, nil
}

// MustParseGroupID parses a string into a GroupID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseGroupID(s string) GroupID {
	id, err := ParseGroupID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the GroupID.
func (id GroupID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id GroupID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this GroupID equals the other GroupID.
func (id GroupID) Equals(other GroupID) bool {
	return id.value == other.value
}
