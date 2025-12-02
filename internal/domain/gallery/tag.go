package gallery

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// MinTagLength is the minimum allowed length for a tag name.
	MinTagLength = 2

	// MaxTagLength is the maximum allowed length for a tag name.
	MaxTagLength = 50

	// MaxTagsPerImage is the maximum number of tags allowed per image.
	MaxTagsPerImage = 20
)

// tagPattern defines the allowed characters for tags: alphanumeric, hyphen, underscore.
var tagPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// Tag is a value object representing a keyword or label for categorizing images.
// Tags are normalized to lowercase and validated for allowed characters.
type Tag struct {
	name string
	slug string
}

// NewTag creates a new Tag with validation and normalization.
// The name is normalized to lowercase, and a URL-friendly slug is generated.
// Valid characters: alphanumeric, hyphen, underscore.
func NewTag(name string) (Tag, error) {
	// Normalize: trim whitespace and convert to lowercase
	name = strings.TrimSpace(strings.ToLower(name))

	// Validate length
	if len(name) < MinTagLength {
		return Tag{}, fmt.Errorf("%w: must be at least %d characters", ErrTagTooShort, MinTagLength)
	}
	if len(name) > MaxTagLength {
		return Tag{}, fmt.Errorf("%w: maximum %d characters, got %d", ErrTagTooLong, MaxTagLength, len(name))
	}

	// Replace spaces with hyphens for slug
	slug := strings.ReplaceAll(name, " ", "-")

	// Remove any characters that aren't alphanumeric, hyphen, or underscore
	slug = cleanSlug(slug)

	// Validate pattern (after cleaning)
	if !tagPattern.MatchString(slug) {
		return Tag{}, fmt.Errorf("%w: only alphanumeric, hyphen, and underscore allowed", ErrTagInvalid)
	}

	// Ensure slug is not empty after cleaning
	if slug == "" {
		return Tag{}, fmt.Errorf("%w: tag contains no valid characters", ErrTagInvalid)
	}

	return Tag{
		name: name,
		slug: slug,
	}, nil
}

// MustNewTag creates a new Tag and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustNewTag(name string) Tag {
	tag, err := NewTag(name)
	if err != nil {
		panic(err)
	}
	return tag
}

// Name returns the display name of the tag.
func (t Tag) Name() string {
	return t.name
}

// Slug returns the URL-friendly slug of the tag.
func (t Tag) Slug() string {
	return t.slug
}

// Equals returns true if this tag equals the other tag.
// Tags are considered equal if their slugs match.
func (t Tag) Equals(other Tag) bool {
	return t.slug == other.slug
}

// String returns the string representation of the tag (the name).
func (t Tag) String() string {
	return t.name
}

// cleanSlug removes any character that isn't alphanumeric, hyphen, or underscore.
// It also collapses multiple hyphens/underscores into a single one.
func cleanSlug(s string) string {
	var result strings.Builder
	lastWasSpecial := false

	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			lastWasSpecial = false
		} else if (r == '-' || r == '_') && !lastWasSpecial {
			result.WriteRune(r)
			lastWasSpecial = true
		}
	}

	// Trim leading/trailing special characters
	slug := strings.Trim(result.String(), "-_")
	return slug
}
