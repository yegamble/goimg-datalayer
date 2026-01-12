//nolint:dupl // ID types are intentionally similar for type safety in DDD
package community

import (
	"fmt"

	"github.com/google/uuid"
)

// MembershipID is a value object representing a unique group membership identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type MembershipID struct {
	value uuid.UUID
}

// NewMembershipID creates a new MembershipID with a generated UUID.
func NewMembershipID() MembershipID {
	return MembershipID{value: uuid.New()}
}

// ParseMembershipID parses a string into a MembershipID.
// Returns an error if the string is not a valid UUID.
func ParseMembershipID(s string) (MembershipID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return MembershipID{}, fmt.Errorf("invalid membership id: %w", err)
	}
	return MembershipID{value: id}, nil
}

// MustParseMembershipID parses a string into a MembershipID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseMembershipID(s string) MembershipID {
	id, err := ParseMembershipID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the MembershipID.
func (id MembershipID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id MembershipID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this MembershipID equals the other MembershipID.
func (id MembershipID) Equals(other MembershipID) bool {
	return id.value == other.value
}
