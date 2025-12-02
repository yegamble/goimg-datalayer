package gallery

import "fmt"

// Visibility defines the access level for images and albums.
// It controls who can view the content and whether it appears in search results.
type Visibility string

const (
	// VisibilityPublic makes content visible to everyone and searchable.
	VisibilityPublic Visibility = "public"

	// VisibilityPrivate makes content visible only to the owner.
	VisibilityPrivate Visibility = "private"

	// VisibilityUnlisted makes content accessible via direct link but not searchable.
	VisibilityUnlisted Visibility = "unlisted"
)

// AllVisibilities returns all valid visibility values.
func AllVisibilities() []Visibility {
	return []Visibility{
		VisibilityPublic,
		VisibilityPrivate,
		VisibilityUnlisted,
	}
}

// ParseVisibility parses a string into a Visibility.
// Returns an error if the string is not a valid visibility value.
func ParseVisibility(s string) (Visibility, error) {
	v := Visibility(s)
	switch v {
	case VisibilityPublic, VisibilityPrivate, VisibilityUnlisted:
		return v, nil
	default:
		return "", fmt.Errorf("%w: invalid visibility '%s'", ErrInvalidVisibility, s)
	}
}

// IsValid returns true if this is a valid visibility value.
func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate, VisibilityUnlisted:
		return true
	default:
		return false
	}
}

// IsPublic returns true if content with this visibility is publicly accessible.
func (v Visibility) IsPublic() bool {
	return v == VisibilityPublic
}

// IsPrivate returns true if content with this visibility is private.
func (v Visibility) IsPrivate() bool {
	return v == VisibilityPrivate
}

// IsSearchable returns true if content with this visibility appears in search results.
// Public content is searchable, while private and unlisted content is not.
func (v Visibility) IsSearchable() bool {
	return v == VisibilityPublic
}

// String returns the string representation of the visibility.
func (v Visibility) String() string {
	return string(v)
}
