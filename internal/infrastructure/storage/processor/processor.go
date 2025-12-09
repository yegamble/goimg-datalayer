//go:build cgo

package processor

import (
	"context"
	"fmt"

	"github.com/h2non/bimg"
)

const (
	// Bytes per megabyte for memory calculations.
	bytesPerMB = 1024 * 1024
)

// Processor handles image processing operations using libvips (via bimg).
// It provides variant generation, EXIF stripping, and format conversion.
type Processor struct {
	config    Config
	semaphore chan struct{} // Limits concurrent operations
}

// New creates a new image processor with the given configuration.
func New(cfg Config) (*Processor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize bimg/libvips memory settings
	bimg.VipsCacheSetMaxMem(cfg.MemoryLimitMB * bytesPerMB)
	bimg.VipsCacheSetMax(0) // Disable operation cache (use memory limit only)

	// Create semaphore for limiting concurrent operations
	semaphore := make(chan struct{}, cfg.MaxConcurrentOps)

	return &Processor{
		config:    cfg,
		semaphore: semaphore,
	}, nil
}

// Process performs the full image processing pipeline on the input data.
// Pipeline steps:
// 1. Decode image and validate format
// 2. Strip EXIF metadata (security/privacy)
// 3. Generate each variant (thumbnail, small, medium, large)
// 4. Re-encode original through libvips (prevent polyglot exploits)
//
// Returns a ProcessResult containing all variants.
//
//nolint:cyclop // Image processing pipeline requires sequential steps: validation, metadata extraction, variant generation, and error handling
func (p *Processor) Process(ctx context.Context, input []byte, _ string) (*ProcessResult, error) {
	// Acquire semaphore slot (limits concurrent operations)
	select {
	case p.semaphore <- struct{}{}:
		defer func() { <-p.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Step 1: Decode and validate image
	img := bimg.NewImage(input)
	size, err := img.Size()
	if err != nil {
		return nil, fmt.Errorf("decode image: %w: %w", ErrProcessingFailed, err)
	}

	// Validate dimensions
	if size.Width <= 0 || size.Height <= 0 {
		return nil, fmt.Errorf("%w: %dx%d", ErrInvalidDimensions, size.Width, size.Height)
	}

	// Minimum dimension check (avoid processing tiny images)
	if size.Width < 10 || size.Height < 10 {
		return nil, fmt.Errorf("%w: minimum 10x10 pixels required", ErrImageTooSmall)
	}

	// Detect format
	imgType := img.Type()
	if imgType == "" {
		return nil, fmt.Errorf("%w: could not detect format", ErrUnsupportedFormat)
	}

	originalFormat := bimgTypeToString(bimg.DetermineImageType(input))
	if !IsSupportedFormat(originalFormat) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, originalFormat)
	}

	result := &ProcessResult{
		OriginalFormat: originalFormat,
		OriginalWidth:  size.Width,
		OriginalHeight: size.Height,
	}

	// Step 2 & 3: Generate all variants
	// Process in order: thumbnail, small, medium, large, original
	variantTypes := []VariantType{
		VariantThumbnail,
		VariantSmall,
		VariantMedium,
		VariantLarge,
		VariantOriginal,
	}

	for _, vt := range variantTypes {
		variantData, err := p.GenerateVariant(ctx, input, vt)
		if err != nil {
			return nil, fmt.Errorf("generate %s variant: %w", vt, err)
		}

		// Store variant in result
		switch vt {
		case VariantThumbnail:
			result.Thumbnail = *variantData
		case VariantSmall:
			result.Small = *variantData
		case VariantMedium:
			result.Medium = *variantData
		case VariantLarge:
			result.Large = *variantData
		case VariantOriginal:
			result.Original = *variantData
		}
	}

	return result, nil
}

// GenerateVariant generates a single image variant from the input data.
// The variant is processed according to the configuration (size, format, quality).
// EXIF metadata is always stripped for security/privacy.
//
//nolint:cyclop // Variant generation requires format-specific processing logic with multiple conditional branches
func (p *Processor) GenerateVariant(ctx context.Context, input []byte, variant VariantType) (*VariantData, error) {
	if !variant.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrInvalidVariantType, variant)
	}

	// Check context before processing
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Get variant specification
	spec := p.config.GetVariantSpec(variant)

	// Create bimg image
	img := bimg.NewImage(input)

	// Get original dimensions
	size, err := img.Size()
	if err != nil {
		return nil, fmt.Errorf("get image size: %w", err)
	}

	// Prepare processing options
	options := bimg.Options{
		Quality:        spec.Quality,
		StripMetadata:  p.config.StripMetadata,
		Interpretation: bimg.InterpretationSRGB, // Ensure sRGB color space
	}

	// Handle animated GIFs (extract first frame)
	if bimg.DetermineImageType(input) == bimg.GIF {
		// For GIFs, we extract the first frame and convert to static image
		options.Type = spec.Format
	}

	// For original variant, re-encode with original format
	if variant == VariantOriginal {
		// Keep original dimensions, just re-encode through libvips
		originalType := bimg.DetermineImageType(input)
		options.Type = originalType

		// Re-encode to prevent polyglot exploits
		processed, err := img.Process(options)
		if err != nil {
			return nil, fmt.Errorf("re-encode original: %w", err)
		}

		format := bimgTypeToString(originalType)
		return &VariantData{
			Data:        processed,
			Width:       size.Width,
			Height:      size.Height,
			Format:      format,
			ContentType: formatToContentType(format),
			FileSize:    int64(len(processed)),
		}, nil
	}

	// Calculate resize dimensions (preserve aspect ratio)
	targetWidth, targetHeight := calculateTargetDimensions(
		size.Width,
		size.Height,
		spec.MaxWidth,
	)

	// Only resize if image is larger than target
	if targetWidth < size.Width {
		options.Width = targetWidth
		options.Height = targetHeight
		options.Enlarge = false // Never enlarge images
		options.Force = false   // Preserve aspect ratio
	}

	// Set output format (WebP for variants)
	options.Type = spec.Format

	// Process the image
	processed, err := img.Process(options)
	if err != nil {
		return nil, fmt.Errorf("process image: %w", err)
	}

	// Get dimensions of processed image
	processedImg := bimg.NewImage(processed)
	processedSize, err := processedImg.Size()
	if err != nil {
		return nil, fmt.Errorf("get processed size: %w", err)
	}

	format := bimgTypeToString(spec.Format)

	return &VariantData{
		Data:        processed,
		Width:       processedSize.Width,
		Height:      processedSize.Height,
		Format:      format,
		ContentType: formatToContentType(format),
		FileSize:    int64(len(processed)),
	}, nil
}

// calculateTargetDimensions calculates the target dimensions for resizing.
// Preserves aspect ratio and never enlarges the image.
func calculateTargetDimensions(originalWidth, originalHeight, maxWidth int) (int, int) {
	// If no max width (original), return original dimensions
	if maxWidth == 0 {
		return originalWidth, originalHeight
	}

	// If image is already smaller, don't resize
	if originalWidth <= maxWidth {
		return originalWidth, originalHeight
	}

	// Calculate aspect ratio
	aspectRatio := float64(originalHeight) / float64(originalWidth)

	// Calculate new dimensions
	targetWidth := maxWidth
	targetHeight := int(float64(targetWidth) * aspectRatio)

	return targetWidth, targetHeight
}

// Shutdown cleans up processor resources.
// Call this when shutting down the application.
func (p *Processor) Shutdown() {
	// Note: bimg/libvips manages its own memory and cache internally.
	// No explicit cleanup is required as libvips handles this automatically.
}
