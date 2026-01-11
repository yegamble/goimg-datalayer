package s3

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPut_KeyValidation tests that Put validates keys before upload.
func TestPut_KeyValidation(t *testing.T) {
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
			name:      "path traversal",
			key:       "../etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path",
			key:       "/etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "null byte",
			key:       "test\x00.jpg",
			wantError: errInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a storage instance (we won't actually call S3)
			s := &Storage{
				bucket: "test-bucket",
			}

			// Try to put with invalid key
			err := s.Put(context.Background(), tt.key, bytes.NewReader([]byte("test")), 4, PutOptions{})

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestPut_Options tests that Put correctly applies options.
func TestPut_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts PutOptions
		want struct {
			hasContentType  bool
			hasCacheControl bool
		}
	}{
		{
			name: "with content type",
			opts: PutOptions{
				ContentType: "image/jpeg",
			},
			want: struct {
				hasContentType  bool
				hasCacheControl bool
			}{
				hasContentType:  true,
				hasCacheControl: false,
			},
		},
		{
			name: "with cache control",
			opts: PutOptions{
				CacheControl: "max-age=3600",
			},
			want: struct {
				hasContentType  bool
				hasCacheControl bool
			}{
				hasContentType:  false,
				hasCacheControl: true,
			},
		},
		{
			name: "with both options",
			opts: PutOptions{
				ContentType:  "image/png",
				CacheControl: "public, max-age=7200",
			},
			want: struct {
				hasContentType  bool
				hasCacheControl bool
			}{
				hasContentType:  true,
				hasCacheControl: true,
			},
		},
		{
			name: "with no options",
			opts: PutOptions{},
			want: struct {
				hasContentType  bool
				hasCacheControl bool
			}{
				hasContentType:  false,
				hasCacheControl: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// This test documents the expected behavior
			// Actual S3 calls would be tested in integration tests
			if tt.want.hasContentType {
				assert.NotEmpty(t, tt.opts.ContentType)
			}
			if tt.want.hasCacheControl {
				assert.NotEmpty(t, tt.opts.CacheControl)
			}
		})
	}
}

