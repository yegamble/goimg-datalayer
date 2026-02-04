// Package validator provides image validation for secure uploads.
// It implements a multi-step validation pipeline to prevent malicious uploads.
package validator

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

const (
	// Default validation limits.
	defaultMaxFileSizeMB  = 50
	defaultMaxWidth       = 8192
	defaultMaxHeight      = 8192
	defaultMaxPixels      = 100_000_000
	defaultMinFilenameLen = 12
	bytesPerMB            = 1024 * 1024
)

// ValidationResult contains the result of validating an image.
type ValidationResult struct {
	// Valid is true if the image passed all validation checks.
	Valid bool

	// MIMEType is the detected MIME type (from content, not extension).
	MIMEType string

	// Width is the image width in pixels.
	Width int

	// Height is the image height in pixels.
	Height int

	// FileSize is the size in bytes.
	FileSize int64

	// Errors contains any validation errors encountered.
	Errors []string

	// ScanResult contains the malware scan result (if scanning was performed).
	ScanResult *clamav.ScanResult
}

// Validator provides image validation capabilities.
type Validator struct {
	clamavClient clamav.Scanner
	config       Config
}

// Config configures the image validator.
type Config struct {
	// MaxFileSize is the maximum allowed file size in bytes.
	MaxFileSize int64

	// MaxWidth is the maximum allowed image width in pixels.
	MaxWidth int

	// MaxHeight is the maximum allowed image height in pixels.
	MaxHeight int

	// MaxPixels is the maximum total pixel count (width * height).
	MaxPixels int64

	// AllowedMIMETypes is the list of allowed MIME types.
	AllowedMIMETypes []string

	// EnableMalwareScan enables ClamAV malware scanning.
	EnableMalwareScan bool
}

// DefaultConfig returns sensible defaults for image validation.
func DefaultConfig() Config {
	return Config{
		MaxFileSize: defaultMaxFileSizeMB * bytesPerMB,
		MaxWidth:    defaultMaxWidth,
		MaxHeight:   defaultMaxHeight,
		MaxPixels:   defaultMaxPixels,
		AllowedMIMETypes: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"video/mp4",
			"video/webm",
			"video/quicktime",
		},
		EnableMalwareScan: true,
	}
}

// New creates a new image validator.
func New(cfg Config, clamavClient clamav.Scanner) *Validator {
	if len(cfg.AllowedMIMETypes) == 0 {
		cfg.AllowedMIMETypes = DefaultConfig().AllowedMIMETypes
	}
	return &Validator{
		clamavClient: clamavClient,
		config:       cfg,
	}
}

// Validate performs the full validation pipeline on uploaded image data.
// The 7-step pipeline:
// 1. Size check (max 50MB)
// 2. MIME sniffing (by content, not extension)
// 3. Magic byte validation
// 4. Dimension check (max 8192x8192)
// 5. Pixel count check (max 100M pixels)
// 6. ClamAV malware scan
// 7. Filename sanitization.
func (v *Validator) Validate(ctx context.Context, data []byte, _ string) (*ValidationResult, error) {
	result := &ValidationResult{
		FileSize: int64(len(data)),
		Errors:   make([]string, 0),
	}

	// Step 1: Size check
	if err := v.validateSize(result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Step 2: MIME sniffing by content
	mimeType := http.DetectContentType(data)
	result.MIMEType = mimeType
	if err := v.validateMIMEType(mimeType); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Step 3: Magic byte validation
	if err := v.validateMagicBytes(data); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Step 4 & 5: Dimension and pixel count checks
	// Note: Actual dimension extraction requires image decoding (bimg).
	// This is a basic check; real dimensions come from the image processor.
	// For now, we'll set placeholders and validate in the processor.

	// Step 6: ClamAV malware scan
	if v.config.EnableMalwareScan && v.clamavClient != nil {
		scanResult, err := v.clamavClient.Scan(ctx, data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("malware scan failed: %v", err))
			return result, fmt.Errorf("malware scan failed: %w", err)
		}
		result.ScanResult = scanResult
		if scanResult.Infected {
			result.Errors = append(result.Errors, fmt.Sprintf("malware detected: %s", scanResult.Virus))
			return result, gallery.ErrMalwareDetected
		}
	}

	// Step 7: Filename sanitization (done separately, not blocking)
	// The sanitized filename is returned by SanitizeFilename()

	result.Valid = len(result.Errors) == 0
	return result, nil
}

// validateSize checks if the file size is within limits.
func (v *Validator) validateSize(result *ValidationResult) error {
	if result.FileSize > v.config.MaxFileSize {
		return fmt.Errorf("%w: %d bytes exceeds %d byte limit",
			gallery.ErrFileTooLarge, result.FileSize, v.config.MaxFileSize)
	}
	return nil
}

