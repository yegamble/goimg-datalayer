package moderation

import "fmt"

// NSFWCategory represents the category of NSFW content detected.
type NSFWCategory string

const (
	// CategorySafe indicates the content is safe for all audiences.
	CategorySafe NSFWCategory = "safe"
	// CategorySuggestive indicates mildly suggestive content.
	CategorySuggestive NSFWCategory = "suggestive"
	// CategoryNudity indicates the presence of nudity.
	CategoryNudity NSFWCategory = "nudity"
	// CategoryExplicit indicates explicit sexual content.
	CategoryExplicit NSFWCategory = "explicit"
	// CategoryViolence indicates violent or graphic content.
	CategoryViolence NSFWCategory = "violence"
	// CategoryUnknown indicates the category could not be determined.
	CategoryUnknown NSFWCategory = "unknown"
)

// ParseNSFWCategory creates an NSFWCategory from a string value.
// Returns an error if the string is not a valid category.
func ParseNSFWCategory(s string) (NSFWCategory, error) {
	category := NSFWCategory(s)
	if !category.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidNSFWCategory, s)
	}
	return category, nil
}

// String returns the string representation of the NSFWCategory.
func (c NSFWCategory) String() string {
	return string(c)
}

// IsValid returns true if the NSFWCategory is a valid category value.
func (c NSFWCategory) IsValid() bool {
	switch c {
	case CategorySafe, CategorySuggestive, CategoryNudity, CategoryExplicit, CategoryViolence, CategoryUnknown:
		return true
	default:
		return false
	}
}

// IsNSFW returns true if the category indicates NSFW content.
func (c NSFWCategory) IsNSFW() bool {
	return c == CategoryNudity || c == CategoryExplicit || c == CategoryViolence
}

// IsSuggestive returns true if the category indicates suggestive content.
func (c NSFWCategory) IsSuggestive() bool {
	return c == CategorySuggestive
}

// RequiresReview returns true if content in this category should be reviewed.
func (c NSFWCategory) RequiresReview() bool {
	return c != CategorySafe
}

// Severity constants for NSFW category levels.
const (
	SeveritySafe       = 0
	SeveritySuggestive = 1
	SeverityNudity     = 2
	SeverityViolence   = 3
	SeverityExplicit   = 4
	SeverityUnknown    = -1
)

// Severity returns a numeric severity level (0-4) for sorting/filtering.
// 0 = safe, 1 = suggestive, 2 = nudity, 3 = violence, 4 = explicit, -1 = unknown.
func (c NSFWCategory) Severity() int {
	switch c {
	case CategorySafe:
		return SeveritySafe
	case CategorySuggestive:
		return SeveritySuggestive
	case CategoryNudity:
		return SeverityNudity
	case CategoryViolence:
		return SeverityViolence
	case CategoryExplicit:
		return SeverityExplicit
	case CategoryUnknown:
		return SeverityUnknown
	default:
		return SeverityUnknown
	}
}