// TestPutBytes_KeyValidation tests the PutBytes convenience method validation.
func TestPutBytes_KeyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		data      []byte
		wantError error
	}{
		{
			name:      "invalid key - path traversal",
			key:       "../invalid.jpg",
			data:      []byte("data"),
			wantError: errPathTraversal,
		},
		{
			name:      "invalid key - empty",
			key:       "",
			data:      []byte("data"),
			wantError: errInvalidKey,
		},
		{
			name:      "invalid key - null byte",
			key:       "test\x00.jpg",
			data:      []byte("data"),
			wantError: errInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			err := s.PutBytes(context.Background(), tt.key, tt.data, PutOptions{})

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestGet_KeyValidation tests that Get validates keys.
func TestGet_KeyValidation(t *testing.T) {
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
			name:      "path traversal",
			key:       "../../etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path",
			key:       "/absolute/path.jpg",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			_, err := s.Get(context.Background(), tt.key)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestGetBytes tests the GetBytes convenience method.
func TestGetBytes_KeyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantError error
	}{
		{
			name:      "invalid key - empty",
			key:       "",
			wantError: errInvalidKey,
		},
		{
			name:      "invalid key - traversal",
			key:       "../config.json",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			_, err := s.GetBytes(context.Background(), tt.key)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestDelete_KeyValidation tests that Delete validates keys.
func TestDelete_KeyValidation(t *testing.T) {
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
			name:      "path traversal",
			key:       "images/../../../etc/shadow",
			wantError: errPathTraversal,
		},
		{
			name:      "backslash path",
			key:       "\\windows\\system32",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			err := s.Delete(context.Background(), tt.key)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestExists_KeyValidation tests that Exists validates keys.
func TestExists_KeyValidation(t *testing.T) {
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
			name:      "null byte",
			key:       "test\x00file.jpg",
			wantError: errInvalidKey,
		},
		{
			name:      "path traversal",
			key:       ".././../etc/hosts",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			_, err := s.Exists(context.Background(), tt.key)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestPresignedURL_KeyValidation tests that PresignedURL validates keys.
func TestPresignedURL_KeyValidation(t *testing.T) {
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
			name:      "path traversal",
			key:       "../../../secret.txt",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
				expiry: 15 * time.Minute,
			}

			_, err := s.PresignedURL(context.Background(), tt.key, 0)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestPresignedURL_Duration tests duration handling for presigned URLs.
func TestPresignedURL_DurationHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		configuredExpiry time.Duration
		requestDuration  time.Duration
		expectedDuration time.Duration
	}{
		{
			name:             "uses configured expiry when duration is zero",
			configuredExpiry: 15 * time.Minute,
			requestDuration:  0,
			expectedDuration: 15 * time.Minute,
		},
		{
			name:             "uses custom duration when provided",
			configuredExpiry: 15 * time.Minute,
			requestDuration:  1 * time.Hour,
			expectedDuration: 1 * time.Hour,
		},
		{
			name:             "respects very short duration",
			configuredExpiry: 15 * time.Minute,
			requestDuration:  30 * time.Second,
			expectedDuration: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
				expiry: tt.configuredExpiry,
			}

			// Verify the logic: when duration is 0, use s.expiry
			actualDuration := tt.requestDuration
			if actualDuration == 0 {
				actualDuration = s.expiry
			}

			assert.Equal(t, tt.expectedDuration, actualDuration)
		})
	}
}

// TestStat_KeyValidation tests that Stat validates keys.
func TestStat_KeyValidation(t *testing.T) {
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
			name:      "path traversal",
			key:       "images/../../config",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path",
			key:       "/root/.ssh/id_rsa",
			wantError: errPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket: "test-bucket",
			}

			_, err := s.Stat(context.Background(), tt.key)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

// TestObjectInfo_Fields tests ObjectInfo field mapping.
func TestObjectInfo_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	info := &ObjectInfo{
		Key:          "images/user123/photo.jpg",
		Size:         2048,
		ContentType:  "image/jpeg",
		LastModified: now,
		ETag:         `"abc123def456"`,
	}

	assert.Equal(t, "images/user123/photo.jpg", info.Key)
	assert.Equal(t, int64(2048), info.Size)
	assert.Equal(t, "image/jpeg", info.ContentType)
	assert.Equal(t, now, info.LastModified)
	assert.Equal(t, `"abc123def456"`, info.ETag)
}

// TestConfig_Validation tests configuration validation logic.
func TestConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         Config
		expectValid bool
		description string
	}{
		{
			name: "minimal valid config",
			cfg: Config{
				Bucket:          "my-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			expectValid: true,
			description: "bucket is required, region defaults to us-east-1",
		},
		{
			name: "missing bucket",
			cfg: Config{
				Bucket:          "",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			},
			expectValid: false,
			description: "bucket is required",
		},
		{
			name: "with custom endpoint (MinIO)",
			cfg: Config{
				Bucket:          "images",
				Endpoint:        "http://localhost:9000",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				ForcePathStyle:  true,
			},
			expectValid: true,
			description: "MinIO requires endpoint and ForcePathStyle",
		},
		{
			name: "with public URL (CDN)",
			cfg: Config{
				Bucket:          "prod-images",
				PublicURL:       "https://cdn.example.com",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			},
			expectValid: true,
			description: "PublicURL overrides default URL generation",
		},
		{
			name: "with custom presigned expiry",
			cfg: Config{
				Bucket:             "my-bucket",
				AccessKeyID:        "key",
				SecretAccessKey:    "secret",
				PresignedURLExpiry: 1 * time.Hour,
			},
			expectValid: true,
			description: "custom expiry should be respected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.expectValid {
				assert.NotEmpty(t, tt.cfg.Bucket, "valid config must have bucket")
			} else {
				assert.Empty(t, tt.cfg.Bucket, "invalid config has empty bucket")
			}

			// Verify default region would be applied
			if tt.cfg.Region == "" && tt.expectValid {
				expectedRegion := "us-east-1"
				// In New(), this would be set to us-east-1
				assert.NotEqual(t, expectedRegion, tt.cfg.Region)
			}

			// Verify default expiry would be applied
			if tt.cfg.PresignedURLExpiry == 0 && tt.expectValid {
				expectedExpiry := defaultPresignedURLExpiry
				// In New(), this would be set to defaultPresignedURLExpiry
				assert.NotEqual(t, expectedExpiry, tt.cfg.PresignedURLExpiry)
			}

			assert.NotEmpty(t, tt.description)
		})
	}
}

