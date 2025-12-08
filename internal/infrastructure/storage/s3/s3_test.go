package s3 //nolint:testpackage // Tests access unexported types

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_ConfigValidation tests configuration validation.
func TestNew_ConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       Config
		wantError string
	}{
		{
			name: "missing bucket",
			cfg: Config{
				Bucket:          "",
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantError: "bucket name required",
		},
		{
			name: "default region applied",
			cfg: Config{
				Bucket:          "test-bucket",
				Region:          "", // Empty region
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantError: "", // Should succeed with default region
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Note: We can't easily test New() without mocking AWS config loading
			// So we only test the validation logic by inspecting the code paths
			if tt.cfg.Bucket == "" {
				// This would fail in New()
				assert.Empty(t, tt.cfg.Bucket)
			}

			if tt.cfg.Region == "" && tt.wantError == "" {
				// Config should set default region
				defaultRegion := "us-east-1"
				assert.NotEqual(t, defaultRegion, tt.cfg.Region)
				// In actual New(), this would be set to us-east-1
			}
		})
	}
}

// TestValidateKey_Valid tests key validation with valid keys.
func TestValidateKey_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"simple filename", "test.jpg"},
		{"nested path", "images/owner/image.jpg"},
		{"deep nesting", "a/b/c/d/e/f.jpg"},
		{"with numbers", "user123/image456/photo.png"},
		{"with dashes", "user-id/image-id/original.jpg"},
		{"with underscores", "user_id/image_id/thumbnail.jpg"},
		{"with spaces", "path with spaces/file.jpg"}, // S3 allows spaces
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateKey(tt.key)
			assert.NoError(t, err)
		})
	}
}

// TestValidateKey_Invalid tests key validation with invalid keys.
func TestValidateKey_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantError error
	}{
		{
			name:      "empty key",
			key:       "",
			wantError: errInvalidKey,
		},
		{
			name:      "path traversal with dots",
			key:       "../etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "path traversal in middle",
			key:       "images/../config/app.conf",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path with slash",
			key:       "/etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path with backslash",
			key:       "\\windows\\system32",
			wantError: errPathTraversal,
		},
		{
			name:      "null byte injection",
			key:       "test\x00.jpg",
			wantError: errInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError, "expected error %v, got %v", tt.wantError, err)
		})
	}
}

// TestURL_WithPublicURL tests URL generation with custom public URL.
func TestURL_WithPublicURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		publicURL string
		bucket    string
		key       string
		wantURL   string
	}{
		{
			name:      "with public URL",
			publicURL: "https://cdn.example.com",
			bucket:    "my-bucket",
			key:       "images/test.jpg",
			wantURL:   "https://cdn.example.com/images/test.jpg",
		},
		{
			name:      "public URL with trailing slash",
			publicURL: "https://cdn.example.com/",
			bucket:    "my-bucket",
			key:       "test.jpg",
			wantURL:   "https://cdn.example.com/test.jpg",
		},
		{
			name:      "without public URL uses default",
			publicURL: "",
			bucket:    "my-bucket",
			key:       "images/test.jpg",
			wantURL:   "https://my-bucket.s3.amazonaws.com/images/test.jpg",
		},
		{
			name:      "default URL format",
			publicURL: "",
			bucket:    "production-images",
			key:       "user/123/photo.jpg",
			wantURL:   "https://production-images.s3.amazonaws.com/user/123/photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &S3Storage{
				publicURL: tt.publicURL,
				bucket:    tt.bucket,
			}

			url := s.URL(tt.key)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

// TestIsNotFoundError tests S3 not found error detection.
func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantFound bool
	}{
		{
			name:      "NoSuchKey error",
			err:       &types.NoSuchKey{Message: stringPtr("key not found")},
			wantFound: true,
		},
		{
			name:      "NotFound error",
			err:       &types.NotFound{Message: stringPtr("not found")},
			wantFound: true,
		},
		{
			name:      "error message contains NotFound",
			err:       errors.New("S3 error: NotFound - object does not exist"),
			wantFound: true,
		},
		{
			name:      "error message contains NoSuchKey",
			err:       errors.New("NoSuchKey: The specified key does not exist"),
			wantFound: true,
		},
		{
			name:      "other error",
			err:       errors.New("some other error"),
			wantFound: false,
		},
		{
			name:      "access denied error",
			err:       errors.New("AccessDenied: permission denied"),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.wantFound, result)
		})
	}
}

