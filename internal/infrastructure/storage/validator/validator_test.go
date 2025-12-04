package validator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, int64(10*1024*1024), cfg.MaxFileSize)
	assert.Equal(t, 8192, cfg.MaxWidth)
	assert.Equal(t, 8192, cfg.MaxHeight)
	assert.Equal(t, int64(100_000_000), cfg.MaxPixels)
	assert.Equal(t, []string{"image/jpeg", "image/png", "image/gif", "image/webp"}, cfg.AllowedMIMETypes)
	assert.True(t, cfg.EnableMalwareScan)
}

// TestNew_WithCustomConfig tests validator creation with custom config.
func TestNew_WithCustomConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxFileSize:      5 * 1024 * 1024,
		MaxWidth:         4096,
		MaxHeight:        4096,
		MaxPixels:        50_000_000,
		AllowedMIMETypes: []string{"image/jpeg", "image/png"},
		EnableMalwareScan: false,
	}

	v := New(cfg, nil)
	require.NotNil(t, v)
	assert.Equal(t, cfg.MaxFileSize, v.config.MaxFileSize)
	assert.Equal(t, cfg.MaxWidth, v.config.MaxWidth)
}

// TestNew_DefaultsAllowedMIMETypes tests that empty AllowedMIMETypes gets defaults.
func TestNew_DefaultsAllowedMIMETypes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxFileSize:       10 * 1024 * 1024,
		AllowedMIMETypes:  []string{}, // Empty
		EnableMalwareScan: false,
	}

	v := New(cfg, nil)
	require.NotNil(t, v)
	assert.NotEmpty(t, v.config.AllowedMIMETypes)
	assert.Contains(t, v.config.AllowedMIMETypes, "image/jpeg")
}

// TestValidateSize_Success tests successful size validation.
func TestValidateSize_Success(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: false,
	}
	v := New(cfg, nil)

	result := &ValidationResult{
		FileSize: 512, // Under limit
	}

	err := v.validateSize(result)
	assert.NoError(t, err)
}

// TestValidateSize_ExceedsLimit tests size validation when file is too large.
func TestValidateSize_ExceedsLimit(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: false,
	}
	v := New(cfg, nil)

	result := &ValidationResult{
		FileSize: 2048, // Over limit
	}

	err := v.validateSize(result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gallery.ErrFileTooLarge))
	assert.Contains(t, err.Error(), "2048 bytes exceeds 1024 byte limit")
}

// TestValidateMIMEType_Valid tests MIME type validation with allowed types.
func TestValidateMIMEType_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mimeType string
	}{
		{"jpeg", "image/jpeg"},
		{"png", "image/png"},
		{"gif", "image/gif"},
		{"webp", "image/webp"},
		{"with charset", "image/jpeg; charset=utf-8"},
		{"with whitespace", " image/png "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			v := New(cfg, nil)

			err := v.validateMIMEType(tt.mimeType)
			assert.NoError(t, err)
		})
	}
}

// TestValidateMIMEType_Invalid tests MIME type validation with disallowed types.
func TestValidateMIMEType_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mimeType string
	}{
		{"text file", "text/plain"},
		{"pdf", "application/pdf"},
		{"executable", "application/x-executable"},
		{"video", "video/mp4"},
		{"audio", "audio/mpeg"},
		{"svg", "image/svg+xml"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			v := New(cfg, nil)

			err := v.validateMIMEType(tt.mimeType)
			require.Error(t, err)
			assert.True(t, errors.Is(err, gallery.ErrInvalidMimeType))
		})
	}
}

// TestValidateMagicBytes_ValidFormats tests magic byte validation with valid formats.
func TestValidateMagicBytes_ValidFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "JPEG",
			data: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
		},
		{
			name: "PNG",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
		},
		{
			name: "GIF87a",
			data: []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "GIF89a",
			data: []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "WebP",
			data: []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			v := New(cfg, nil)

			err := v.validateMagicBytes(tt.data)
			assert.NoError(t, err)
		})
	}
}

// TestValidateMagicBytes_Invalid tests magic byte validation with invalid data.
func TestValidateMagicBytes_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "text file",
			data: []byte("This is a text file"),
		},
		{
			name: "PDF file",
			data: []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "ZIP file",
			data: []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "too small",
			data: []byte{0xFF, 0xD8},
		},
		{
			name: "fake WebP (missing WEBP at offset 8)",
			data: []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x46, 0x41, 0x4B, 0x45},
		},
		{
			name: "empty",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			v := New(cfg, nil)

			err := v.validateMagicBytes(tt.data)
			require.Error(t, err)
			assert.True(t, errors.Is(err, gallery.ErrInvalidMimeType))
		})
	}
}

// TestValidateDimensions_Success tests successful dimension validation.
func TestValidateDimensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"small image", 100, 100},
		{"wide image", 4096, 1024},
		{"tall image", 1024, 4096},
		{"max allowed", 8192, 8192},
		{"single pixel", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			v := New(cfg, nil)

			err := v.ValidateDimensions(tt.width, tt.height)
			assert.NoError(t, err)
		})
	}
}

