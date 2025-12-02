package moderation

import (
	"fmt"

	"github.com/google/uuid"
)

// ReviewID is a value object representing a unique review identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type ReviewID struct {
	value uuid.UUID
}

// NewReviewID creates a new ReviewID with a generated UUID.
func NewReviewID() ReviewID {
	return ReviewID{value: uuid.New()}
}

// ParseReviewID parses a string into a ReviewID.
// Returns an error if the string is not a valid UUID.
func ParseReviewID(s string) (ReviewID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ReviewID{}, fmt.Errorf("invalid review id: %w", err)
	}
	return ReviewID{value: id}, nil
}

// MustParseReviewID parses a string into a ReviewID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseReviewID(s string) ReviewID {
	id, err := ParseReviewID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the ReviewID.
func (id ReviewID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id ReviewID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ReviewID equals the other ReviewID.
func (id ReviewID) Equals(other ReviewID) bool {
	return id.value == other.value
}