// validateMIMEType checks if the MIME type is allowed.
func (v *Validator) validateMIMEType(mimeType string) error {
	// Normalize MIME type (remove parameters like charset)
	mimeType = strings.Split(mimeType, ";")[0]
	mimeType = strings.TrimSpace(mimeType)

	for _, allowed := range v.config.AllowedMIMETypes {
		if mimeType == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", gallery.ErrInvalidMimeType, mimeType)
}

// validateMagicBytes verifies the file starts with valid image magic bytes.
func (v *Validator) validateMagicBytes(data []byte) error {
	if len(data) < defaultMinFilenameLen {
		return fmt.Errorf("%w: file too small", gallery.ErrInvalidMimeType)
	}

	// Check magic bytes for each supported format
	magicBytes := map[string][]byte{
		"jpeg": {0xFF, 0xD8, 0xFF},
		"png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		"gif":  {0x47, 0x49, 0x46, 0x38},
		"webp": {0x52, 0x49, 0x46, 0x46}, // RIFF header
		// Video formats
		"mp4":  {0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, // ftyp
		"webm": {0x1A, 0x45, 0xDF, 0xA3},                         // EBML ID
		// mov often starts with ftyp at offset 4, checking common signature
		// Typically 00 00 00 14 66 74 79 70 (ftyp) or similar box size
	}

	for format, magic := range magicBytes {
		if format == "mp4" {
			// MP4 usually starts with size (4 bytes) then 'ftyp' (4 bytes)
			// The size is usually 00 00 00 18 or 00 00 00 20
			// We check for 'ftyp' signature at offset 4
			if len(data) >= 8 && string(data[4:8]) == "ftyp" {
				return nil
			}
		}

		if bytes.HasPrefix(data, magic) {
			// For WebP, also check for WEBP signature at offset 8
			if format == "webp" && len(data) >= defaultMinFilenameLen {
				if !bytes.Equal(data[8:defaultMinFilenameLen], []byte("WEBP")) {
					continue
				}
			}
			return nil
		}
	}

	// Additional check for QuickTime/MOV if not caught by generic MP4/ftyp check
	// QuickTime files also use ftyp atoms, often 'qt  ' major brand
	if len(data) >= 8 && string(data[4:8]) == "ftyp" {
		return nil
	}
	if len(data) >= 8 && string(data[4:8]) == "moov" {
		return nil
	}
	// Check for 'skip' atom (free space) which some MOVs start with
	if len(data) >= 8 && string(data[4:8]) == "free" {
		return nil
	}
	// Check for 'wide' atom
	if len(data) >= 8 && string(data[4:8]) == "wide" {
		return nil
	}

	// Check for 'mdat' atom early in file (less common as start)
	if len(data) >= 8 && string(data[4:8]) == "mdat" {
		return nil
	}

	// Check for Matroska/WebM EBML Header specifically if prefix check failed
	// The prefix 1A 45 DF A3 is the EBML header.
	// If the file starts with this, it is likely WebM or MKV.
	if len(data) >= 4 {
		id := binary.BigEndian.Uint32(data[0:4])
		if id == 0x1A45DFA3 {
			return nil
		}
	}

	return fmt.Errorf("%w: invalid magic bytes", gallery.ErrInvalidMimeType)
}

// ValidateDimensions checks if image dimensions are within limits.
// This should be called after image decoding.
func (v *Validator) ValidateDimensions(width, height int) error {
	// For video/unknown dimensions (0x0), we skip validation here.
	// The processor will validate if it can process it.
	if width == 0 && height == 0 {
		return nil
	}

	if width < 0 || height < 0 {
		return gallery.ErrInvalidDimensions
	}

	if width > v.config.MaxWidth || height > v.config.MaxHeight {
		return fmt.Errorf("%w: %dx%d exceeds %dx%d limit",
			gallery.ErrImageTooLarge, width, height, v.config.MaxWidth, v.config.MaxHeight)
	}

	pixels := int64(width) * int64(height)
	if pixels > v.config.MaxPixels {
		return fmt.Errorf("%w: %d pixels exceeds %d limit",
			gallery.ErrImageTooManyPixels, pixels, v.config.MaxPixels)
	}

	return nil
}

// SanitizeFilename is re-exported from the storage package for convenience.
// It removes dangerous characters from a filename using a conservative
// whitelist approach (only alphanumeric, dots, hyphens, underscores).
// See storage.SanitizeFilename for the canonical implementation.
var SanitizeFilename = storage.SanitizeFilename
