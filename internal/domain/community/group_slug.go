package community

import (
	"regexp"
	"strings"
)

// GroupSlug validation constants.
const (
	minGroupSlugLength = 3   // Minimum group slug length
	maxGroupSlugLength = 110 // Maximum group slug length
)

var (
	// slugRegex validates that slug contains only lowercase letters, numbers, and hyphens.
	slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
	// invalidSlugCharsRegex matches any characters that are not lowercase letters, numbers, or hyphens.
	invalidSlugCharsRegex = regexp.MustCompile(`[^a-z0-9-]+`)
	// consecutiveHyphensRegex matches multiple consecutive hyphens.
	consecutiveHyphensRegex = regexp.MustCompile(`-+`)
)

// GroupSlug is a value object representing a URL-safe group identifier.
// Slugs must be unique, contain only lowercase letters, numbers, and hyphens,
// and be between 3-110 characters.
type GroupSlug struct {
	value string
}

// NewGroupSlug creates a new GroupSlug value object after validating the input.
// The slug is normalized to lowercase and must only contain a-z, 0-9, and hyphens.
func NewGroupSlug(value string) (GroupSlug, error) {
	// Normalize: trim and lowercase
	value = strings.TrimSpace(strings.ToLower(value))

	// Validate
	if value == "" {
		return GroupSlug{}, ErrGroupSlugRequired
	}

	if len(value) < minGroupSlugLength {
		return GroupSlug{}, ErrGroupSlugTooShort
	}

	if len(value) > maxGroupSlugLength {
		return GroupSlug{}, ErrGroupSlugTooLong
	}

	if !slugRegex.MatchString(value) {
		return GroupSlug{}, ErrGroupSlugInvalid
	}

	return GroupSlug{value: value}, nil
}

// GenerateSlugFromName generates a URL-safe slug from a group name.
// This is a convenience function for generating slugs automatically.
// The application layer should handle uniqueness checks.
func GenerateSlugFromName(name string) (GroupSlug, error) {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove any characters that aren't a-z, 0-9, or hyphen
	slug = invalidSlugCharsRegex.ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	slug = consecutiveHyphensRegex.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Truncate if too long
	if len(slug) > maxGroupSlugLength {
		slug = slug[:maxGroupSlugLength]
		// Trim trailing hyphen again if truncation created one
		slug = strings.TrimRight(slug, "-")
	}

	// Validate the generated slug
	return NewGroupSlug(slug)
}

// String returns the string representation of the group slug.
func (s GroupSlug) String() string {
	return s.value
}

// IsEmpty returns true if the group slug is the zero value.
func (s GroupSlug) IsEmpty() bool {
	return s.value == ""
}

// Equals returns true if this GroupSlug equals the other GroupSlug.
func (s GroupSlug) Equals(other GroupSlug) bool {
	return s.value == other.value
}
