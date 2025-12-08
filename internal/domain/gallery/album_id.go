//nolint:dupl // ID types are intentionally similar for type safety in DDD
package gallery

import (
	"fmt"

	"github.com/google/uuid"
)

// AlbumID is a value object representing a unique album identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type AlbumID struct {
	value uuid.UUID
}

// NewAlbumID creates a new AlbumID with a generated UUID.
func NewAlbumID() AlbumID {
	return AlbumID{value: uuid.New()}
}

// ParseAlbumID parses a string into an AlbumID.
// Returns an error if the string is not a valid UUID.
func ParseAlbumID(s string) (AlbumID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return AlbumID{}, fmt.Errorf("invalid album id: %w", err)
	}
	return AlbumID{value: id}, nil
}

// MustParseAlbumID parses a string into an AlbumID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseAlbumID(s string) AlbumID {
	id, err := ParseAlbumID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the AlbumID.
func (id AlbumID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id AlbumID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this AlbumID equals the other AlbumID.
func (id AlbumID) Equals(other AlbumID) bool {
	return id.value == other.value
}