// TestValidateDimensions_Invalid tests dimension validation with invalid dimensions.
func TestValidateDimensions_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		width     int
		height    int
		maxPixels int64
		wantError error
	}{
		{
			name:      "zero width",
			width:     0,
			height:    100,
			maxPixels: 100_000_000,
			wantError: gallery.ErrInvalidDimensions,
		},
		{
			name:      "zero height",
			width:     100,
			height:    0,
			maxPixels: 100_000_000,
			wantError: gallery.ErrInvalidDimensions,
		},
		{
			name:      "negative width",
			width:     -100,
			height:    100,
			maxPixels: 100_000_000,
			wantError: gallery.ErrInvalidDimensions,
		},
		{
			name:      "negative height",
			width:     100,
			height:    -100,
			maxPixels: 100_000_000,
			wantError: gallery.ErrInvalidDimensions,
		},
		{
			name:      "width exceeds max",
			width:     9000,
			height:    1000,
			maxPixels: 100_000_000,
			wantError: gallery.ErrImageTooLarge,
		},
		{
			name:      "height exceeds max",
			width:     1000,
			height:    9000,
			maxPixels: 100_000_000,
			wantError: gallery.ErrImageTooLarge,
		},
		{
			name:      "both exceed max",
			width:     10000,
			height:    10000,
			maxPixels: 100_000_000,
			wantError: gallery.ErrImageTooLarge,
		},
		{
			name:      "pixel count exceeds max (decompression bomb)",
			width:     8000,
			height:    8000,
			maxPixels: 50_000_000, // 8000 * 8000 = 64M pixels > 50M limit
			wantError: gallery.ErrImageTooManyPixels,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			cfg.MaxPixels = tt.maxPixels
			v := New(cfg, nil)

			err := v.ValidateDimensions(tt.width, tt.height)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantError), "expected %v, got %v", tt.wantError, err)
		})
	}
}

// TestValidate_JPEGSuccess tests full validation pipeline with valid JPEG.
func TestValidate_JPEGSuccess(t *testing.T) {
	t.Parallel()

	// Minimal valid JPEG
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: false, // Disable for basic test
	}
	v := New(cfg, nil)

	result, err := v.Validate(context.Background(), jpegData, "test.jpg")
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "image/jpeg", result.MIMEType)
	assert.Equal(t, int64(len(jpegData)), result.FileSize)
}

// TestValidate_PNGSuccess tests full validation pipeline with valid PNG.
func TestValidate_PNGSuccess(t *testing.T) {
	t.Parallel()

	// Minimal valid PNG
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/png"},
		EnableMalwareScan: false,
	}
	v := New(cfg, nil)

	result, err := v.Validate(context.Background(), pngData, "test.png")
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "image/png", result.MIMEType)
}

// TestValidate_SizeExceeded tests validation failure for oversized file.
func TestValidate_SizeExceeded(t *testing.T) {
	t.Parallel()

	// Create data that exceeds limit
	data := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, 500) // 1500 bytes

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: false,
	}
	v := New(cfg, nil)

	result, err := v.Validate(context.Background(), data, "large.jpg")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gallery.ErrFileTooLarge))
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestValidate_InvalidMIMEType tests validation failure for unsupported MIME type.
func TestValidate_InvalidMIMEType(t *testing.T) {
	t.Parallel()

	// Text data (not an image)
	data := []byte("This is not an image file")

	cfg := DefaultConfig()
	v := New(cfg, nil)

	result, err := v.Validate(context.Background(), data, "fake.jpg")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gallery.ErrInvalidMimeType))
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestValidate_InvalidMagicBytes tests validation failure for invalid magic bytes.
func TestValidate_InvalidMagicBytes(t *testing.T) {
	t.Parallel()

	// PDF file (valid header, but not allowed)
	data := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34, 0x00, 0x00, 0x00, 0x00}

	cfg := DefaultConfig()
	v := New(cfg, nil)

	result, err := v.Validate(context.Background(), data, "document.jpg")
	require.Error(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestValidate_WithMalwareScan tests validation with ClamAV scanning.
func TestValidate_WithMalwareScan(t *testing.T) {
	t.Parallel()

	// Valid JPEG data
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}

	mockScanner := &mockClamAVScanner{
		scanResult: &clamav.ScanResult{
			Infected: false,
			Virus:    "",
		},
		scanError: nil,
	}

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: true,
	}
	v := New(cfg, mockScanner)

	result, err := v.Validate(context.Background(), jpegData, "test.jpg")
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.NotNil(t, result.ScanResult)
	assert.False(t, result.ScanResult.Infected)
}

