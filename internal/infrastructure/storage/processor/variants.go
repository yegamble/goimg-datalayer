package processor

import "errors"

// VariantType defines the different size variants of an image.
// These match the domain's gallery.VariantType but are duplicated here
// to avoid infrastructure -> domain imports.
type VariantType string

const (
	// VariantThumbnail is a small preview image (160px max width).
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

// AllVariantTypes returns all valid variant types.
func AllVariantTypes() []VariantType {
	return []VariantType{
		VariantThumbnail,
		VariantSmall,
		VariantMedium,
		VariantLarge,
		VariantOriginal,
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

// String returns the string representation of the variant type.
func (v VariantType) String() string {
	return string(v)
}

// VariantData contains the processed data for a single image variant.
type VariantData struct {
	// Data is the raw image bytes after processing.
	Data []byte

	// Width is the processed image width in pixels.
	Width int

	// Height is the processed image height in pixels.
	Height int

	// Format is the image format (jpeg, png, webp, etc.).
	Format string

	// ContentType is the HTTP content type (e.g., "image/webp").
	ContentType string

	// FileSize is the size of the processed image in bytes.
	FileSize int64
}

// ProcessResult contains all processed variants of an image.
type ProcessResult struct {
	// Original is the re-encoded original image.
	Original VariantData

	// Thumbnail is the thumbnail variant (160px).
	Thumbnail VariantData

	// Small is the small variant (320px).
	Small VariantData

	// Medium is the medium variant (800px).
	Medium VariantData

	// Large is the large variant (1600px).
	Large VariantData

	// OriginalFormat is the detected format of the input image.
	OriginalFormat string

	// OriginalWidth is the width of the input image.
	OriginalWidth int

	// OriginalHeight is the height of the input image.
	OriginalHeight int
}

// GetVariant retrieves a specific variant from the result.
func (r *ProcessResult) GetVariant(variantType VariantType) (*VariantData, error) {
	switch variantType {
	case VariantThumbnail:
		return &r.Thumbnail, nil
	case VariantSmall:
		return &r.Small, nil
	case VariantMedium:
		return &r.Medium, nil
	case VariantLarge:
		return &r.Large, nil
	case VariantOriginal:
		return &r.Original, nil
	default:
		return nil, ErrInvalidVariantType
	}
}

// Errors
var (
	ErrInvalidVariantType = errors.New("invalid variant type")
	ErrInvalidConfig      = errors.New("invalid processor configuration")
	ErrUnsupportedFormat  = errors.New("unsupported image format")
	ErrProcessingFailed   = errors.New("image processing failed")
	ErrInvalidDimensions  = errors.New("invalid image dimensions")
	ErrImageTooSmall      = errors.New("image too small to process")
)

// formatToContentType converts image format to HTTP content type.
func formatToContentType(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
