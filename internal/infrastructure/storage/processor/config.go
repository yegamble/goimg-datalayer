//go:build cgo

// Package processor provides image processing capabilities using bimg (libvips).
// It handles variant generation, EXIF stripping, and format conversion.
package processor

import "github.com/h2non/bimg"

const (
	// DefaultMemoryLimitMB is the default memory limit for bimg cache in megabytes.
	DefaultMemoryLimitMB = 256

	// DefaultMaxConcurrentOps is the default maximum number of concurrent processing operations.
	DefaultMaxConcurrentOps = 32

	// DefaultThumbnailQuality is the quality setting for thumbnail variants.
	DefaultThumbnailQuality = 82
	// DefaultSmallQuality is the quality setting for small variants.
	DefaultSmallQuality = 85
	// DefaultMediumQuality is the quality setting for medium variants.
	DefaultMediumQuality = 85
	// DefaultLargeQuality is the quality setting for large variants.
	DefaultLargeQuality = 88
	// DefaultOriginalQuality is the quality setting for original variants.
	DefaultOriginalQuality = 100

	// MinQuality is the minimum allowed quality value.
	MinQuality = 1
	// MaxQuality is the maximum allowed quality value.
	MaxQuality = 100

	// ThumbnailWidth is the width in pixels for thumbnail variants.
	ThumbnailWidth = 160
	// SmallWidth is the width in pixels for small variants.
	SmallWidth = 320
	// MediumWidth is the width in pixels for medium variants.
	MediumWidth = 800
	// LargeWidth is the width in pixels for large variants.
	LargeWidth = 1600
)

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
		MemoryLimitMB:    DefaultMemoryLimitMB,
		MaxConcurrentOps: DefaultMaxConcurrentOps,
		StripMetadata:    true,
		ThumbnailQuality: DefaultThumbnailQuality,
		SmallQuality:     DefaultSmallQuality,
		MediumQuality:    DefaultMediumQuality,
		LargeQuality:     DefaultLargeQuality,
		OriginalQuality:  DefaultOriginalQuality,
	}
}

// Validate ensures the configuration is valid.
func (c Config) Validate() error {
	if err := c.validateMemory(); err != nil {
		return err
	}
	if err := c.validateQualitySettings(); err != nil {
		return err
	}
	return nil
}

func (c Config) validateMemory() error {
	if c.MemoryLimitMB <= 0 {
		return ErrInvalidConfig
	}
	if c.MaxConcurrentOps <= 0 {
		return ErrInvalidConfig
	}
	return nil
}

func (c Config) validateQualitySettings() error {
	qualities := []int{
		c.ThumbnailQuality,
		c.SmallQuality,
		c.MediumQuality,
		c.LargeQuality,
		c.OriginalQuality,
	}

	for _, quality := range qualities {
		if quality < MinQuality || quality > MaxQuality {
			return ErrInvalidConfig
		}
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
			MaxWidth: ThumbnailWidth,
			Format:   bimg.WEBP,
			Quality:  c.ThumbnailQuality,
		}
	case VariantSmall:
		return VariantSpec{
			MaxWidth: SmallWidth,
			Format:   bimg.WEBP,
			Quality:  c.SmallQuality,
		}
	case VariantMedium:
		return VariantSpec{
			MaxWidth: MediumWidth,
			Format:   bimg.WEBP,
			Quality:  c.MediumQuality,
		}
	case VariantLarge:
		return VariantSpec{
			MaxWidth: LargeWidth,
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
			MaxWidth: MediumWidth,
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
//
//nolint:cyclop // Exhaustive switch mapping all supported bimg image format types
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
	case bimg.TIFF:
		return "tiff"
	case bimg.PDF:
		return "pdf"
	case bimg.SVG:
		return "svg"
	case bimg.MAGICK:
		return "magick"
	case bimg.HEIF:
		return "heif"
	case bimg.AVIF:
		return "avif"
	case bimg.UNKNOWN:
		return "unknown"
	default:
		// Handle any other unsupported image types
		return "unknown"
	}
}
