// Package processor provides image processing capabilities using bimg (libvips).
// It handles variant generation, EXIF stripping, and format conversion.
package processor

import "github.com/h2non/bimg"

const (
	// Default memory limit for bimg cache in megabytes
	DefaultMemoryLimitMB = 256

	// Default maximum number of concurrent processing operations
	DefaultMaxConcurrentOps = 32

	// Quality settings for image variants
	DefaultThumbnailQuality = 82
	DefaultSmallQuality     = 85
	DefaultMediumQuality    = 85
	DefaultLargeQuality     = 88
	DefaultOriginalQuality  = 100

	// Quality bounds
	MinQuality = 1
	MaxQuality = 100

	// Variant dimensions (width in pixels)
	ThumbnailWidth = 160
	SmallWidth     = 320
	MediumWidth    = 800
	LargeWidth     = 1600
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
