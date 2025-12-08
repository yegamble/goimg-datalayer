package storage

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const (
	// Maximum filename length
	MaxFilenameLength = 200

	// Default fallback filename
	DefaultFilename = "unnamed.jpg"

	// Format constants.
	formatJPG = "jpg"
)

var (
	// validKeyPattern ensures keys only contain safe characters.
	// Pattern: images/{uuid}/{uuid}/{variant}.{ext}.
	validKeyPattern = regexp.MustCompile(`^images/[a-f0-9-]{36}/[a-f0-9-]{36}/[a-z]+\.(jpg|jpeg|png|gif|webp)$`)

	// validExtensions lists allowed file extensions.
	validExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
)

// KeyGenerator generates safe storage keys for images.
type KeyGenerator struct{}

// NewKeyGenerator creates a new storage key generator.
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateKey creates a storage key for an image variant.
// Format: images/{owner_id}/{image_id}/{variant}.{ext}
// This pattern:
// - Organizes by user (enables bulk operations, quota management)
// - Unique image ID prevents collisions
// - Variant type distinguishes sizes
// - Extension for MIME type sniffing.
func (g *KeyGenerator) GenerateKey(ownerID, imageID uuid.UUID, variant, format string) string {
	format = normalizeFormat(format)
	return fmt.Sprintf(
		"images/%s/%s/%s.%s",
		ownerID.String(),
		imageID.String(),
		variant,
		format,
	)
}

// ValidateKey checks if a storage key is safe (prevents path traversal).
// Returns error if key contains:
// - Parent directory references (..)
// - Absolute paths (/)
// - Null bytes
// - Invalid characters.
func (g *KeyGenerator) ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty key", ErrInvalidKey)
	}

	// Prevent path traversal
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: contains '..' path traversal", ErrPathTraversal)
	}

	// Prevent absolute paths
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return fmt.Errorf("%w: cannot be absolute path", ErrPathTraversal)
	}

	// Prevent null bytes
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("%w: contains null byte", ErrInvalidKey)
	}

	// Clean the path and compare
	cleaned := path.Clean(key)
	if cleaned != key {
		return fmt.Errorf("%w: path not canonical", ErrInvalidKey)
	}

	// Validate pattern for image keys
	if strings.HasPrefix(key, "images/") && !validKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: invalid image key format", ErrInvalidKey)
	}

	return nil
}

// ParseKey extracts components from a storage key.
// Returns ownerID, imageID, variant, extension, and any error.
func (g *KeyGenerator) ParseKey(key string) (uuid.UUID, uuid.UUID, string, string, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 4 || parts[0] != "images" {
		return uuid.Nil, uuid.Nil, "", "", fmt.Errorf("%w: expected images/{owner}/{image}/{variant}.ext", ErrInvalidKey)
	}

	ownerID, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", "", fmt.Errorf("%w: invalid owner ID: %w", ErrInvalidKey, err)
	}

	imageID, err := uuid.Parse(parts[2])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", "", fmt.Errorf("%w: invalid image ID: %w", ErrInvalidKey, err)
	}

	// Extract variant and extension from filename (e.g., "thumbnail.jpg")
	filename := parts[3]
	ext := path.Ext(filename)
	variant := strings.TrimSuffix(filename, ext)
	ext = strings.TrimPrefix(ext, ".")

	if variant == "" || ext == "" {
		return uuid.Nil, uuid.Nil, "", "", fmt.Errorf("%w: missing variant or extension", ErrInvalidKey)
	}

	return ownerID, imageID, variant, ext, nil
}

// normalizeFormat converts MIME types and format names to file extensions.
func normalizeFormat(format string) string {
	switch strings.ToLower(format) {
	case "image/jpeg", "jpeg", formatJPG:
		return formatJPG
	case "image/png", "png":
		return "png"
	case "image/gif", "gif":
		return "gif"
	case "image/webp", "webp":
		return "webp"
	default:
		// Default to original extension if provided
		format = strings.ToLower(format)
		format = strings.TrimPrefix(format, ".")
		if validExtensions["."+format] {
			return format
		}
		return formatJPG // Default fallback
	}
}

// SanitizeFilename removes dangerous characters from a filename.
// This is used for the original_filename field, not for storage keys.
func SanitizeFilename(filename string) string {
	// Remove path components
	filename = path.Base(filename)

	// Replace dangerous characters with safe ones
	sanitized := sanitizeCharacters(filename)

	// Ensure valid filename format
	return ensureValidFilename(sanitized)
}

// sanitizeCharacters replaces or removes unsafe characters from a filename.
func sanitizeCharacters(filename string) string {
	var builder strings.Builder
	for _, r := range filename {
		if isSafeCharacter(r) {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteRune('_')
		}
		// Other characters are skipped
	}
	return builder.String()
}

// isSafeCharacter returns true if the character is safe for filenames.
func isSafeCharacter(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '.' || r == '-' || r == '_'
}

// ensureValidFilename ensures the filename has an extension and valid length
func ensureValidFilename(filename string) string {
	// Ensure the filename has an extension
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}

	// Prevent empty filenames
	if filename == "" || filename == "." {
		return DefaultFilename
	}

	// Limit length
	if len(filename) > MaxFilenameLength {
		return truncateFilename(filename)
	}

	return filename
}

// truncateFilename shortens a filename while preserving its extension
func truncateFilename(filename string) string {
	ext := path.Ext(filename)
	maxNameLength := MaxFilenameLength - len(ext)
	return filename[:maxNameLength] + ext
}