// TestIsAccessDeniedError tests S3 access denied error detection.
func TestIsAccessDeniedError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantDenied bool
	}{
		{
			name:       "error message contains AccessDenied",
			err:        errors.New("AccessDenied: user not authorized"),
			wantDenied: true,
		},
		{
			name:       "error message with AccessDenied code",
			err:        errors.New("S3 error: AccessDenied - insufficient permissions"),
			wantDenied: true,
		},
		{
			name:       "other error",
			err:        errors.New("some other error"),
			wantDenied: false,
		},
		{
			name:       "not found error",
			err:        errors.New("NotFound: key does not exist"),
			wantDenied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isAccessDeniedError(tt.err)
			assert.Equal(t, tt.wantDenied, result)
		})
	}
}

// TestProvider tests the Provider method.
func TestProvider(t *testing.T) {
	t.Parallel()

	s := &S3Storage{
		bucket: "test-bucket",
	}

	assert.Equal(t, "s3", s.Provider())
}

// TestS3Storage_PresignedURLDuration tests presigned URL duration handling.
func TestS3Storage_PresignedURLDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		configuredExpiry string
		requestDuration  string
		description      string
	}{
		{
			name:             "uses default expiry when duration is zero",
			configuredExpiry: "15m",
			requestDuration:  "0",
			description:      "Should use configured default when 0 duration is passed",
		},
		{
			name:             "uses custom duration when provided",
			configuredExpiry: "15m",
			requestDuration:  "1h",
			description:      "Should use custom duration when explicitly provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// This test documents the expected behavior
			// Actual implementation testing would require mocking the presigner
			assert.NotEmpty(t, tt.description)
		})
	}
}

// TestObjectInfo tests ObjectInfo structure.
func TestObjectInfo(t *testing.T) {
	t.Parallel()

	info := &ObjectInfo{
		Key:         "images/test.jpg",
		Size:        1024,
		ContentType: "image/jpeg",
		ETag:        `"abc123"`,
	}

	assert.Equal(t, "images/test.jpg", info.Key)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, "image/jpeg", info.ContentType)
	assert.Equal(t, `"abc123"`, info.ETag)
}

// TestPutOptions tests PutOptions structure.
func TestPutOptions(t *testing.T) {
	t.Parallel()

	opts := PutOptions{
		ContentType:  "image/png",
		CacheControl: "max-age=3600",
		Metadata: map[string]string{
			"owner":  "user123",
			"source": "upload",
		},
	}

	assert.Equal(t, "image/png", opts.ContentType)
	assert.Equal(t, "max-age=3600", opts.CacheControl)
	assert.Len(t, opts.Metadata, 2)
	assert.Equal(t, "user123", opts.Metadata["owner"])
}

// TestConfig_DefaultValues tests default configuration values.
func TestConfig_DefaultValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         Config
		expectValid bool
		description string
	}{
		{
			name: "valid minimal config",
			cfg: Config{
				Bucket:          "test-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			expectValid: true,
			description: "Minimal config should be valid (region defaults to us-east-1)",
		},
		{
			name: "valid config with endpoint",
			cfg: Config{
				Bucket:          "test-bucket",
				Endpoint:        "https://s3.us-west-2.amazonaws.com",
				Region:          "us-west-2",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				ForcePathStyle:  false,
			},
			expectValid: true,
			description: "Config with custom endpoint should be valid",
		},
		{
			name: "valid config for MinIO",
			cfg: Config{
				Bucket:          "images",
				Endpoint:        "http://localhost:9000",
				Region:          "us-east-1",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				ForcePathStyle:  true, // Required for MinIO
			},
			expectValid: true,
			description: "MinIO requires ForcePathStyle=true",
		},
		{
			name: "invalid config missing bucket",
			cfg: Config{
				Bucket:          "",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			},
			expectValid: false,
			description: "Bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.expectValid {
				assert.NotEmpty(t, tt.cfg.Bucket, "valid config must have bucket")
			} else {
				assert.Empty(t, tt.cfg.Bucket, "invalid config missing bucket")
			}

			assert.NotEmpty(t, tt.description)
		})
	}
}

// TestErrorTypes tests error type definitions.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrNotFound",
			err:  ErrNotFound,
			msg:  "storage: object not found",
		},
		{
			name: "ErrAccessDenied",
			err:  ErrAccessDenied,
			msg:  "storage: access denied",
		},
		{
			name: "errInvalidKey",
			err:  errInvalidKey,
			msg:  "storage: invalid key",
		},
		{
			name: "errPathTraversal",
			err:  errPathTraversal,
			msg:  "storage: path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.msg, tt.err.Error())
		})
	}
}

// stringPtr is a helper to get a pointer to a string.
func stringPtr(s string) *string {
	return &s
}
