package community

import (
	"strings"
)

// GroupName validation constants.
const (
	minGroupNameLength = 3   // Minimum group name length
	maxGroupNameLength = 100 // Maximum group name length
)

// GroupName is a value object representing a validated group name.
// Group names must be between 3-100 characters.
type GroupName struct {
	value string
}

// NewGroupName creates a new GroupName value object after validating the input.
// The name is normalized: trimmed of whitespace and multiple spaces collapsed.
func NewGroupName(value string) (GroupName, error) {
	// Normalize: trim and collapse multiple spaces
	value = strings.TrimSpace(value)
	value = collapseSpaces(value)

	// Validate
	if value == "" {
		return GroupName{}, ErrGroupNameRequired
	}

	if len(value) < minGroupNameLength {
		return GroupName{}, ErrGroupNameTooShort
	}

	if len(value) > maxGroupNameLength {
		return GroupName{}, ErrGroupNameTooLong
	}

	return GroupName{value: value}, nil
}

// String returns the string representation of the group name.
func (n GroupName) String() string {
	return n.value
}

// IsEmpty returns true if the group name is the zero value.
func (n GroupName) IsEmpty() bool {
	return n.value == ""
}

// Equals returns true if this GroupName equals the other GroupName.
func (n GroupName) Equals(other GroupName) bool {
	return n.value == other.value
}

// collapseSpaces collapses multiple consecutive spaces into a single space.
func collapseSpaces(s string) string {
	// Split by any whitespace and rejoin with single spaces
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