// TestValidate_MalwareDetected tests validation failure when malware is detected.
func TestValidate_MalwareDetected(t *testing.T) {
	t.Parallel()

	// Valid JPEG data (but will be flagged as infected)
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}

	mockScanner := &mockClamAVScanner{
		scanResult: &clamav.ScanResult{
			Infected: true,
			Virus:    "Test.EICAR.Signature",
		},
		scanError: nil,
	}

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: true,
	}
	v := New(cfg, mockScanner)

	result, err := v.Validate(context.Background(), jpegData, "malware.jpg")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gallery.ErrMalwareDetected))
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.NotNil(t, result.ScanResult)
	assert.True(t, result.ScanResult.Infected)
	assert.Equal(t, "Test.EICAR.Signature", result.ScanResult.Virus)
}

// TestValidate_MalwareScanFailed tests validation failure when scan itself fails.
func TestValidate_MalwareScanFailed(t *testing.T) {
	t.Parallel()

	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}

	mockScanner := &mockClamAVScanner{
		scanResult: nil,
		scanError:  errors.New("ClamAV connection failed"),
	}

	cfg := Config{
		MaxFileSize:       1024,
		AllowedMIMETypes:  []string{"image/jpeg"},
		EnableMalwareScan: true,
	}
	v := New(cfg, mockScanner)

	result, err := v.Validate(context.Background(), jpegData, "test.jpg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malware scan failed")
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestSanitizeFilename tests filename sanitization.
func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean filename",
			input:    "photo.jpg",
			expected: "photo.jpg",
		},
		{
			name:     "spaces replaced",
			input:    "my photo.jpg",
			expected: "my_photo.jpg",
		},
		{
			name:     "unsafe characters removed",
			input:    "file<>:\"/\\|?*.jpg",
			expected: "____.jpg", // path.Base sees / and \ as separators
		},
		{
			name:     "path traversal removed",
			input:    "../../etc/passwd",
			expected: "passwd", // path.Base returns the last element
		},
		{
			name:     "leading dots removed",
			input:    "...hidden.jpg",
			expected: "hidden.jpg",
		},
		{
			name:     "trailing dots removed",
			input:    "file...",
			expected: "file",
		},
		{
			name:     "path component ignored",
			input:    "/path/to/file.jpg",
			expected: "file.jpg",
		},
		{
			name:     "windows path ignored",
			input:    "C:\\Users\\file.jpg",
			expected: "C__Users_file.jpg", // On Unix, \ is not a path separator
		},
		{
			name:     "control characters removed",
			input:    "file\x00\x01\x1F.jpg",
			expected: "file___.jpg",
		},
		{
			name:     "long filename truncated",
			input:    strings.Repeat("a", 250) + ".jpg",
			expected: strings.Repeat("a", 196) + ".jpg", // 200 total - 4 for ".jpg"
		},
		{
			name:     "empty becomes unnamed",
			input:    "",
			expected: "unnamed",
		},
		{
			name:     "just dots becomes unnamed",
			input:    "...",
			expected: "unnamed",
		},
		{
			name:     "unicode characters preserved",
			input:    "фото.jpg",
			expected: "фото.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeFilename_Security tests security aspects of filename sanitization.
func TestSanitizeFilename_Security(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		shouldNotContain []string
	}{
		{
			name:        "removes path traversal",
			input:       "../../../etc/passwd",
			shouldNotContain: []string{"..", "/", "\\"},
		},
		{
			name:        "removes null bytes",
			input:       "file\x00.jpg",
			shouldNotContain: []string{"\x00"},
		},
		{
			name:        "removes control characters",
			input:       "file\r\n\t.jpg",
			shouldNotContain: []string{"\r", "\n", "\t"},
		},
		{
			name:        "removes shell metacharacters",
			input:       "file;$(rm -rf /).jpg",
			shouldNotContain: []string{";", "$"}, // ( and ) are allowed characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SanitizeFilename(tt.input)
			for _, forbidden := range tt.shouldNotContain {
				assert.NotContains(t, result, forbidden)
			}
		})
	}
}

// mockClamAVScanner is a mock implementation of clamav.Scanner for testing.
type mockClamAVScanner struct {
	scanResult *clamav.ScanResult
	scanError  error
}

func (m *mockClamAVScanner) Scan(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
	return m.scanResult, m.scanError
}

func (m *mockClamAVScanner) ScanReader(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error) {
	return m.scanResult, m.scanError
}

func (m *mockClamAVScanner) Ping(ctx context.Context) error {
	if m.scanError != nil {
		return m.scanError
	}
	return nil
}

func (m *mockClamAVScanner) Version(ctx context.Context) (string, error) {
	if m.scanError != nil {
		return "", m.scanError
	}
	return "ClamAV 1.0.0 Mock", nil
}

func (m *mockClamAVScanner) Stats(ctx context.Context) (string, error) {
	if m.scanError != nil {
		return "", m.scanError
	}
	return "Mock Stats: POOLS: 1 STATE: CLEAN", nil
}

func (m *mockClamAVScanner) Close() error {
	return nil
}
