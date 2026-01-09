package moderation

import (
	"fmt"

	"github.com/google/uuid"
)

// NSFWScanID is a value object representing a unique NSFW scan identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type NSFWScanID struct {
	value uuid.UUID
}

// NewNSFWScanID creates a new NSFWScanID with a generated UUID.
func NewNSFWScanID() NSFWScanID {
	return NSFWScanID{value: uuid.New()}
}

// NSFWScanIDFromUUID creates an NSFWScanID from an existing UUID.
func NSFWScanIDFromUUID(id uuid.UUID) NSFWScanID {
	return NSFWScanID{value: id}
}

// ParseNSFWScanID parses a string into an NSFWScanID.
// Returns an error if the string is not a valid UUID.
func ParseNSFWScanID(s string) (NSFWScanID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return NSFWScanID{}, fmt.Errorf("invalid NSFW scan id: %w", err)
	}
	return NSFWScanID{value: id}, nil
}

// MustParseNSFWScanID parses a string into an NSFWScanID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseNSFWScanID(s string) NSFWScanID {
	id, err := ParseNSFWScanID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the NSFWScanID.
func (id NSFWScanID) String() string {
	return id.value.String()
}

// UUID returns the underlying UUID value.
func (id NSFWScanID) UUID() uuid.UUID {
	return id.value
}

// IsZero returns true if this is the zero value (nil UUID).
func (id NSFWScanID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this NSFWScanID equals the other NSFWScanID.
func (id NSFWScanID) Equals(other NSFWScanID) bool {
	return id.value == other.value
}
