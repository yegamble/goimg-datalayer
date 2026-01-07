package activity

import (
	"fmt"

	"github.com/google/uuid"
)

// ActivityID is a value object representing a unique activity identifier.
type ActivityID struct {
	value uuid.UUID
}

// NewActivityID creates a new ActivityID with a generated UUID.
func NewActivityID() ActivityID {
	return ActivityID{value: uuid.New()}
}

// ParseActivityID creates an ActivityID from a string representation.
// Returns an error if the string is not a valid UUID.
func ParseActivityID(s string) (ActivityID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ActivityID{}, fmt.Errorf("invalid activity id: %w", err)
	}
	return ActivityID{value: id}, nil
}

// String returns the string representation of the ActivityID.
func (id ActivityID) String() string {
	return id.value.String()
}

// IsZero returns true if the ActivityID is the zero value.
func (id ActivityID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ActivityID equals the other ActivityID.
func (id ActivityID) Equals(other ActivityID) bool {
	return id.value == other.value
}
