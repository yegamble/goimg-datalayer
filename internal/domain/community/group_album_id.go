//nolint:dupl // ID types are intentionally similar for type safety in DDD
package community

import (
	"fmt"

	"github.com/google/uuid"
)

// GroupAlbumID is a value object representing a unique group album identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type GroupAlbumID struct {
	value uuid.UUID
}

// NewGroupAlbumID creates a new GroupAlbumID with a generated UUID.
func NewGroupAlbumID() GroupAlbumID {
	return GroupAlbumID{value: uuid.New()}
}

// ParseGroupAlbumID parses a string into a GroupAlbumID.
// Returns an error if the string is not a valid UUID.
func ParseGroupAlbumID(s string) (GroupAlbumID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return GroupAlbumID{}, fmt.Errorf("invalid group album id: %w", err)
	}
	return GroupAlbumID{value: id}, nil
}

// MustParseGroupAlbumID parses a string into a GroupAlbumID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseGroupAlbumID(s string) GroupAlbumID {
	id, err := ParseGroupAlbumID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the GroupAlbumID.
func (id GroupAlbumID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id GroupAlbumID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this GroupAlbumID equals the other GroupAlbumID.
func (id GroupAlbumID) Equals(other GroupAlbumID) bool {
	return id.value == other.value
}
