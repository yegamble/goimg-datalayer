package gallery

import (
	"fmt"
	"strings"
)

const (
	// MaxTitleLength is the maximum allowed length for image titles.
	MaxTitleLength = 255

	// MaxDescriptionLength is the maximum allowed length for descriptions.
	MaxDescriptionLength = 2000

	// MaxFileSize is the maximum allowed file size in bytes (10MB).
	MaxFileSize = 10485760 // 10 * 1024 * 1024

	// MaxImageDimension is the maximum allowed width or height in pixels.
	MaxImageDimension = 8192

	// MaxImagePixels is the maximum allowed total pixels (100 million).
	MaxImagePixels = 100000000
)

// SupportedMimeTypes lists all allowed image MIME types.
var SupportedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// ImageMetadata is a value object containing image file metadata and descriptive information.
// It is immutable after creation and validates all constraints.
type ImageMetadata struct {
	title            string
	description      string
	originalFilename string
	mimeType         string
	width            int
	height           int
	fileSize         int64
	storageKey       string
	storageProvider  string
}

// NewImageMetadata creates a new ImageMetadata with validation.
// All parameters are validated against business rules:
// - title: max 255 chars (can be empty, will use filename)
// - description: max 2000 chars (optional)
// - originalFilename: required
// - mimeType: must be in whitelist
// - width, height: must be > 0 and <= 8192
// - fileSize: must be > 0 and <= 10MB
// - total pixels: must be <= 100 million
// - storageKey: required
// - storageProvider: required
func NewImageMetadata(
	title,
	description,
	originalFilename,
	mimeType string,
	width,
	height int,
	fileSize int64,
	storageKey,
	storageProvider string,
) (ImageMetadata, error) {
	// Validate and normalize text fields.
	title, err := validateTitle(title, originalFilename)
	if err != nil {
		return ImageMetadata{}, err
	}

	description, err = validateDescription(description)
	if err != nil {
		return ImageMetadata{}, err
	}

	originalFilename, err = validateFilename(originalFilename)
	if err != nil {
		return ImageMetadata{}, err
	}

	mimeType, err = validateMimeType(mimeType)
	if err != nil {
		return ImageMetadata{}, err
	}

	// Validate dimensions and file size.
	if err := validateDimensions(width, height); err != nil {
		return ImageMetadata{}, err
	}

	if err := validateFileSize(fileSize); err != nil {
		return ImageMetadata{}, err
	}

	// Validate storage fields.
	storageKey, err = validateStorageKey(storageKey)
	if err != nil {
		return ImageMetadata{}, err
	}

	storageProvider, err = validateStorageProvider(storageProvider)
	if err != nil {
		return ImageMetadata{}, err
	}

	return ImageMetadata{
		title:            title,
		description:      description,
		originalFilename: originalFilename,
		mimeType:         mimeType,
		width:            width,
		height:           height,
		fileSize:         fileSize,
		storageKey:       storageKey,
		storageProvider:  storageProvider,
	}, nil
}

// validateTitle validates and normalizes the title, using filename as fallback.
func validateTitle(title, originalFilename string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = originalFilename
	}

	if len(title) > MaxTitleLength {
		return "", fmt.Errorf("%w: got %d characters", ErrTitleTooLong, len(title))
	}

	return title, nil
}

// validateDescription validates and normalizes the description.
func validateDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if len(description) > MaxDescriptionLength {
		return "", fmt.Errorf("%w: got %d characters", ErrDescriptionTooLong, len(description))
	}

	return description, nil
}

// validateFilename validates that the filename is not empty.
func validateFilename(filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", fmt.Errorf("%w: original filename is required", ErrInvalidMetadata)
	}

	return filename, nil
}

// validateMimeType validates the MIME type against the whitelist.
func validateMimeType(mimeType string) (string, error) {
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))
	if !SupportedMimeTypes[mimeType] {
		return "", fmt.Errorf("%w: '%s' is not supported", ErrInvalidMimeType, mimeType)
	}

	return mimeType, nil
}

