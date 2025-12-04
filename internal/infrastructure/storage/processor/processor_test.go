package processor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/processor"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  processor.Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  processor.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: processor.Config{
				MemoryLimitMB:    128,
				MaxConcurrentOps: 16,
				StripMetadata:    true,
				ThumbnailQuality: 80,
				SmallQuality:     85,
				MediumQuality:    85,
				LargeQuality:     90,
				OriginalQuality:  95,
			},
			wantErr: false,
		},
		{
			name: "invalid memory limit",
			config: processor.Config{
				MemoryLimitMB:    0,
				MaxConcurrentOps: 32,
				StripMetadata:    true,
				ThumbnailQuality: 82,
				SmallQuality:     85,
				MediumQuality:    85,
				LargeQuality:     88,
				OriginalQuality:  90,
			},
			wantErr: true,
		},
		{
			name: "invalid quality - too low",
			config: processor.Config{
				MemoryLimitMB:    256,
				MaxConcurrentOps: 32,
				StripMetadata:    true,
				ThumbnailQuality: 0,
				SmallQuality:     85,
				MediumQuality:    85,
				LargeQuality:     88,
				OriginalQuality:  90,
			},
			wantErr: true,
		},
		{
			name: "invalid quality - too high",
			config: processor.Config{
				MemoryLimitMB:    256,
				MaxConcurrentOps: 32,
				StripMetadata:    true,
				ThumbnailQuality: 101,
				SmallQuality:     85,
				MediumQuality:    85,
				LargeQuality:     88,
				OriginalQuality:  90,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := processor.New(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, p)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
				if p != nil {
					p.Shutdown()
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  processor.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  processor.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "zero memory limit",
			config: processor.Config{
				MemoryLimitMB: 0,
			},
			wantErr: true,
		},
		{
			name: "zero concurrent ops",
			config: processor.Config{
				MemoryLimitMB:    256,
				MaxConcurrentOps: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVariantType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		variant processor.VariantType
		want    bool
	}{
		{"thumbnail", processor.VariantThumbnail, true},
		{"small", processor.VariantSmall, true},
		{"medium", processor.VariantMedium, true},
		{"large", processor.VariantLarge, true},
		{"original", processor.VariantOriginal, true},
		{"invalid", processor.VariantType("invalid"), false},
		{"empty", processor.VariantType(""), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.variant.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllVariantTypes(t *testing.T) {
	t.Parallel()

	variants := processor.AllVariantTypes()

	assert.Len(t, variants, 5)
	assert.Contains(t, variants, processor.VariantThumbnail)
	assert.Contains(t, variants, processor.VariantSmall)
	assert.Contains(t, variants, processor.VariantMedium)
	assert.Contains(t, variants, processor.VariantLarge)
	assert.Contains(t, variants, processor.VariantOriginal)
}

func TestProcessResult_GetVariant(t *testing.T) {
	t.Parallel()

	result := &processor.ProcessResult{
		Thumbnail: processor.VariantData{Width: 160},
		Small:     processor.VariantData{Width: 320},
		Medium:    processor.VariantData{Width: 800},
		Large:     processor.VariantData{Width: 1600},
		Original:  processor.VariantData{Width: 3000},
	}

	tests := []struct {
		name        string
		variantType processor.VariantType
		wantWidth   int
		wantErr     bool
	}{
		{"thumbnail", processor.VariantThumbnail, 160, false},
		{"small", processor.VariantSmall, 320, false},
		{"medium", processor.VariantMedium, 800, false},
		{"large", processor.VariantLarge, 1600, false},
		{"original", processor.VariantOriginal, 3000, false},
		{"invalid", processor.VariantType("invalid"), 0, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			variant, err := result.GetVariant(tt.variantType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, variant)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, variant)
				assert.Equal(t, tt.wantWidth, variant.Width)
			}
		})
	}
}

func TestConfig_GetVariantSpec(t *testing.T) {
	t.Parallel()

	cfg := processor.DefaultConfig()

	tests := []struct {
		name         string
		variantType  processor.VariantType
		wantMaxWidth int
		wantQuality  int
	}{
		{"thumbnail", processor.VariantThumbnail, 160, cfg.ThumbnailQuality},
		{"small", processor.VariantSmall, 320, cfg.SmallQuality},
		{"medium", processor.VariantMedium, 800, cfg.MediumQuality},
		{"large", processor.VariantLarge, 1600, cfg.LargeQuality},
		{"original", processor.VariantOriginal, 0, cfg.OriginalQuality},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := cfg.GetVariantSpec(tt.variantType)

			assert.Equal(t, tt.wantMaxWidth, spec.MaxWidth)
			assert.Equal(t, tt.wantQuality, spec.Quality)
		})
	}
}

