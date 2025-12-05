// Package processor provides image processing capabilities using bimg (libvips).
// It handles variant generation, EXIF stripping, and format conversion.
package processor

import "github.com/h2non/bimg"

// Config defines the image processor configuration.
type Config struct {
	// MemoryLimitMB is the maximum memory for bimg cache in megabytes.
	// Default: 256MB
	MemoryLimitMB int

	// MaxConcurrentOps is the maximum number of concurrent processing operations.
	// Uses a worker pool pattern to limit resource usage.
	// Default: 32
	MaxConcurrentOps int

	// StripMetadata controls whether to strip EXIF metadata from images.
	// Should always be true for privacy and security.
	// Default: true
	StripMetadata bool

	// Quality settings for each variant type
	ThumbnailQuality int // Default: 82
	SmallQuality     int // Default: 85
	MediumQuality    int // Default: 85
	LargeQuality     int // Default: 88
	OriginalQuality  int // Default: 100 (maximum quality, near-lossless)
}

// DefaultConfig returns the recommended processor configuration.
func DefaultConfig() Config {
	return Config{
		MemoryLimitMB:    256,
		MaxConcurrentOps: 32,
		StripMetadata:    true,
		ThumbnailQuality: 82,
		SmallQuality:     85,
		MediumQuality:    85,
		LargeQuality:     88,
		OriginalQuality:  100,
	}
}

// Validate ensures the configuration is valid.
func (c Config) Validate() error {
	if c.MemoryLimitMB <= 0 {
		return ErrInvalidConfig
	}
	if c.MaxConcurrentOps <= 0 {
		return ErrInvalidConfig
	}
	if c.ThumbnailQuality < 1 || c.ThumbnailQuality > 100 {
		return ErrInvalidConfig
	}
	if c.SmallQuality < 1 || c.SmallQuality > 100 {
		return ErrInvalidConfig
	}
	if c.MediumQuality < 1 || c.MediumQuality > 100 {
		return ErrInvalidConfig
	}
	if c.LargeQuality < 1 || c.LargeQuality > 100 {
		return ErrInvalidConfig
	}
	if c.OriginalQuality < 1 || c.OriginalQuality > 100 {
		return ErrInvalidConfig
	}
	return nil
}

// VariantSpec defines the specifications for an image variant.
type VariantSpec struct {
	// MaxWidth is the maximum width in pixels (0 = no limit).
	MaxWidth int

	// Format is the output format (jpeg, png, webp, etc.).
	Format bimg.ImageType

	// Quality is the compression quality (1-100).
	Quality int
}

// GetVariantSpec returns the processing specification for a variant type.
func (c Config) GetVariantSpec(variantType VariantType) VariantSpec {
	switch variantType {
	case VariantThumbnail:
		return VariantSpec{
			MaxWidth: 160,
			Format:   bimg.WEBP,
			Quality:  c.ThumbnailQuality,
		}
	case VariantSmall:
		return VariantSpec{
			MaxWidth: 320,
			Format:   bimg.WEBP,
			Quality:  c.SmallQuality,
		}
	case VariantMedium:
		return VariantSpec{
			MaxWidth: 800,
			Format:   bimg.WEBP,
			Quality:  c.MediumQuality,
		}
	case VariantLarge:
		return VariantSpec{
			MaxWidth: 1600,
			Format:   bimg.WEBP,
			Quality:  c.LargeQuality,
		}
	case VariantOriginal:
		// Original keeps the source format
		return VariantSpec{
			MaxWidth: 0, // No resizing
			Format:   0, // Keep original format (detected at runtime)
			Quality:  c.OriginalQuality,
		}
	default:
		// Fallback to medium
		return VariantSpec{
			MaxWidth: 800,
			Format:   bimg.WEBP,
			Quality:  c.MediumQuality,
		}
	}
}

// SupportedFormats returns the list of supported input image formats.
func SupportedFormats() []string {
	return []string{"jpeg", "png", "gif", "webp"}
}

// IsSupportedFormat checks if a format is supported for processing.
func IsSupportedFormat(format string) bool {
	for _, f := range SupportedFormats() {
		if f == format {
			return true
		}
	}
	return false
}

// bimgTypeToString converts bimg.ImageType to string format.
func bimgTypeToString(t bimg.ImageType) string {
	switch t {
	case bimg.JPEG:
		return "jpeg"
	case bimg.PNG:
		return "png"
	case bimg.WEBP:
		return "webp"
	case bimg.GIF:
		return "gif"
	default:
		return "unknown"
	}
}

// stringToBimgType converts string format to bimg.ImageType.
func stringToBimgType(format string) bimg.ImageType {
	switch format {
	case "jpeg", "jpg":
		return bimg.JPEG
	case "png":
		return bimg.PNG
	case "webp":
		return bimg.WEBP
	case "gif":
		return bimg.GIF
	default:
		return bimg.UNKNOWN
	}
}
