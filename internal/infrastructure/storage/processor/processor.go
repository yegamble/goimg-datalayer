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
func (p *Processor) Process(ctx context.Context, input []byte, _ string) (*ProcessResult, error) {
	// Acquire semaphore slot (limits concurrent operations)
	select {
	case p.semaphore <- struct{}{}:
		defer func() { <-p.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Step 1: Validate image dimensions
	size, err := p.validateImageDimensions(input)
	if err != nil {
		return nil, err
	}

	// Step 2: Detect and validate format
	originalFormat, err := p.detectAndValidateFormat(input)
	if err != nil {
		return nil, err
	}

	// Step 3: Generate all variants
	result, err := p.generateAllVariants(ctx, input, originalFormat, size)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// validateImageDimensions validates the image can be decoded and has valid dimensions.
func (p *Processor) validateImageDimensions(input []byte) (bimg.ImageSize, error) {
	img := bimg.NewImage(input)
	size, err := img.Size()
	if err != nil {
		return bimg.ImageSize{}, fmt.Errorf("decode image: %w: %w", ErrProcessingFailed, err)
	}

	// Validate dimensions
	if size.Width <= 0 || size.Height <= 0 {
		return bimg.ImageSize{}, fmt.Errorf("%w: %dx%d", ErrInvalidDimensions, size.Width, size.Height)
	}

	// Minimum dimension check (avoid processing tiny images)
	if size.Width < 10 || size.Height < 10 {
		return bimg.ImageSize{}, fmt.Errorf("%w: minimum 10x10 pixels required", ErrImageTooSmall)
	}

	return size, nil
}

// detectAndValidateFormat detects the image format and ensures it's supported.
func (p *Processor) detectAndValidateFormat(input []byte) (string, error) {
	img := bimg.NewImage(input)
	imgType := img.Type()
	if imgType == "" {
		return "", fmt.Errorf("%w: could not detect format", ErrUnsupportedFormat)
	}

	originalFormat := bimgTypeToString(bimg.DetermineImageType(input))
	if !IsSupportedFormat(originalFormat) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, originalFormat)
	}

	return originalFormat, nil
}

// generateAllVariants generates all image variants and returns the result.
func (p *Processor) generateAllVariants(
	ctx context.Context,
	input []byte,
	originalFormat string,
	size bimg.ImageSize,
) (*ProcessResult, error) {
	result := &ProcessResult{
		OriginalFormat: originalFormat,
		OriginalWidth:  size.Width,
		OriginalHeight: size.Height,
	}

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

	// For original variant, use special processing
	if variant == VariantOriginal {
		return p.processOriginalVariant(input, img, size, spec)
	}

	// For standard variants, apply resize and format conversion
	return p.processStandardVariant(input, img, size, spec)
}

// processOriginalVariant processes the original variant by re-encoding through libvips.
func (p *Processor) processOriginalVariant(
	input []byte,
	img *bimg.Image,
	size bimg.ImageSize,
	spec VariantSpec,
) (*VariantData, error) {
	// Keep original dimensions, just re-encode through libvips
	originalType := bimg.DetermineImageType(input)

	options := bimg.Options{
		Quality:        spec.Quality,
		StripMetadata:  p.config.StripMetadata,
		Interpretation: bimg.InterpretationSRGB,
		Type:           originalType,
	}

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

// processStandardVariant processes a standard variant with resize and format conversion.
func (p *Processor) processStandardVariant(
	input []byte,
	img *bimg.Image,
	size bimg.ImageSize,
	spec VariantSpec,
) (*VariantData, error) {
	// Prepare processing options
	options := bimg.Options{
		Quality:        spec.Quality,
		StripMetadata:  p.config.StripMetadata,
		Interpretation: bimg.InterpretationSRGB,
		Type:           spec.Format,
	}

	// Handle animated GIFs (extract first frame)
	if bimg.DetermineImageType(input) == bimg.GIF {
		options.Type = spec.Format
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

// CustomVariantOptions defines parameters for custom variant generation.
type CustomVariantOptions struct {
	MaxWidth  int
	MaxHeight int
	Format    string // jpeg, png, webp, avif
	Quality   int    // 1-100
	CropMode  string // fit, fill, crop
}

// CustomVariantData represents the output of custom variant generation.
type CustomVariantData struct {
	Data        []byte
	Width       int
	Height      int
	Format      string
	ContentType string
	FileSize    int64
}

// GenerateCustomVariant generates a single custom variant with user-specified parameters.
// This method supports custom dimensions, format, quality, and crop mode.
//
// Parameters:
//   - input: Raw image bytes
//   - maxWidth: Maximum width in pixels (1-8192)
//   - maxHeight: Maximum height in pixels (1-8192)
//   - format: Output format (jpeg, png, webp, avif)
//   - quality: Compression quality (1-100)
//   - cropMode: Resize behavior (fit, fill, crop)
//
// Crop modes:
//   - fit: Resize to fit within bounds, preserving aspect ratio (may not fill bounds)
//   - fill: Resize to fill bounds, preserving aspect ratio (may crop)
//   - crop: Resize and center-crop to exact dimensions
func (p *Processor) GenerateCustomVariant(
	ctx context.Context,
	input []byte,
	maxWidth, maxHeight int,
	format string,
	quality int,
	cropMode string,
) (*CustomVariantData, error) {
	// Acquire semaphore slot (limits concurrent operations)
	select {
	case p.semaphore <- struct{}{}:
		defer func() { <-p.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Validate parameters
	if maxWidth < 1 || maxWidth > 8192 {
		return nil, fmt.Errorf("%w: max_width must be between 1 and 8192", ErrInvalidDimensions)
	}
	if maxHeight < 1 || maxHeight > 8192 {
		return nil, fmt.Errorf("%w: max_height must be between 1 and 8192", ErrInvalidDimensions)
	}
	if quality < 1 || quality > 100 {
		return nil, fmt.Errorf("%w: quality must be between 1 and 100", ErrInvalidConfig)
	}

	// Convert format string to bimg type
	outputFormat, err := stringToBimgType(format)
	if err != nil {
		return nil, err
	}

	// Create bimg image
	img := bimg.NewImage(input)

	// Get original dimensions
	size, err := img.Size()
	if err != nil {
		return nil, fmt.Errorf("get image size: %w", err)
	}

	// Calculate target dimensions based on crop mode
	targetWidth, targetHeight := calculateCustomDimensions(
		size.Width, size.Height,
		maxWidth, maxHeight,
		cropMode,
	)

	// Build processing options
	options := bimg.Options{
		Width:         targetWidth,
		Height:        targetHeight,
		Quality:       quality,
		StripMetadata: p.config.StripMetadata,
		Type:          outputFormat,
	}

	// Apply crop mode specific options
	switch cropMode {
	case "fill", "crop":
		// For fill/crop: resize and crop to exact dimensions
		options.Crop = true
		options.Gravity = bimg.GravityCentre
	case "fit":
		// For fit: resize to fit within bounds, preserving aspect ratio
		options.Crop = false
		options.Force = false
		options.Enlarge = false
	default:
		// Default to fit behavior
		options.Crop = false
		options.Force = false
		options.Enlarge = false
	}

	// Process the image
	processed, err := img.Process(options)
	if err != nil {
		return nil, fmt.Errorf("process image: %w: %w", ErrProcessingFailed, err)
	}

	// Get dimensions of processed image
	processedImg := bimg.NewImage(processed)
	processedSize, err := processedImg.Size()
	if err != nil {
		return nil, fmt.Errorf("get processed size: %w", err)
	}

	formatStr := bimgTypeToString(outputFormat)

	return &CustomVariantData{
		Data:        processed,
		Width:       processedSize.Width,
		Height:      processedSize.Height,
		Format:      formatStr,
		ContentType: formatToContentType(formatStr),
		FileSize:    int64(len(processed)),
	}, nil
}

// stringToBimgType converts a format string to bimg.ImageType.
func stringToBimgType(format string) (bimg.ImageType, error) {
	switch format {
	case "jpeg", "jpg":
		return bimg.JPEG, nil
	case "png":
		return bimg.PNG, nil
	case "webp":
		return bimg.WEBP, nil
	case "avif":
		return bimg.AVIF, nil
	case "gif":
		return bimg.GIF, nil
	default:
		return bimg.UNKNOWN, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// calculateCustomDimensions calculates target dimensions based on crop mode.
func calculateCustomDimensions(
	originalWidth, originalHeight int,
	maxWidth, maxHeight int,
	cropMode string,
) (int, int) {
	aspectRatio := float64(originalWidth) / float64(originalHeight)
	targetAspect := float64(maxWidth) / float64(maxHeight)

	switch cropMode {
	case "fill", "crop":
		// Fill/crop: resize to fill bounds, then crop
		// Return exact dimensions - bimg will handle the crop
		return maxWidth, maxHeight

	case "fit":
		// Fit: resize to fit within bounds, preserving aspect ratio
		if aspectRatio > targetAspect {
			// Image is wider than target - constrain by width
			return maxWidth, int(float64(maxWidth) / aspectRatio)
		}
		// Image is taller than target - constrain by height
		return int(float64(maxHeight) * aspectRatio), maxHeight

	default:
		// Default to fit behavior
		if aspectRatio > targetAspect {
			return maxWidth, int(float64(maxWidth) / aspectRatio)
		}
		return int(float64(maxHeight) * aspectRatio), maxHeight
	}
}
