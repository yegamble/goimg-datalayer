package security_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// TestStorage_PreventsPathTraversal verifies keys with ".." are rejected.
// Security Control: Path traversal prevention protects against directory escape attacks.
//
//nolint:funlen // Security test with comprehensive attack scenarios
func TestStorage_PreventsPathTraversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantError error
	}{
		{
			name:      "valid image key",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantError: nil,
		},
		{
			name:      "path traversal - parent directory",
			key:       "images/../etc/passwd",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "path traversal - multiple levels",
			key:       "images/../../etc/shadow",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "path traversal - in middle",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/../../../etc/passwd",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "path traversal - encoded",
			key:       "images/..%2F..%2Fetc%2Fpasswd",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "absolute path - unix",
			key:       "/etc/passwd",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "absolute path - windows",
			key:       "\\Windows\\System32\\config\\SAM",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "null byte injection",
			key:       "images/file\x00.jpg.exe",
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "empty key",
			key:       "",
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "non-canonical path (extra slashes)",
			key:       "images//550e8400-e29b-41d4-a716-446655440000//thumbnail.jpg",
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "non-canonical path (trailing slash)",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/thumbnail.jpg/",
			wantError: storage.ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			generator := storage.NewKeyGenerator()

			// Act
			err := generator.ValidateKey(tt.key)

			// Assert
			if tt.wantError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestStorage_GeneratesNonGuessableKeys verifies storage keys use UUIDs.
// Security Control: Non-guessable keys prevent enumeration attacks.
func TestStorage_GeneratesNonGuessableKeys(t *testing.T) {
	t.Parallel()

	// Arrange
	generator := storage.NewKeyGenerator()
	ownerID := uuid.New()
	imageID := uuid.New()

	// Act
	key1 := generator.GenerateKey(ownerID, imageID, "thumbnail", "jpg")
	key2 := generator.GenerateKey(ownerID, imageID, "original", "jpg")
	key3 := generator.GenerateKey(uuid.New(), uuid.New(), "thumbnail", "jpg")

	// Assert - Keys should contain UUIDs (not sequential integers)
	assert.Contains(t, key1, ownerID.String())
	assert.Contains(t, key1, imageID.String())

	// Same owner and image, different variant
	assert.Contains(t, key2, ownerID.String())
	assert.Contains(t, key2, imageID.String())

	// Different keys should not be guessable from each other
	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key1, key3)

	// Verify format
	assert.True(t, strings.HasPrefix(key1, "images/"))
	assert.True(t, strings.HasSuffix(key1, "/thumbnail.jpg"))

	// Verify UUIDs are valid
	parts := strings.Split(key1, "/")
	require.Len(t, parts, 4)

	parsedOwner, err := uuid.Parse(parts[1])
	require.NoError(t, err)
	assert.Equal(t, ownerID, parsedOwner)

	parsedImage, err := uuid.Parse(parts[2])
	require.NoError(t, err)
	assert.Equal(t, imageID, parsedImage)
}

// TestStorage_ValidatesKeyFormat verifies invalid key formats are rejected.
// Security Control: Strict key format validation prevents injection attacks.
//
//nolint:funlen // Security test with comprehensive attack scenarios
func TestStorage_ValidatesKeyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantValid bool
		wantError error
	}{
		{
			name:      "valid key - thumbnail",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantValid: true,
		},
		{
			name:      "valid key - original",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/original.png",
			wantValid: true,
		},
		{
			name:      "valid key - medium variant",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/medium.webp",
			wantValid: true,
		},
		{
			name:      "invalid - not starting with images/",
			key:       "photos/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantValid: true, // Pattern validation only applies to images/ prefix
		},
		{
			name:      "invalid - malformed UUID (owner)",
			key:       "images/not-a-uuid/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - malformed UUID (image)",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/not-a-uuid/thumbnail.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - unsupported file extension",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.exe",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - missing variant",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - too few path components",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/thumbnail.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - too many path components",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/extra/thumbnail.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
		{
			name:      "invalid - uppercase in variant (should be lowercase)",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/THUMBNAIL.jpg",
			wantValid: false,
			wantError: storage.ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			generator := storage.NewKeyGenerator()

			// Act
			err := generator.ValidateKey(tt.key)

			// Assert
			if tt.wantValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tt.wantError != nil {
					assert.ErrorIs(t, err, tt.wantError)
				}
			}
		})
	}
}

