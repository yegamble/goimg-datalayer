package storage

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// validKeyPattern ensures keys only contain safe characters.
	// Pattern: images/{uuid}/{uuid}/{variant}.{ext}
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
// - Extension for MIME type sniffing
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
// - Invalid characters
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
func (g *KeyGenerator) ParseKey(key string) (ownerID, imageID uuid.UUID, variant, ext string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 4 || parts[0] != "images" {
		err = fmt.Errorf("%w: expected images/{owner}/{image}/{variant}.ext", ErrInvalidKey)
		return
	}

	ownerID, err = uuid.Parse(parts[1])
	if err != nil {
		err = fmt.Errorf("%w: invalid owner ID: %v", ErrInvalidKey, err)
		return
	}

	imageID, err = uuid.Parse(parts[2])
	if err != nil {
		err = fmt.Errorf("%w: invalid image ID: %v", ErrInvalidKey, err)
		return
	}

	// Extract variant and extension from filename (e.g., "thumbnail.jpg")
	filename := parts[3]
	ext = path.Ext(filename)
	variant = strings.TrimSuffix(filename, ext)
	ext = strings.TrimPrefix(ext, ".")

	if variant == "" || ext == "" {
		err = fmt.Errorf("%w: missing variant or extension", ErrInvalidKey)
		return
	}

	return
}

// normalizeFormat converts MIME types and format names to file extensions.
func normalizeFormat(format string) string {
	switch strings.ToLower(format) {
	case "image/jpeg", "jpeg", "jpg":
		return "jpg"
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
		return "jpg" // Default fallback
	}
}

// SanitizeFilename removes dangerous characters from a filename.
// This is used for the original_filename field, not for storage keys.
func SanitizeFilename(filename string) string {
	// Remove path components
	filename = path.Base(filename)

	// Replace dangerous characters
	var builder strings.Builder
	for _, r := range filename {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			builder.WriteRune(r)
		case r == ' ':
			builder.WriteRune('_')
		default:
			// Skip other characters
		}
	}

	result := builder.String()

	// Ensure the filename has an extension
	if !strings.Contains(result, ".") {
		result += ".jpg"
	}

	// Prevent empty filenames
	if result == "" || result == "." {
		result = "unnamed.jpg"
	}

	// Limit length
	if len(result) > 200 {
		ext := path.Ext(result)
		result = result[:200-len(ext)] + ext
	}

	return result
}
