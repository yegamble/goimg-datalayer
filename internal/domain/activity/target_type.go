package activity

import "fmt"

// TargetType represents the type of target for an activity.
type TargetType string

const (
	// TargetTypeImage represents an image target.
	TargetTypeImage TargetType = "image"

	// TargetTypeUser represents a user target.
	TargetTypeUser TargetType = "user"

	// TargetTypeAlbum represents an album target.
	TargetTypeAlbum TargetType = "album"

	// TargetTypeComment represents a comment target.
	TargetTypeComment TargetType = "comment"
)

// Valid returns true if the target type is valid.
func (t TargetType) Valid() bool {
	switch t {
	case TargetTypeImage,
		TargetTypeUser,
		TargetTypeAlbum,
		TargetTypeComment:
		return true
	default:
		return false
	}
}

// String returns the string representation of the target type.
func (t TargetType) String() string {
	return string(t)
}

// ParseTargetType creates a TargetType from a string.
func ParseTargetType(s string) (TargetType, error) {
	t := TargetType(s)
	if !t.Valid() {
		return "", fmt.Errorf("invalid target type: %s", s)
	}
	return t, nil
}