// TestStorage_ParseKey verifies key parsing extracts correct components.
// Security Control: Validates key components before use in storage operations.
//
//nolint:funlen // Security test with comprehensive attack scenarios
func TestStorage_ParseKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		wantOwnerID string
		wantImageID string
		wantVariant string
		wantExt     string
		wantError   bool
	}{
		{
			name:        "valid thumbnail key",
			key:         "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantOwnerID: "550e8400-e29b-41d4-a716-446655440000",
			wantImageID: "7c9e6679-7425-40de-944b-e07fc1f90ae7",
			wantVariant: "thumbnail",
			wantExt:     "jpg",
			wantError:   false,
		},
		{
			name:        "valid original key",
			key:         "images/a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d/f1e2d3c4-b5a6-4c5d-9e8f-7a6b5c4d3e2f/original.png",
			wantOwnerID: "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
			wantImageID: "f1e2d3c4-b5a6-4c5d-9e8f-7a6b5c4d3e2f",
			wantVariant: "original",
			wantExt:     "png",
			wantError:   false,
		},
		{
			name:      "invalid - not starting with images/",
			key:       "photos/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantError: true,
		},
		{
			name:      "invalid - too few components",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/thumbnail.jpg",
			wantError: true,
		},
		{
			name:      "invalid - malformed owner UUID",
			key:       "images/invalid-uuid/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail.jpg",
			wantError: true,
		},
		{
			name:      "invalid - malformed image UUID",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/invalid-uuid/thumbnail.jpg",
			wantError: true,
		},
		{
			name:      "invalid - no extension",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/thumbnail",
			wantError: true,
		},
		{
			name:      "invalid - no variant",
			key:       "images/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/.jpg",
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			generator := storage.NewKeyGenerator()

			// Act
			ownerID, imageID, variant, ext, err := generator.ParseKey(tt.key)

			// Assert
			if tt.wantError {
				require.Error(t, err)
				assert.ErrorIs(t, err, storage.ErrInvalidKey)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOwnerID, ownerID.String())
				assert.Equal(t, tt.wantImageID, imageID.String())
				assert.Equal(t, tt.wantVariant, variant)
				assert.Equal(t, tt.wantExt, ext)
			}
		})
	}
}

// TestStorage_SanitizeFilename verifies dangerous characters are removed from filenames.
// Security Control: Prevents stored filenames from containing executable or special characters.
//
//nolint:funlen // Security test with comprehensive attack scenarios
func TestStorage_SanitizeFilename(t *testing.T) {
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
			name:     "path components removed",
			input:    "/path/to/photo.jpg",
			expected: "photo.jpg",
		},
		{
			name:     "spaces replaced",
			input:    "my vacation photo.jpg",
			expected: "my_vacation_photo.jpg",
		},
		{
			name:     "special characters removed",
			input:    "photo<>:\"/\\|?*.jpg",
			expected: ".jpg", // path.Base extracts after "/", special chars removed
		},
		{
			name:     "unicode removed",
			input:    "фото日本語.jpg",
			expected: ".jpg",
		},
		{
			name:     "very long filename truncated",
			input:    strings.Repeat("a", 300) + ".jpg",
			expected: strings.Repeat("a", 196) + ".jpg",
		},
		{
			name:     "only extension preserved",
			input:    ".jpg",
			expected: ".jpg", // Dots are allowed
		},
		{
			name:     "empty string gets default",
			input:    "",
			expected: "unnamed.jpg",
		},
		{
			name:     "multiple dots preserved",
			input:    "....jpg",
			expected: "....jpg", // Dots are valid characters
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := storage.SanitizeFilename(tt.input)

			// Assert
			assert.Equal(t, tt.expected, result)

			// Verify security properties
			assert.NotContains(t, result, "/")
			assert.NotContains(t, result, "\\")
			// Note: dots are allowed in filenames (e.g., "image.thumb.jpg")
			// path.Base() ensures no path traversal via ".." in directory structure
			assert.LessOrEqual(t, len(result), 200)
		})
	}
}

// TestStorage_KeyGeneration_Uniqueness verifies generated keys are unique.
// Security Control: Ensures no key collisions occur.
func TestStorage_KeyGeneration_Uniqueness(t *testing.T) {
	t.Parallel()

	// Arrange
	generator := storage.NewKeyGenerator()
	ownerID := uuid.New()
	keys := make(map[string]bool)
	numKeys := 1000

	// Act - Generate many keys
	for i := 0; i < numKeys; i++ {
		imageID := uuid.New()
		key := generator.GenerateKey(ownerID, imageID, "thumbnail", "jpg")

		// Assert - No duplicates
		assert.False(t, keys[key], "Generated duplicate key: %s", key)
		keys[key] = true

		// Verify key is valid
		err := generator.ValidateKey(key)
		require.NoError(t, err)
	}

	// Assert - All keys are unique
	assert.Len(t, keys, numKeys)
}

// TestStorage_KeyGeneration_FormatNormalization verifies format strings are normalized.
// Security Control: Consistent file extensions prevent MIME type confusion.
func TestStorage_KeyGeneration_FormatNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inputFormat    string
		expectedFormat string
	}{
		{
			name:           "jpg extension",
			inputFormat:    "jpg",
			expectedFormat: "jpg",
		},
		{
			name:           "jpeg normalized to jpg",
			inputFormat:    "jpeg",
			expectedFormat: "jpg",
		},
		{
			name:           "MIME type normalized",
			inputFormat:    "image/jpeg",
			expectedFormat: "jpg",
		},
		{
			name:           "png extension",
			inputFormat:    "png",
			expectedFormat: "png",
		},
		{
			name:           "PNG uppercase normalized",
			inputFormat:    "PNG",
			expectedFormat: "png",
		},
		{
			name:           "webp extension",
			inputFormat:    "webp",
			expectedFormat: "webp",
		},
		{
			name:           "gif extension",
			inputFormat:    "gif",
			expectedFormat: "gif",
		},
		{
			name:           "unsupported defaults to jpg",
			inputFormat:    "bmp",
			expectedFormat: "jpg",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			generator := storage.NewKeyGenerator()
			ownerID := uuid.New()
			imageID := uuid.New()

			// Act
			key := generator.GenerateKey(ownerID, imageID, "thumbnail", tt.inputFormat)

			// Assert
			assert.True(t, strings.HasSuffix(key, "."+tt.expectedFormat))
		})
	}
}
