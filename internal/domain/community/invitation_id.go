//nolint:dupl // ID types are intentionally similar for type safety in DDD
package community

import (
	"fmt"

	"github.com/google/uuid"
)

// InvitationID is a value object representing a unique group invitation identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type InvitationID struct {
	value uuid.UUID
}

// NewInvitationID creates a new InvitationID with a generated UUID.
func NewInvitationID() InvitationID {
	return InvitationID{value: uuid.New()}
}

// ParseInvitationID parses a string into an InvitationID.
// Returns an error if the string is not a valid UUID.
func ParseInvitationID(s string) (InvitationID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return InvitationID{}, fmt.Errorf("invalid invitation id: %w", err)
	}
	return InvitationID{value: id}, nil
}

// MustParseInvitationID parses a string into an InvitationID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseInvitationID(s string) InvitationID {
	id, err := ParseInvitationID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the InvitationID.
func (id InvitationID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id InvitationID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this InvitationID equals the other InvitationID.
func (id InvitationID) Equals(other InvitationID) bool {
	return id.value == other.value
}
