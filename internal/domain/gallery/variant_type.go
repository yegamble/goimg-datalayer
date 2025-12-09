package gallery

import "fmt"

const (
	// Maximum width in pixels for each variant type.
	variantThumbnailMaxWidth = 150
	variantSmallMaxWidth     = 320
	variantMediumMaxWidth    = 800
	variantLargeMaxWidth     = 1600
)

// VariantType defines the different size variants of an image.
// Each variant has a maximum width, and the height is scaled proportionally.
type VariantType string

const (
	// VariantThumbnail is a small preview image (150px max width).
	VariantThumbnail VariantType = "thumbnail"

	// VariantSmall is suitable for mobile devices (320px max width).
	VariantSmall VariantType = "small"

	// VariantMedium is suitable for tablets and web previews (800px max width).
	VariantMedium VariantType = "medium"

	// VariantLarge is suitable for desktop displays (1600px max width).
	VariantLarge VariantType = "large"

	// VariantOriginal is the full-size original image (no resizing).
	VariantOriginal VariantType = "original"
)

// AllVariantTypes returns all valid variant type values.
func AllVariantTypes() []VariantType {
	return []VariantType{
		VariantThumbnail,
		VariantSmall,
		VariantMedium,
		VariantLarge,
		VariantOriginal,
	}
}

// ParseVariantType parses a string into a VariantType.
// Returns an error if the string is not a valid variant type.
func ParseVariantType(s string) (VariantType, error) {
	vt := VariantType(s)
	switch vt {
	case VariantThumbnail, VariantSmall, VariantMedium, VariantLarge, VariantOriginal:
		return vt, nil
	default:
		return "", fmt.Errorf("%w: invalid variant type '%s'", ErrInvalidVariantType, s)
	}
}

// IsValid returns true if this is a valid variant type.
func (v VariantType) IsValid() bool {
	switch v {
	case VariantThumbnail, VariantSmall, VariantMedium, VariantLarge, VariantOriginal:
		return true
	default:
		return false
	}
}

// MaxWidth returns the maximum width in pixels for this variant type.
// Returns 0 for original (no size limit).
func (v VariantType) MaxWidth() int {
	switch v {
	case VariantThumbnail:
		return variantThumbnailMaxWidth
	case VariantSmall:
		return variantSmallMaxWidth
	case VariantMedium:
		return variantMediumMaxWidth
	case VariantLarge:
		return variantLargeMaxWidth
	case VariantOriginal:
		return 0 // No limit
	default:
		return 0
	}
}

// String returns the string representation of the variant type.
func (v VariantType) String() string {
	return string(v)
}
