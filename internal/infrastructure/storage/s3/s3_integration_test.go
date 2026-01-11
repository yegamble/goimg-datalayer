//go:build integration

package s3_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/s3"
)

const (
	testBucket    = "test-bucket"
	minioRootUser = "minioadmin"
	minioRootPass = "minioadmin"
	minioPort     = "9000/tcp"
)

// setupMinIO starts a MinIO container for testing.
func setupMinIO(t *testing.T, ctx context.Context) (string, func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{minioPort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     minioRootUser,
			"MINIO_ROOT_PASSWORD": minioRootPass,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForLog("API:").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	endpoint, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	// Return endpoint without scheme (will be added by config)
	return "http://" + endpoint, cleanup
}

// createTestStorage creates a Storage instance connected to MinIO.
func createTestStorage(t *testing.T, ctx context.Context, endpoint string) *s3.Storage {
	t.Helper()

	cfg := s3.Config{
		Endpoint:        endpoint,
		Bucket:          testBucket,
		Region:          "us-east-1",
		AccessKeyID:     minioRootUser,
		SecretAccessKey: minioRootPass,
		ForcePathStyle:  true, // Required for MinIO
	}

	storage, err := s3.New(ctx, cfg)
	require.NoError(t, err)

	return storage
}

// TestIntegration_PutAndGet tests uploading and retrieving data.
func TestIntegration_PutAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	tests := []struct {
		name        string
		key         string
		data        []byte
		opts        s3.PutOptions
		description string
	}{
		{
			name:        "simple upload",
			key:         "test/simple.txt",
			data:        []byte("hello world"),
			opts:        s3.PutOptions{},
			description: "Upload and retrieve simple text file",
		},
		{
			name: "with content type",
			key:  "test/image.jpg",
			data: []byte("fake image data"),
			opts: s3.PutOptions{
				ContentType: "image/jpeg",
			},
			description: "Upload with Content-Type header",
		},
		{
			name: "with cache control",
			key:  "test/cached.png",
			data: []byte("cached data"),
			opts: s3.PutOptions{
				ContentType:  "image/png",
				CacheControl: "max-age=3600",
			},
			description: "Upload with Cache-Control header",
		},
		{
			name:        "large file",
			key:         "test/large.bin",
			data:        make([]byte, 1024*1024), // 1MB
			opts:        s3.PutOptions{},
			description: "Upload larger file",
		},
		{
			name:        "nested path",
			key:         "a/b/c/d/e/file.txt",
			data:        []byte("deeply nested"),
			opts:        s3.PutOptions{},
			description: "Upload to deeply nested path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Upload data
			err := storage.Put(ctx, tt.key, bytes.NewReader(tt.data), int64(len(tt.data)), tt.opts)
			require.NoError(t, err, "Put should succeed")

			// Retrieve data
			reader, err := storage.Get(ctx, tt.key)
			require.NoError(t, err, "Get should succeed")
			defer reader.Close()

			// Read all data
			retrieved, err := io.ReadAll(reader)
			require.NoError(t, err, "ReadAll should succeed")

			// Verify data matches
			assert.Equal(t, tt.data, retrieved, "Retrieved data should match uploaded data")

			// Test GetBytes convenience method
			data, err := storage.GetBytes(ctx, tt.key)
			require.NoError(t, err, "GetBytes should succeed")
			assert.Equal(t, tt.data, data, "GetBytes should return same data")
		})
	}
}

// TestIntegration_PutBytes tests the PutBytes convenience method.
func TestIntegration_PutBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	testData := []byte("test data for PutBytes")
	testKey := "test/putbytes.txt"

	// Upload using PutBytes
	err := storage.PutBytes(ctx, testKey, testData, s3.PutOptions{
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	// Verify it was uploaded
	retrieved, err := storage.GetBytes(ctx, testKey)
	require.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}

// TestIntegration_Delete tests deletion of objects.
func TestIntegration_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	testKey := "test/delete-me.txt"
	testData := []byte("this will be deleted")

	// Upload first
	err := storage.PutBytes(ctx, testKey, testData, s3.PutOptions{})
	require.NoError(t, err)

	// Verify it exists
	exists, err := storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists, "Object should exist after upload")

	// Delete it
	err = storage.Delete(ctx, testKey)
	require.NoError(t, err)

	// Verify it's gone
	exists, err = storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.False(t, exists, "Object should not exist after deletion")

	// Try to get deleted object - should return ErrNotFound
	_, err = storage.Get(ctx, testKey)
	require.Error(t, err)
	assert.ErrorIs(t, err, s3.ErrNotFound)
}