// validateDimensions validates image width and height.
func validateDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf(
			"%w: dimensions must be positive, got %dx%d",
			ErrInvalidDimensions,
			width,
			height,
		)
	}

	if width > MaxImageDimension || height > MaxImageDimension {
		return fmt.Errorf(
			"%w: maximum %dx%d, got %dx%d",
			ErrImageTooLarge,
			MaxImageDimension,
			MaxImageDimension,
			width,
			height,
		)
	}

	totalPixels := int64(width) * int64(height)
	if totalPixels > MaxImagePixels {
		return fmt.Errorf(
			"%w: %d pixels exceeds maximum of %d",
			ErrImageTooManyPixels,
			totalPixels,
			MaxImagePixels,
		)
	}

	return nil
}

// validateFileSize validates the file size.
func validateFileSize(fileSize int64) error {
	if fileSize <= 0 {
		return fmt.Errorf("%w: file size must be positive", ErrInvalidMetadata)
	}

	if fileSize > MaxFileSize {
		return fmt.Errorf(
			"%w: %d bytes exceeds maximum of %d",
			ErrFileTooLarge,
			fileSize,
			MaxFileSize,
		)
	}

	return nil
}

// validateStorageKey validates the storage key.
func validateStorageKey(storageKey string) (string, error) {
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return "", ErrStorageKeyRequired
	}

	return storageKey, nil
}

// validateStorageProvider validates the storage provider.
func validateStorageProvider(storageProvider string) (string, error) {
	storageProvider = strings.TrimSpace(storageProvider)
	if storageProvider == "" {
		return "", ErrProviderRequired
	}

	return storageProvider, nil
}

// Title returns the image title.
func (m ImageMetadata) Title() string {
	return m.title
}

// Description returns the image description.
func (m ImageMetadata) Description() string {
	return m.description
}

// OriginalFilename returns the original filename when uploaded.
func (m ImageMetadata) OriginalFilename() string {
	return m.originalFilename
}

// MimeType returns the MIME type of the image.
func (m ImageMetadata) MimeType() string {
	return m.mimeType
}

// Width returns the image width in pixels.
func (m ImageMetadata) Width() int {
	return m.width
}

// Height returns the image height in pixels.
func (m ImageMetadata) Height() int {
	return m.height
}

// FileSize returns the file size in bytes.
func (m ImageMetadata) FileSize() int64 {
	return m.fileSize
}

// StorageKey returns the key used to retrieve the image from storage.
func (m ImageMetadata) StorageKey() string {
	return m.storageKey
}

// StorageProvider returns the storage provider name (e.g., "s3", "local", "ipfs").
func (m ImageMetadata) StorageProvider() string {
	return m.storageProvider
}

// AspectRatio returns the width/height ratio.
func (m ImageMetadata) AspectRatio() float64 {
	if m.height == 0 {
		return 0
	}
	return float64(m.width) / float64(m.height)
}

// IsLandscape returns true if the image is wider than it is tall.
func (m ImageMetadata) IsLandscape() bool {
	return m.width > m.height
}

// IsPortrait returns true if the image is taller than it is wide.
func (m ImageMetadata) IsPortrait() bool {
	return m.height > m.width
}

// IsSquare returns true if the image has equal width and height.
func (m ImageMetadata) IsSquare() bool {
	return m.width == m.height
}

// WithTitle returns a new ImageMetadata with the title updated.
// This is used for immutable updates.
func (m ImageMetadata) WithTitle(title string) (ImageMetadata, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = m.originalFilename
	}
	if len(title) > MaxTitleLength {
		return ImageMetadata{}, fmt.Errorf("%w: got %d characters", ErrTitleTooLong, len(title))
	}

	return ImageMetadata{
		title:            title,
		description:      m.description,
		originalFilename: m.originalFilename,
		mimeType:         m.mimeType,
		width:            m.width,
		height:           m.height,
		fileSize:         m.fileSize,
		storageKey:       m.storageKey,
		storageProvider:  m.storageProvider,
	}, nil
}

// WithDescription returns a new ImageMetadata with the description updated.
// This is used for immutable updates.
func (m ImageMetadata) WithDescription(description string) (ImageMetadata, error) {
	description = strings.TrimSpace(description)
	if len(description) > MaxDescriptionLength {
		return ImageMetadata{}, fmt.Errorf("%w: got %d characters", ErrDescriptionTooLong, len(description))
	}

	return ImageMetadata{
		title:            m.title,
		description:      description,
		originalFilename: m.originalFilename,
		mimeType:         m.mimeType,
		width:            m.width,
		height:           m.height,
		fileSize:         m.fileSize,
		storageKey:       m.storageKey,
		storageProvider:  m.storageProvider,
	}, nil
}
