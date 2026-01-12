package gallery

import (
	"fmt"

	"github.com/google/uuid"
)

// FeaturedPickID is a value object representing a unique identifier for a featured pick.
type FeaturedPickID struct {
	value uuid.UUID
}

// NewFeaturedPickID generates a new unique FeaturedPickID.
func NewFeaturedPickID() FeaturedPickID {
	return FeaturedPickID{value: uuid.New()}
}

// ParseFeaturedPickID parses a string UUID into a FeaturedPickID.
func ParseFeaturedPickID(s string) (FeaturedPickID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return FeaturedPickID{}, fmt.Errorf("invalid featured pick id: %w", err)
	}
	return FeaturedPickID{value: id}, nil
}

// String returns the string representation of the FeaturedPickID.
func (id FeaturedPickID) String() string {
	return id.value.String()
}

// IsZero returns true if the ID is the zero value.
func (id FeaturedPickID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ID equals the other ID.
func (id FeaturedPickID) Equals(other FeaturedPickID) bool {
	return id.value == other.value
}
