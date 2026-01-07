package notification

import (
	"fmt"

	"github.com/google/uuid"
)

// NotificationID is a value object representing a unique notification identifier.
// It wraps a UUID to provide type safety and domain-specific methods.
type NotificationID struct {
	value uuid.UUID
}

// NewNotificationID generates a new unique notification ID.
func NewNotificationID() NotificationID {
	return NotificationID{value: uuid.New()}
}

// ParseNotificationID parses a string into a NotificationID.
// Returns an error if the string is not a valid UUID.
func ParseNotificationID(s string) (NotificationID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return NotificationID{}, fmt.Errorf("invalid notification id: %w", err)
	}
	return NotificationID{value: id}, nil
}

// String returns the string representation of the notification ID.
func (id NotificationID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id NotificationID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ID equals another notification ID.
func (id NotificationID) Equals(other NotificationID) bool {
	return id.value == other.value
}
