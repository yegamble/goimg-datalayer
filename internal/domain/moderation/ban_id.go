//nolint:dupl // ID types are intentionally similar for type safety in DDD
package moderation

import (
	"fmt"

	"github.com/google/uuid"
)

// BanID is a value object representing a unique ban identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type BanID struct {
	value uuid.UUID
}

// NewBanID creates a new BanID with a generated UUID.
func NewBanID() BanID {
	return BanID{value: uuid.New()}
}

// ParseBanID parses a string into a BanID.
// Returns an error if the string is not a valid UUID.
func ParseBanID(s string) (BanID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return BanID{}, fmt.Errorf("invalid ban id: %w", err)
	}
	return BanID{value: id}, nil
}

// MustParseBanID parses a string into a BanID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseBanID(s string) BanID {
	id, err := ParseBanID(s)
	if err != nil {
		panic(err) // Intentional panic for Must* function
	}
	return id
}

// String returns the string representation of the BanID.
func (id BanID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id BanID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this BanID equals the other BanID.
func (id BanID) Equals(other BanID) bool {
	return id.value == other.value
}
