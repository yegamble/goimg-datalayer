//nolint:dupl // ID types are intentionally similar for type safety in DDD
package gallery

import (
	"fmt"

	"github.com/google/uuid"
)

// ImageID is a value object representing a unique image identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type ImageID struct {
	value uuid.UUID
}

// NewImageID creates a new ImageID with a generated UUID.
func NewImageID() ImageID {
	return ImageID{value: uuid.New()}
}

// ParseImageID parses a string into an ImageID.
// Returns an error if the string is not a valid UUID.
func ParseImageID(s string) (ImageID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ImageID{}, fmt.Errorf("invalid image id: %w", err)
	}
	return ImageID{value: id}, nil
}

// MustParseImageID parses a string into an ImageID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseImageID(s string) ImageID {
	id, err := ParseImageID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the ImageID.
func (id ImageID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id ImageID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ImageID equals the other ImageID.
func (id ImageID) Equals(other ImageID) bool {
	return id.value == other.value
}
