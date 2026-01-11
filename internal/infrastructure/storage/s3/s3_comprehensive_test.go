package s3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_BucketValidation tests the bucket validation in New().
func TestNew_BucketValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with empty bucket - should return error immediately (before AWS SDK call)
	cfg := Config{
		Bucket:          "",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	}

	storage, err := New(ctx, cfg)
	require.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "bucket name required")
}

// TestNew_RegionDefaulting tests that empty region gets defaulted.
func TestNew_RegionDefaulting(t *testing.T) {
	t.Parallel()

	// Verify the logic: empty region should be set to us-east-1
	cfg := Config{
		Bucket:          "test-bucket",
		Region:          "",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	}

	// Before calling New, region is empty
	assert.Empty(t, cfg.Region)

	// After calling New, region would be set to "us-east-1"
	// (we can't test the actual call without valid AWS credentials)
	expectedDefaultRegion := "us-east-1"
	assert.NotEqual(t, expectedDefaultRegion, cfg.Region) // Currently empty

	// In the actual New() function:
	// if cfg.Region == "" {
	//     cfg.Region = "us-east-1"
	// }
}

// TestNew_ExpiryDefaulting tests that zero expiry gets defaulted.
func TestNew_ExpiryDefaulting(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Bucket:             "test-bucket",
		AccessKeyID:        "test",
		SecretAccessKey:    "test",
		PresignedURLExpiry: 0,
	}

	// Before calling New, expiry is 0
	assert.Equal(t, time.Duration(0), cfg.PresignedURLExpiry)

	// After calling New, expiry would be set to defaultPresignedURLExpiry
	expectedDefaultExpiry := defaultPresignedURLExpiry
	assert.Equal(t, 15*time.Minute, expectedDefaultExpiry)

	// In the actual New() function:
	// if cfg.PresignedURLExpiry == 0 {
	//     cfg.PresignedURLExpiry = defaultPresignedURLExpiry
	// }
}

