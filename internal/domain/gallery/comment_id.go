package gallery

import (
	"fmt"

	"github.com/google/uuid"
)

// CommentID is a value object representing a unique comment identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type CommentID struct {
	value uuid.UUID
}

// NewCommentID creates a new CommentID with a generated UUID.
func NewCommentID() CommentID {
	return CommentID{value: uuid.New()}
}

// ParseCommentID parses a string into a CommentID.
// Returns an error if the string is not a valid UUID.
func ParseCommentID(s string) (CommentID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return CommentID{}, fmt.Errorf("invalid comment id: %w", err)
	}
	return CommentID{value: id}, nil
}

// MustParseCommentID parses a string into a CommentID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseCommentID(s string) CommentID {
	id, err := ParseCommentID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the CommentID.
func (id CommentID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id CommentID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this CommentID equals the other CommentID.
func (id CommentID) Equals(other CommentID) bool {
	return id.value == other.value
}
