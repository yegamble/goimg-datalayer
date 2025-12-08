package processor_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/processor"
)

// Example_processor demonstrates basic usage of the image processor.
func Example_processor() {
	// Create processor with default configuration
	cfg := processor.DefaultConfig()
	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer proc.Shutdown()

	// Load an image file
	imageData, err := os.ReadFile("photo.jpg")
	if err != nil {
		panic(err)
	}

	// Process the image to generate all variants
	ctx := context.Background()
	result, err := proc.Process(ctx, imageData, "photo.jpg")
	if err != nil {
		panic(err)
	}

	// Print information about generated variants
	fmt.Printf("Original: %dx%d (%s)\n",
		result.OriginalWidth,
		result.OriginalHeight,
		result.OriginalFormat)

	fmt.Printf("Thumbnail: %dx%d, %d bytes\n",
		result.Thumbnail.Width,
		result.Thumbnail.Height,
		result.Thumbnail.FileSize)

	fmt.Printf("Small: %dx%d, %d bytes\n",
		result.Small.Width,
		result.Small.Height,
		result.Small.FileSize)

	fmt.Printf("Medium: %dx%d, %d bytes\n",
		result.Medium.Width,
		result.Medium.Height,
		result.Medium.FileSize)

	fmt.Printf("Large: %dx%d, %d bytes\n",
		result.Large.Width,
		result.Large.Height,
		result.Large.FileSize)

	// Save variants to disk
	variants := map[string]processor.VariantData{
		"thumbnail": result.Thumbnail,
		"small":     result.Small,
		"medium":    result.Medium,
		"large":     result.Large,
		"original":  result.Original,
	}

	for name, variant := range variants {
		filename := fmt.Sprintf("photo_%s.%s", name, variant.Format)
		//nolint:gosec // G306: Example test code with appropriate permissions for test output
		if err := os.WriteFile(filename, variant.Data, 0644); err != nil {
			log.Printf("Failed to save %s: %v\n", name, err)
		}
	}
	// Output:
}

// Example_processor_customConfig demonstrates using custom configuration.
func Example_processor_customConfig() {
	// Create custom configuration
	cfg := processor.Config{
		MemoryLimitMB:    512, // Increase memory limit
		MaxConcurrentOps: 64,  // Allow more concurrent operations
		StripMetadata:    true,
		ThumbnailQuality: 80,
		SmallQuality:     85,
		MediumQuality:    90,
		LargeQuality:     95,
		OriginalQuality:  100, // Maximum quality for originals
	}

	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer proc.Shutdown()

	// Use the processor...
	_ = proc
	// Output:
}

// Example_processor_GenerateVariant demonstrates generating a single variant.
func Example_processor_GenerateVariant() {
	cfg := processor.DefaultConfig()
	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer proc.Shutdown()

	imageData, err := os.ReadFile("photo.jpg")
	if err != nil {
		panic(err)
	}

	// Generate just a thumbnail
	ctx := context.Background()
	thumbnail, err := proc.GenerateVariant(ctx, imageData, processor.VariantThumbnail)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Thumbnail: %dx%d, %s, %d bytes\n",
		thumbnail.Width,
		thumbnail.Height,
		thumbnail.Format,
		thumbnail.FileSize)

	// Save the thumbnail
	//nolint:gosec // G306: Example test code with appropriate permissions for test output
	if err := os.WriteFile("thumbnail.webp", thumbnail.Data, 0644); err != nil {
		panic(err)
	}
	// Output:
}

// Example_processResult_GetVariant demonstrates accessing specific variants.
func Example_processResult_GetVariant() {
	cfg := processor.DefaultConfig()
	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer proc.Shutdown()

	imageData, err := os.ReadFile("photo.jpg")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	result, err := proc.Process(ctx, imageData, "photo.jpg")
	if err != nil {
		panic(err)
	}

	// Get specific variant
	medium, err := result.GetVariant(processor.VariantMedium)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Medium variant: %dx%d\n", medium.Width, medium.Height)
	// Output:
}

// ExampleConfig_GetVariantSpec demonstrates getting variant specifications.
func ExampleConfig_GetVariantSpec() {
	cfg := processor.DefaultConfig()

	// Get specifications for each variant type
	for _, vt := range processor.AllVariantTypes() {
		spec := cfg.GetVariantSpec(vt)
		fmt.Printf("%s: max width=%d, quality=%d\n",
			vt,
			spec.MaxWidth,
			spec.Quality)
	}

	// Output:
	// thumbnail: max width=160, quality=82
	// small: max width=320, quality=85
	// medium: max width=800, quality=85
	// large: max width=1600, quality=88
	// original: max width=0, quality=100
}

// ExampleAllVariantTypes demonstrates listing all variant types.
func ExampleAllVariantTypes() {
	variants := processor.AllVariantTypes()

	for _, vt := range variants {
		fmt.Printf("- %s (valid: %t)\n", vt, vt.IsValid())
	}

	// Output:
	// - thumbnail (valid: true)
	// - small (valid: true)
	// - medium (valid: true)
	// - large (valid: true)
	// - original (valid: true)
}

// ExampleSupportedFormats demonstrates checking supported formats.
func ExampleSupportedFormats() {
	formats := processor.SupportedFormats()

	for _, format := range formats {
		fmt.Printf("- %s (supported: %t)\n",
			format,
			processor.IsSupportedFormat(format))
	}

	// Check unsupported format
	fmt.Printf("- bmp (supported: %t)\n",
		processor.IsSupportedFormat("bmp"))

	// Output:
	// - jpeg (supported: true)
	// - png (supported: true)
	// - gif (supported: true)
	// - webp (supported: true)
	// - bmp (supported: false)
}