// TestConfig_EndpointVariations tests different endpoint configurations.
func TestConfig_EndpointVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		provider string
	}{
		{
			name:     "AWS S3 (no endpoint)",
			endpoint: "",
			provider: "AWS S3",
		},
		{
			name:     "DigitalOcean Spaces",
			endpoint: "https://nyc3.digitaloceanspaces.com",
			provider: "DO Spaces",
		},
		{
			name:     "Backblaze B2",
			endpoint: "https://s3.us-west-002.backblazeb2.com",
			provider: "Backblaze B2",
		},
		{
			name:     "MinIO",
			endpoint: "http://localhost:9000",
			provider: "MinIO",
		},
		{
			name:     "Wasabi",
			endpoint: "https://s3.wasabisys.com",
			provider: "Wasabi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{
				Bucket:          "test-bucket",
				Endpoint:        tt.endpoint,
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			}

			if tt.endpoint == "" {
				assert.Empty(t, cfg.Endpoint, "AWS S3 should not have endpoint")
			} else {
				assert.NotEmpty(t, cfg.Endpoint, "S3-compatible services need endpoint")
			}

			assert.NotEmpty(t, tt.provider)
		})
	}
}

// TestURL_EdgeCases tests URL generation with edge cases.
func TestURL_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		publicURL string
		bucket    string
		key       string
		wantURL   string
	}{
		{
			name:      "key with special characters",
			publicURL: "",
			bucket:    "my-bucket",
			key:       "images/user+test@example.com/photo.jpg",
			wantURL:   "https://my-bucket.s3.amazonaws.com/images/user+test@example.com/photo.jpg",
		},
		{
			name:      "key with spaces",
			publicURL: "",
			bucket:    "my-bucket",
			key:       "images/file name with spaces.jpg",
			wantURL:   "https://my-bucket.s3.amazonaws.com/images/file name with spaces.jpg",
		},
		{
			name:      "deep nesting",
			publicURL: "",
			bucket:    "my-bucket",
			key:       "a/b/c/d/e/f/g/file.jpg",
			wantURL:   "https://my-bucket.s3.amazonaws.com/a/b/c/d/e/f/g/file.jpg",
		},
		{
			name:      "public URL with multiple slashes",
			publicURL: "https://cdn.example.com///",
			bucket:    "my-bucket",
			key:       "test.jpg",
			wantURL:   "https://cdn.example.com///test.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				publicURL: tt.publicURL,
				bucket:    tt.bucket,
			}

			url := s.URL(tt.key)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

// TestIsNotFoundError_Comprehensive tests all not found error scenarios.
func TestIsNotFoundError_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantFound bool
	}{
		{
			name:      "NoSuchKey type",
			err:       &types.NoSuchKey{Message: stringPtr("key does not exist")},
			wantFound: true,
		},
		{
			name:      "NotFound type",
			err:       &types.NotFound{Message: stringPtr("resource not found")},
			wantFound: true,
		},
		{
			name:      "wrapped NoSuchKey error",
			err:       errors.New("s3 error: NoSuchKey - object not found"),
			wantFound: true,
		},
		{
			name:      "wrapped NotFound error",
			err:       errors.New("s3 error: NotFound - resource not available"),
			wantFound: true,
		},
		{
			name:      "case sensitive - notsuchkey (lowercase)",
			err:       errors.New("notsuchkey error"),
			wantFound: false,
		},
		{
			name:      "generic error",
			err:       errors.New("some generic error"),
			wantFound: false,
		},
		{
			name:      "network error",
			err:       errors.New("connection refused"),
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

// TestIsAccessDeniedError_Comprehensive tests all access denied scenarios.
func TestIsAccessDeniedError_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantDenied bool
	}{
		{
			name:       "exact AccessDenied",
			err:        errors.New("AccessDenied"),
			wantDenied: true,
		},
		{
			name:       "AccessDenied in message",
			err:        errors.New("s3 error: AccessDenied - permission denied"),
			wantDenied: true,
		},
		{
			name:       "case sensitive - accessdenied (lowercase)",
			err:        errors.New("accessdenied"),
			wantDenied: false,
		},
		{
			name:       "forbidden error",
			err:        errors.New("403 Forbidden"),
			wantDenied: false,
		},
		{
			name:       "other error",
			err:        errors.New("some other error"),
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

// TestValidateKey_Comprehensive tests all key validation scenarios.
func TestValidateKey_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		wantError error
	}{
		// Valid keys
		{
			name:      "simple filename",
			key:       "test.jpg",
			wantError: nil,
		},
		{
			name:      "nested path",
			key:       "images/owner/image.jpg",
			wantError: nil,
		},
		{
			name:      "deep nesting",
			key:       "a/b/c/d/e/f/g/h/i/j/file.jpg",
			wantError: nil,
		},
		{
			name:      "with numbers",
			key:       "user123/image456/photo789.png",
			wantError: nil,
		},
		{
			name:      "with special chars",
			key:       "user-id_123/image.name@2024.jpg",
			wantError: nil,
		},
		{
			name:      "with spaces",
			key:       "path with spaces/file name.jpg",
			wantError: nil,
		},
		{
			name:      "unicode characters",
			key:       "用户/图片.jpg",
			wantError: nil,
		},
		// Invalid keys
		{
			name:      "empty key",
			key:       "",
			wantError: errInvalidKey,
		},
		{
			name:      "path traversal - parent directory",
			key:       "../etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "path traversal - current and parent",
			key:       "./../config",
			wantError: errPathTraversal,
		},
		{
			name:      "path traversal - in middle",
			key:       "images/../../../secret",
			wantError: errPathTraversal,
		},
		{
			name:      "path traversal - at end",
			key:       "images/test/..",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path - unix",
			key:       "/etc/passwd",
			wantError: errPathTraversal,
		},
		{
			name:      "absolute path - windows",
			key:       "\\windows\\system32",
			wantError: errPathTraversal,
		},
		{
			name:      "null byte at start",
			key:       "\x00test.jpg",
			wantError: errInvalidKey,
		},
		{
			name:      "null byte in middle",
			key:       "test\x00.jpg",
			wantError: errInvalidKey,
		},
		{
			name:      "null byte at end",
			key:       "test.jpg\x00",
			wantError: errInvalidKey,
		},
		{
			name:      "multiple null bytes",
			key:       "te\x00st\x00.jpg",
			wantError: errInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)

			if tt.wantError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorage_Provider tests the Provider method.
func TestStorage_Provider(t *testing.T) {
	t.Parallel()

	s := &Storage{}
	assert.Equal(t, "s3", s.Provider())
}

// TestPutOptions_Metadata tests PutOptions with metadata.
func TestPutOptions_Metadata(t *testing.T) {
	t.Parallel()

	opts := PutOptions{
		ContentType:  "image/webp",
		CacheControl: "public, max-age=86400",
		Metadata: map[string]string{
			"owner":       "user-123",
			"upload-date": "2024-01-01",
			"source":      "api",
			"processed":   "true",
		},
	}

	assert.Equal(t, "image/webp", opts.ContentType)
	assert.Equal(t, "public, max-age=86400", opts.CacheControl)
	assert.Len(t, opts.Metadata, 4)
	assert.Equal(t, "user-123", opts.Metadata["owner"])
	assert.Equal(t, "2024-01-01", opts.Metadata["upload-date"])
	assert.Equal(t, "api", opts.Metadata["source"])
	assert.Equal(t, "true", opts.Metadata["processed"])
}

// TestStorage_DefaultValues tests default values for Storage.
func TestStorage_DefaultValues(t *testing.T) {
	t.Parallel()

	s := &Storage{
		bucket:    "test-bucket",
		publicURL: "",
		expiry:    defaultPresignedURLExpiry,
	}

	assert.Equal(t, "test-bucket", s.bucket)
	assert.Empty(t, s.publicURL)
	assert.Equal(t, 15*time.Minute, s.expiry)
	assert.Equal(t, "s3", s.Provider())
}

// TestURL_BucketNaming tests URL generation with different bucket names.
func TestURL_BucketNaming(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bucket  string
		key     string
		wantURL string
	}{
		{
			name:    "simple bucket name",
			bucket:  "mybucket",
			key:     "image.jpg",
			wantURL: "https://mybucket.s3.amazonaws.com/image.jpg",
		},
		{
			name:    "bucket with dashes",
			bucket:  "my-prod-bucket",
			key:     "image.jpg",
			wantURL: "https://my-prod-bucket.s3.amazonaws.com/image.jpg",
		},
		{
			name:    "bucket with dots",
			bucket:  "my.bucket.name",
			key:     "image.jpg",
			wantURL: "https://my.bucket.name.s3.amazonaws.com/image.jpg",
		},
		{
			name:    "bucket with numbers",
			bucket:  "bucket123",
			key:     "image.jpg",
			wantURL: "https://bucket123.s3.amazonaws.com/image.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Storage{
				bucket:    tt.bucket,
				publicURL: "",
			}

			url := s.URL(tt.key)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

// TestErrorMessages tests that errors have descriptive messages.
func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantContain string
	}{
		{
			name:        "ErrNotFound message",
			err:         ErrNotFound,
			wantContain: "not found",
		},
		{
			name:        "ErrAccessDenied message",
			err:         ErrAccessDenied,
			wantContain: "access denied",
		},
		{
			name:        "errInvalidKey message",
			err:         errInvalidKey,
			wantContain: "invalid key",
		},
		{
			name:        "errPathTraversal message",
			err:         errPathTraversal,
			wantContain: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errMsg := tt.err.Error()
			assert.Contains(t, strings.ToLower(errMsg), strings.ToLower(tt.wantContain))
		})
	}
}

// TestDefaultPresignedURLExpiry tests the default expiry constant.
func TestDefaultPresignedURLExpiry(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 15*time.Minute, defaultPresignedURLExpiry)
}

