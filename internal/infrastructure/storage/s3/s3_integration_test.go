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

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/s3"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func newTestStorage(t *testing.T, ctx context.Context, c *containers.MinIOContainer) *s3.Storage {
	t.Helper()

	cfg := s3.Config{
		Endpoint:        c.Endpoint,
		Bucket:          c.BucketName,
		Region:          "us-east-1",
		AccessKeyID:     c.AccessKey,
		SecretAccessKey: c.SecretKey,
		ForcePathStyle:  true,
	}

	storage, err := s3.New(ctx, cfg)
	require.NoError(t, err)

	return storage
}

func TestIntegration_PutAndGet(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

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
			data:        make([]byte, 1024*1024),
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
			err := storage.Put(ctx, tt.key, bytes.NewReader(tt.data), int64(len(tt.data)), tt.opts)
			require.NoError(t, err, "Put should succeed")

			reader, err := storage.Get(ctx, tt.key)
			require.NoError(t, err, "Get should succeed")
			defer reader.Close()

			retrieved, err := io.ReadAll(reader)
			require.NoError(t, err, "ReadAll should succeed")

			assert.Equal(t, tt.data, retrieved, "Retrieved data should match uploaded data")

			data, err := storage.GetBytes(ctx, tt.key)
			require.NoError(t, err, "GetBytes should succeed")
			assert.Equal(t, tt.data, data, "GetBytes should return same data")
		})
	}
}

func TestIntegration_PutBytes(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	testData := []byte("test data for PutBytes")
	testKey := "test/putbytes.txt"

	err = storage.PutBytes(ctx, testKey, testData, s3.PutOptions{
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	retrieved, err := storage.GetBytes(ctx, testKey)
	require.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}

func TestIntegration_Delete(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	testKey := "test/delete-me.txt"
	testData := []byte("this will be deleted")

	err = storage.PutBytes(ctx, testKey, testData, s3.PutOptions{})
	require.NoError(t, err)

	exists, err := storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists, "Object should exist after upload")

	err = storage.Delete(ctx, testKey)
	require.NoError(t, err)

	exists, err = storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.False(t, exists, "Object should not exist after deletion")

	_, err = storage.Get(ctx, testKey)
	require.Error(t, err)
	assert.ErrorIs(t, err, s3.ErrNotFound)
}

func TestIntegration_Exists(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	exists, err := storage.Exists(ctx, "test/does-not-exist.txt")
	require.NoError(t, err)
	assert.False(t, exists, "Non-existent object should return false")

	testKey := "test/exists.txt"
	err = storage.PutBytes(ctx, testKey, []byte("test"), s3.PutOptions{})
	require.NoError(t, err)

	exists, err = storage.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists, "Uploaded object should exist")
}

func TestIntegration_Stat(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	testKey := "test/stat-me.txt"
	testData := []byte("test data for stats")
	contentType := "text/plain"

	err = storage.PutBytes(ctx, testKey, testData, s3.PutOptions{
		ContentType: contentType,
	})
	require.NoError(t, err)

	info, err := storage.Stat(ctx, testKey)
	require.NoError(t, err)
	assert.NotNil(t, info)

	assert.Equal(t, testKey, info.Key)
	assert.Equal(t, int64(len(testData)), info.Size)
	assert.Equal(t, contentType, info.ContentType)
	assert.NotEmpty(t, info.ETag)
	assert.False(t, info.LastModified.IsZero())

	_, err = storage.Stat(ctx, "test/does-not-exist.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, s3.ErrNotFound)
}

func TestIntegration_URL(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	testKey := "test/url-test.txt"

	url := storage.URL(testKey)
	assert.Contains(t, url, minioC.BucketName)
	assert.Contains(t, url, testKey)
	assert.Contains(t, url, "s3.amazonaws.com")
}

func TestIntegration_PresignedURL(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	testKey := "test/presign-me.txt"
	testData := []byte("presigned data")

	err = storage.PutBytes(ctx, testKey, testData, s3.PutOptions{})
	require.NoError(t, err)

	url, err := storage.PresignedURL(ctx, testKey, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, minioC.BucketName)
	assert.Contains(t, url, testKey)

	customDuration := 30 * time.Minute
	url, err = storage.PresignedURL(ctx, testKey, customDuration)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
}

func TestIntegration_Provider(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	assert.Equal(t, "s3", storage.Provider())
}

func TestIntegration_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

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

func TestIntegration_ContextCancellation(t *testing.T) {
	ctx := context.Background()

	minioC, err := containers.NewMinIOContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, minioC)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })

	storage := newTestStorage(t, ctx, minioC)

	t.Run("cancelled context for Put", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		err := storage.PutBytes(cancelledCtx, "test/cancelled.txt", []byte("data"), s3.PutOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	testKey := "test/cancel-get.txt"
	err = storage.PutBytes(ctx, testKey, []byte("test data"), s3.PutOptions{})
	require.NoError(t, err)

	t.Run("cancelled context for Get", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := storage.Get(cancelledCtx, testKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