func TestSupportedFormats(t *testing.T) {
	t.Parallel()

	formats := processor.SupportedFormats()

	assert.Len(t, formats, 4)
	assert.Contains(t, formats, "jpeg")
	assert.Contains(t, formats, "png")
	assert.Contains(t, formats, "gif")
	assert.Contains(t, formats, "webp")
}

func TestIsSupportedFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format string
		want   bool
	}{
		{"jpeg", true},
		{"png", true},
		{"gif", true},
		{"webp", true},
		{"bmp", false},
		{"tiff", false},
		{"", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()

			got := processor.IsSupportedFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Integration tests with actual image processing
// These require libvips to be installed

func TestProcessor_Process_Integration(t *testing.T) {
	// Skip if libvips is not available
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a simple test image (1x1 PNG)
	// This is a minimal valid PNG image
	testPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG header
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x64, // 100x100
		0x08, 0x02, 0x00, 0x00, 0x00, 0xFF, 0x80, 0x02, 0x03,
	}

	// For real testing, we need a valid image file
	// Let's load a test image if available
	testImagePath := filepath.Join("testdata", "test.jpg")
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Skip("test image not found, skipping integration test")
		return
	}

	testImage, err := os.ReadFile(testImagePath)
	require.NoError(t, err, "failed to read test image")

	cfg := processor.DefaultConfig()
	p, err := processor.New(cfg)
	require.NoError(t, err)
	defer p.Shutdown()

	ctx := context.Background()
	result, err := p.Process(ctx, testImage, "test.jpg")
	require.NoError(t, err)

	// Verify all variants were generated
	assert.NotEmpty(t, result.Thumbnail.Data)
	assert.NotEmpty(t, result.Small.Data)
	assert.NotEmpty(t, result.Medium.Data)
	assert.NotEmpty(t, result.Large.Data)
	assert.NotEmpty(t, result.Original.Data)

	// Verify dimensions are correct
	assert.True(t, result.Thumbnail.Width <= 160)
	assert.True(t, result.Small.Width <= 320)
	assert.True(t, result.Medium.Width <= 800)
	assert.True(t, result.Large.Width <= 1600)

	// Verify formats
	assert.Equal(t, "webp", result.Thumbnail.Format)
	assert.Equal(t, "webp", result.Small.Format)
	assert.Equal(t, "webp", result.Medium.Format)
	assert.Equal(t, "webp", result.Large.Format)

	// Verify file sizes
	assert.Greater(t, result.Thumbnail.FileSize, int64(0))
	assert.Greater(t, result.Small.FileSize, int64(0))
	assert.Greater(t, result.Medium.FileSize, int64(0))
	assert.Greater(t, result.Large.FileSize, int64(0))
	assert.Greater(t, result.Original.FileSize, int64(0))

	// Verify content types
	assert.Equal(t, "image/webp", result.Thumbnail.ContentType)
	assert.Equal(t, "image/webp", result.Small.ContentType)
	assert.Equal(t, "image/webp", result.Medium.ContentType)
	assert.Equal(t, "image/webp", result.Large.ContentType)
}

func TestProcessor_GenerateVariant_InvalidInput(t *testing.T) {
	t.Parallel()

	cfg := processor.DefaultConfig()
	p, err := processor.New(cfg)
	require.NoError(t, err)
	defer p.Shutdown()

	ctx := context.Background()

	tests := []struct {
		name        string
		input       []byte
		variantType processor.VariantType
		wantErr     error
	}{
		{
			name:        "empty input",
			input:       []byte{},
			variantType: processor.VariantThumbnail,
			wantErr:     processor.ErrProcessingFailed,
		},
		{
			name:        "invalid variant type",
			input:       []byte("not an image"),
			variantType: processor.VariantType("invalid"),
			wantErr:     processor.ErrInvalidVariantType,
		},
		{
			name:        "corrupted data",
			input:       []byte{0x00, 0x01, 0x02, 0x03},
			variantType: processor.VariantThumbnail,
			wantErr:     nil, // Will fail during processing
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Note: Can't run in parallel because we're using a shared processor

			_, err := p.GenerateVariant(ctx, tt.input, tt.variantType)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.Error(t, err) // Should error, but not a specific type
			}
		})
	}
}

func TestProcessor_Process_ContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := processor.DefaultConfig()
	p, err := processor.New(cfg)
	require.NoError(t, err)
	defer p.Shutdown()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	testImage := []byte("not important for this test")

	_, err = p.Process(ctx, testImage, "test.jpg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}