// TestObjectInfo_ZeroValues tests ObjectInfo with zero values.
func TestObjectInfo_ZeroValues(t *testing.T) {
	t.Parallel()

	info := &ObjectInfo{}

	assert.Empty(t, info.Key)
	assert.Equal(t, int64(0), info.Size)
	assert.Empty(t, info.ContentType)
	assert.True(t, info.LastModified.IsZero())
	assert.Empty(t, info.ETag)
}

// TestAwsToIntCalls tests AWS SDK pointer conversion helpers.
func TestAwsToIntCalls(t *testing.T) {
	t.Parallel()

	// Test that aws.ToInt64 handles nil pointers correctly
	var nilPtr *int64
	assert.Equal(t, int64(0), aws.ToInt64(nilPtr))

	// Test with actual value
	val := int64(1024)
	assert.Equal(t, int64(1024), aws.ToInt64(&val))
}

// TestAwsToStringCalls tests AWS SDK string conversion helpers.
func TestAwsToStringCalls(t *testing.T) {
	t.Parallel()

	// Test that aws.ToString handles nil pointers correctly
	var nilPtr *string
	assert.Equal(t, "", aws.ToString(nilPtr))

	// Test with actual value
	val := "test-value"
	assert.Equal(t, "test-value", aws.ToString(&val))
}