// TestIntegration_Exists tests the Exists method.
func TestIntegration_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	// Check non-existent key
	exists, err := storage.Exists(ctx, "test/does-not-exist.txt")
	require.NoError(t, err)
	assert.False(t, exists, "Non-existent object should return false")

	// Upload a file
	testKey := "test/exists.txt"
	err = storage.PutBytes(ctx, testKey, []byte("test"), s3.PutOptions{})
	require.NoError(t, err)

	// Check it exists
	exists, err = storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists, "Uploaded object should exist")
}

// TestIntegration_Stat tests the Stat method.
func TestIntegration_Stat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	testKey := "test/stat-me.txt"
	testData := []byte("test data for stats")
	contentType := "text/plain"

	// Upload file
	err := storage.PutBytes(ctx, testKey, testData, s3.PutOptions{
		ContentType: contentType,
	})
	require.NoError(t, err)

	// Get stats
	info, err := storage.Stat(ctx, testKey)
	require.NoError(t, err)
	assert.NotNil(t, info)

	// Verify fields
	assert.Equal(t, testKey, info.Key)
	assert.Equal(t, int64(len(testData)), info.Size)
	assert.Equal(t, contentType, info.ContentType)
	assert.NotEmpty(t, info.ETag)
	assert.False(t, info.LastModified.IsZero())

	// Stat non-existent file
	_, err = storage.Stat(ctx, "test/does-not-exist.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, s3.ErrNotFound)
}

// TestIntegration_URL tests URL generation.
func TestIntegration_URL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	testKey := "test/url-test.txt"

	// Get URL (default AWS format)
	url := storage.URL(testKey)
	assert.Contains(t, url, testBucket)
	assert.Contains(t, url, testKey)
	assert.Contains(t, url, "s3.amazonaws.com")
}

// TestIntegration_PresignedURL tests presigned URL generation.
func TestIntegration_PresignedURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	testKey := "test/presign-me.txt"
	testData := []byte("presigned data")

	// Upload file
	err := storage.PutBytes(ctx, testKey, testData, s3.PutOptions{})
	require.NoError(t, err)

	// Generate presigned URL with default expiry
	url, err := storage.PresignedURL(ctx, testKey, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, testBucket)
	assert.Contains(t, url, testKey)

	// Generate presigned URL with custom expiry
	customDuration := 30 * time.Minute
	url, err = storage.PresignedURL(ctx, testKey, customDuration)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
}

// TestIntegration_Provider tests the Provider method.
func TestIntegration_Provider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	assert.Equal(t, "s3", storage.Provider())
}

// TestIntegration_ErrorHandling tests error scenarios.
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	t.Run("get non-existent object", func(t *testing.T) {
		_, err := storage.Get(ctx, "test/not-found.txt")
		require.Error(t, err)
		assert.ErrorIs(t, err, s3.ErrNotFound)
	})

	t.Run("get bytes non-existent object", func(t *testing.T) {
		_, err := storage.GetBytes(ctx, "test/not-found.txt")
		require.Error(t, err)
		assert.ErrorIs(t, err, s3.ErrNotFound)
	})

	t.Run("stat non-existent object", func(t *testing.T) {
		_, err := storage.Stat(ctx, "test/not-found.txt")
		require.Error(t, err)
		assert.ErrorIs(t, err, s3.ErrNotFound)
	})

	t.Run("invalid key - empty", func(t *testing.T) {
		err := storage.PutBytes(ctx, "", []byte("data"), s3.PutOptions{})
		require.Error(t, err)
	})

	t.Run("invalid key - path traversal", func(t *testing.T) {
		err := storage.PutBytes(ctx, "../etc/passwd", []byte("data"), s3.PutOptions{})
		require.Error(t, err)
	})
}

// TestIntegration_ContextCancellation tests context cancellation.
func TestIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	endpoint, cleanup := setupMinIO(t, ctx)
	defer cleanup()

	storage := createTestStorage(t, ctx, endpoint)

	t.Run("cancelled context for Put", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		err := storage.PutBytes(cancelledCtx, "test/cancelled.txt", []byte("data"), s3.PutOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	// Upload a file first for Get test
	testKey := "test/cancel-get.txt"
	err := storage.PutBytes(ctx, testKey, []byte("test data"), s3.PutOptions{})
	require.NoError(t, err)

	t.Run("cancelled context for Get", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		_, err := storage.Get(cancelledCtx, testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