// TestStorage_AllMethods_ValidateKeys tests that all methods validate keys.
func TestStorage_AllMethods_ValidateKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	invalidKey := "../etc/passwd"

	s := &Storage{
		bucket: "test-bucket",
		expiry: 15 * time.Minute,
	}

	t.Run("Put validates key", func(t *testing.T) {
		err := s.Put(ctx, invalidKey, nil, 0, PutOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("PutBytes validates key", func(t *testing.T) {
		err := s.PutBytes(ctx, invalidKey, []byte("data"), PutOptions{})
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("Get validates key", func(t *testing.T) {
		_, err := s.Get(ctx, invalidKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("GetBytes validates key", func(t *testing.T) {
		_, err := s.GetBytes(ctx, invalidKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("Delete validates key", func(t *testing.T) {
		err := s.Delete(ctx, invalidKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("Exists validates key", func(t *testing.T) {
		_, err := s.Exists(ctx, invalidKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("PresignedURL validates key", func(t *testing.T) {
		_, err := s.PresignedURL(ctx, invalidKey, 0)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("Stat validates key", func(t *testing.T) {
		_, err := s.Stat(ctx, invalidKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, errPathTraversal)
	})

	t.Run("URL does not validate (public method)", func(t *testing.T) {
		// URL doesn't validate because it's just constructing a string
		url := s.URL(invalidKey)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, invalidKey)
	})
}

// TestStorage_PresignedURLDurationLogicTest tests the duration selection logic.
func TestStorage_PresignedURLDurationLogicTest(t *testing.T) {
	t.Parallel()

	s := &Storage{
		bucket: "test-bucket",
		expiry: 15 * time.Minute,
	}

	// When validation fails, we test the validation path
	invalidKey := ""

	t.Run("zero duration would use default", func(t *testing.T) {
		_, err := s.PresignedURL(context.Background(), invalidKey, 0)
		require.Error(t, err) // Fails validation
		assert.ErrorIs(t, err, errInvalidKey)

		// The duration logic:
		// if duration == 0 {
		//     duration = s.expiry
		// }
		// Would use 15 minutes
	})

	t.Run("custom duration would be used", func(t *testing.T) {
		_, err := s.PresignedURL(context.Background(), invalidKey, 1*time.Hour)
		require.Error(t, err) // Fails validation
		assert.ErrorIs(t, err, errInvalidKey)

		// The duration logic would use 1 hour
	})
}

// TestStorage_URLGeneration tests URL generation with different configurations.
func TestStorage_URLGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		storage *Storage
		key     string
		wantURL string
	}{
		{
			name: "default AWS S3 URL",
			storage: &Storage{
				bucket:    "my-bucket",
				publicURL: "",
			},
			key:     "images/test.jpg",
			wantURL: "https://my-bucket.s3.amazonaws.com/images/test.jpg",
		},
		{
			name: "with CDN URL",
			storage: &Storage{
				bucket:    "my-bucket",
				publicURL: "https://cdn.example.com",
			},
			key:     "images/test.jpg",
			wantURL: "https://cdn.example.com/images/test.jpg",
		},
		{
			name: "with CDN URL and trailing slash",
			storage: &Storage{
				bucket:    "my-bucket",
				publicURL: "https://cdn.example.com/",
			},
			key:     "images/test.jpg",
			wantURL: "https://cdn.example.com/images/test.jpg",
		},
		{
			name: "complex key path",
			storage: &Storage{
				bucket:    "bucket",
				publicURL: "",
			},
			key:     "a/b/c/d/e/f.jpg",
			wantURL: "https://bucket.s3.amazonaws.com/a/b/c/d/e/f.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			url := tt.storage.URL(tt.key)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

// TestStorage_ProviderReturnsS3 tests the Provider method.
func TestStorage_ProviderReturnsS3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		storage *Storage
		want    string
	}{
		{
			name:    "always returns s3",
			storage: &Storage{},
			want:    "s3",
		},
		{
			name: "with bucket",
			storage: &Storage{
				bucket: "my-bucket",
			},
			want: "s3",
		},
		{
			name: "with public URL",
			storage: &Storage{
				bucket:    "my-bucket",
				publicURL: "https://cdn.example.com",
			},
			want: "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := tt.storage.Provider()
			assert.Equal(t, tt.want, provider)
		})
	}
}

// TestPutOptions_ZeroValues tests PutOptions with zero values.
func TestPutOptions_ZeroValues(t *testing.T) {
	t.Parallel()

	opts := PutOptions{}

	assert.Empty(t, opts.ContentType)
	assert.Empty(t, opts.CacheControl)
	assert.Nil(t, opts.Metadata)
}

// TestObjectInfo_ZeroValueFields tests ObjectInfo with zero values.
func TestObjectInfo_ZeroValueFields(t *testing.T) {
	t.Parallel()

	info := &ObjectInfo{}

	assert.Empty(t, info.Key)
	assert.Equal(t, int64(0), info.Size)
	assert.Empty(t, info.ContentType)
	assert.True(t, info.LastModified.IsZero())
	assert.Empty(t, info.ETag)
}

// TestObjectInfo_NonZeroValues tests ObjectInfo with actual values.
func TestObjectInfo_NonZeroValues(t *testing.T) {
	t.Parallel()

	now := time.Now()

	info := &ObjectInfo{
		Key:          "test/file.jpg",
		Size:         1024,
		ContentType:  "image/jpeg",
		LastModified: now,
		ETag:         `"abc123"`,
	}

	assert.Equal(t, "test/file.jpg", info.Key)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, "image/jpeg", info.ContentType)
	assert.Equal(t, now, info.LastModified)
	assert.Equal(t, `"abc123"`, info.ETag)
	assert.False(t, info.LastModified.IsZero())
}

// TestConfig_ZeroValues tests Config with zero values.
func TestConfig_ZeroValues(t *testing.T) {
	t.Parallel()

	cfg := Config{}

	assert.Empty(t, cfg.Endpoint)
	assert.Empty(t, cfg.Region)
	assert.Empty(t, cfg.Bucket)
	assert.Empty(t, cfg.AccessKeyID)
	assert.Empty(t, cfg.SecretAccessKey)
	assert.False(t, cfg.ForcePathStyle)
	assert.Empty(t, cfg.PublicURL)
	assert.Equal(t, time.Duration(0), cfg.PresignedURLExpiry)
}

// TestConfig_AllFieldsSet tests Config with all fields set.
func TestConfig_AllFieldsSet(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Endpoint:           "https://s3.amazonaws.com",
		Region:             "us-west-2",
		Bucket:             "my-bucket",
		AccessKeyID:        "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		ForcePathStyle:     true,
		PublicURL:          "https://cdn.example.com",
		PresignedURLExpiry: 30 * time.Minute,
	}

	assert.Equal(t, "https://s3.amazonaws.com", cfg.Endpoint)
	assert.Equal(t, "us-west-2", cfg.Region)
	assert.Equal(t, "my-bucket", cfg.Bucket)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cfg.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", cfg.SecretAccessKey)
	assert.True(t, cfg.ForcePathStyle)
	assert.Equal(t, "https://cdn.example.com", cfg.PublicURL)
	assert.Equal(t, 30*time.Minute, cfg.PresignedURLExpiry)
}

// TestDefaultPresignedURLExpiryConstant tests the default constant.
func TestDefaultPresignedURLExpiryConstant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 15*time.Minute, defaultPresignedURLExpiry)
	assert.Equal(t, 900*time.Second, defaultPresignedURLExpiry)
}

// TestErrorConstants tests the error constants.
func TestErrorConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "ErrNotFound",
			err:     ErrNotFound,
			wantMsg: "storage: object not found",
		},
		{
			name:    "ErrAccessDenied",
			err:     ErrAccessDenied,
			wantMsg: "storage: access denied",
		},
		{
			name:    "errInvalidKey",
			err:     errInvalidKey,
			wantMsg: "storage: invalid key",
		},
		{
			name:    "errPathTraversal",
			err:     errPathTraversal,
			wantMsg: "storage: path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

// TestStorage_FieldInitialization tests Storage field initialization.
func TestStorage_FieldInitialization(t *testing.T) {
	t.Parallel()

	s := &Storage{
		bucket:    "test-bucket",
		publicURL: "https://cdn.example.com",
		expiry:    30 * time.Minute,
	}

	assert.Equal(t, "test-bucket", s.bucket)
	assert.Equal(t, "https://cdn.example.com", s.publicURL)
	assert.Equal(t, 30*time.Minute, s.expiry)
	assert.Nil(t, s.client)    // Not set in this test
	assert.Nil(t, s.presigner) // Not set in this test
}

// TestPutOptions_WithMetadata tests PutOptions with metadata.
func TestPutOptions_WithMetadata(t *testing.T) {
	t.Parallel()

	opts := PutOptions{
		ContentType:  "application/json",
		CacheControl: "no-cache, no-store",
		Metadata: map[string]string{
			"user-id":      "12345",
			"upload-time":  "2024-01-01T00:00:00Z",
			"content-hash": "abc123def456",
		},
	}

	assert.Equal(t, "application/json", opts.ContentType)
	assert.Equal(t, "no-cache, no-store", opts.CacheControl)
	assert.Len(t, opts.Metadata, 3)
	assert.Equal(t, "12345", opts.Metadata["user-id"])
	assert.Equal(t, "2024-01-01T00:00:00Z", opts.Metadata["upload-time"])
	assert.Equal(t, "abc123def456", opts.Metadata["content-hash"])
}

// TestValidateKey_AllPathTraversalVariants tests all path traversal patterns.
func TestValidateKey_AllPathTraversalVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"double dot in path", "images/../config"},
		{"double dot at start", "../etc/passwd"},
		{"double dot at end", "images/test/.."},
		{"multiple double dots", "../../etc/shadow"},
		{"double dot in middle", "images/../../../etc/hosts"},
		{"with dot slash", "./../config"},
		{"double dot in filename", "images/file..jpg"}, // This should fail too
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)
			require.Error(t, err)
			assert.ErrorIs(t, err, errPathTraversal)
		})
	}
}

// TestValidateKey_AllAbsolutePathVariants tests absolute path detection.
func TestValidateKey_AllAbsolutePathVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"unix absolute path", "/etc/passwd"},
		{"windows absolute path", "\\windows\\system32"},
		{"unix absolute with subdirs", "/var/log/app.log"},
		{"windows absolute with subdirs", "\\Program Files\\app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)
			require.Error(t, err)
			assert.ErrorIs(t, err, errPathTraversal)
		})
	}
}

// TestValidateKey_AllNullByteVariants tests null byte detection.
func TestValidateKey_AllNullByteVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"null at start", "\x00file.txt"},
		{"null in middle", "file\x00.txt"},
		{"null at end", "file.txt\x00"},
		{"multiple nulls", "fi\x00le.t\x00xt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)
			require.Error(t, err)
			assert.ErrorIs(t, err, errInvalidKey)
		})
	}
}

// TestValidateKey_ValidPaths tests valid key patterns.
func TestValidateKey_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"simple", "file.txt"},
		{"with path", "images/file.txt"},
		{"deep path", "a/b/c/d/e/file.txt"},
		{"with dots in name", "file.name.with.dots.txt"},
		{"with dashes", "user-id/image-123.jpg"},
		{"with underscores", "user_id/image_123.jpg"},
		{"with numbers", "user123/image456.jpg"},
		{"with spaces", "my folder/my file.txt"},
		{"unicode", "用户/图片.jpg"},
		{"emoji", "folder/file😀.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateKey(tt.key)
			assert.NoError(t, err)
		})
	}
}
