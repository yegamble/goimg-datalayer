package gallery

import "fmt"

// ImageVariant represents a processed version of an image at a specific size.
// Each image can have multiple variants (thumbnail, small, medium, large, original).
// Variants are immutable once created.
type ImageVariant struct {
	variantType VariantType
	storageKey  string
	width       int
	height      int
	fileSize    int64
	format      string // "jpeg", "png", "webp", etc.
}

// NewImageVariant creates a new ImageVariant with validation.
// All parameters must be valid:
// - variantType: must be a valid VariantType
// - storageKey: required
// - width, height: must be positive
// - fileSize: must be positive.
// - format: required (normalized to lowercase).
func NewImageVariant(
	variantType VariantType,
	storageKey string,
	width,
	height int,
	fileSize int64,
	format string,
) (ImageVariant, error) {
	// Validate variant type
	if !variantType.IsValid() {
		return ImageVariant{}, fmt.Errorf("%w: %s", ErrInvalidVariantType, variantType)
	}

	// Validate storage key
	if storageKey == "" {
		return ImageVariant{}, fmt.Errorf("%w: storage key is required", ErrInvalidVariantData)
	}

	// Validate dimensions
	if width <= 0 || height <= 0 {
		return ImageVariant{}, fmt.Errorf("%w: dimensions must be positive, got %dx%d", ErrInvalidVariantData, width, height)
	}

	// Validate file size
	if fileSize <= 0 {
		return ImageVariant{}, fmt.Errorf("%w: file size must be positive", ErrInvalidVariantData)
	}

	// Validate and normalize format
	if format == "" {
		return ImageVariant{}, fmt.Errorf("%w: format is required", ErrInvalidVariantData)
	}
	format = normalizeFormat(format)

	return ImageVariant{
		variantType: variantType,
		storageKey:  storageKey,
		width:       width,
		height:      height,
		fileSize:    fileSize,
		format:      format,
	}, nil
}

// VariantType returns the variant type (thumbnail, small, medium, large, original).
func (v ImageVariant) VariantType() VariantType {
	return v.variantType
}

// StorageKey returns the storage key for retrieving this variant.
func (v ImageVariant) StorageKey() string {
	return v.storageKey
}

// Width returns the width in pixels.
func (v ImageVariant) Width() int {
	return v.width
}

// Height returns the height in pixels.
func (v ImageVariant) Height() int {
	return v.height
}

// FileSize returns the file size in bytes.
func (v ImageVariant) FileSize() int64 {
	return v.fileSize
}

// Format returns the image format (e.g., "jpeg", "png", "webp").
func (v ImageVariant) Format() string {
	return v.format
}

// AspectRatio returns the width/height ratio.
func (v ImageVariant) AspectRatio() float64 {
	if v.height == 0 {
		return 0
	}
	return float64(v.width) / float64(v.height)
}

// normalizeFormat normalizes image format strings to lowercase.
func normalizeFormat(format string) string {
	// Handle common variations
	switch format {
	case "jpg", "JPG", "JPEG":
		return "jpeg"
	case "png", "PNG":
		return "png"
	case "gif", "GIF":
		return "gif"
	case "webp", "WEBP", "WebP":
		return "webp"
	default:
		// For other formats, just lowercase
		return format
	}
}
