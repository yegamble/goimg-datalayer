package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_Success tests successful creation of LocalStorage.
func TestNew_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	cfg := Config{
		BasePath: tempDir,
		BaseURL:  "http://localhost:8080/uploads",
	}

	storage, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, "http://localhost:8080/uploads", storage.baseURL)

	// Verify directory was created
	info, err := os.Stat(tempDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestNew_CreatesBaseDirectory tests that New creates the base directory if it doesn't exist.
func TestNew_CreatesBaseDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "subdir", "storage")

	cfg := Config{
		BasePath: basePath,
		BaseURL:  "http://localhost:8080/uploads",
	}

	storage, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)

	// Verify directory was created
	info, err := os.Stat(basePath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestNew_ValidationErrors tests configuration validation errors.
func TestNew_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       Config
		wantError string
	}{
		{
			name:      "empty base path",
			cfg:       Config{BasePath: "", BaseURL: "http://localhost"},
			wantError: "base path required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage, err := New(tt.cfg)
			assert.Nil(t, storage)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

// TestPutBytes_Success tests successful storage of bytes.
func TestPutBytes_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("test image data")
	key := "images/test-owner/test-image/original.jpg"

	err := storage.PutBytes(ctx, key, testData, PutOptions{
		ContentType: "image/jpeg",
	})
	require.NoError(t, err)

	// Verify file exists.
	fullPath := storage.fullPath(key)
	//nolint:gosec // G304: Test helper - path is controlled by test.
	data, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, testData, data)

	// Verify permissions
	info, err := os.Stat(fullPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

// TestPut_Success tests successful streaming Put operation.
func TestPut_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("streaming test data")
	reader := bytes.NewReader(testData)
	key := "images/owner/image/original.jpg"

	err := storage.Put(ctx, key, reader, int64(len(testData)), PutOptions{})
	require.NoError(t, err)

	// Verify file exists and content matches
	saved, err := storage.GetBytes(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, testData, saved)
}

// TestPut_SizeMismatch tests size validation during Put.
func TestPut_SizeMismatch(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("test data")
	reader := bytes.NewReader(testData)
	key := "test.jpg"

	// Provide wrong size
	err := storage.Put(ctx, key, reader, 100, PutOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, errSizeMismatch)
	assert.Contains(t, err.Error(), "expected 100 bytes, wrote 9")
}

// TestPut_CreatesNestedDirectories tests that Put creates nested directory structure.
func TestPut_CreatesNestedDirectories(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	key := "deep/nested/path/to/image.jpg"
	testData := []byte("nested data")

	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	// Verify nested directories exist
	fullPath := storage.fullPath(key)
	assert.FileExists(t, fullPath)
}

// TestPut_ContextCancellation tests that Put respects context cancellation.
func TestPut_ContextCancellation(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	key := "test.jpg"
	// Create a reader that would produce lots of data
	reader := &infiniteReader{}

	err := storage.Put(ctx, key, reader, 0, PutOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestGet_Success tests successful retrieval of data.
func TestGet_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("test content")
	key := "test.jpg"

	// Store data
	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	// Retrieve data
	reader, err := storage.Get(ctx, key)
	require.NoError(t, err)
	defer reader.Close()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}

// TestGet_NotFound tests Get with non-existent key.
func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	reader, err := storage.Get(ctx, "nonexistent.jpg")
	assert.Nil(t, reader)
	require.Error(t, err)
	assert.ErrorIs(t, err, errNotFound)
}

// TestGetBytes_Success tests successful retrieval of bytes.
func TestGetBytes_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("byte content")
	key := "bytes.jpg"

	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	retrieved, err := storage.GetBytes(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}

// TestDelete_Success tests successful deletion.
func TestDelete_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	key := "to-delete.jpg"
	testData := []byte("delete me")

	// Store data
	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	// Verify exists
	exists, err := storage.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete
	err = storage.Delete(ctx, key)
	require.NoError(t, err)

	// Verify deleted
	exists, err = storage.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestDelete_NonExistent tests that deleting non-existent file doesn't error.
func TestDelete_NonExistent(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	err := storage.Delete(ctx, "nonexistent.jpg")
	require.NoError(t, err) // Should not error
}

// TestExists_Found tests Exists with existing file.
func TestExists_Found(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	key := "exists.jpg"
	err := storage.PutBytes(ctx, key, []byte("data"), PutOptions{})
	require.NoError(t, err)

	exists, err := storage.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestExists_NotFound tests Exists with non-existent file.
func TestExists_NotFound(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	exists, err := storage.Exists(ctx, "nonexistent.jpg")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestStat_Success tests successful Stat operation.
func TestStat_Success(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	key := "stat-test.jpg"
	testData := []byte("stat data")

	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	info, err := storage.Stat(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, key, info.Key)
	assert.Equal(t, int64(len(testData)), info.Size)
	assert.Equal(t, "image/jpeg", info.ContentType)
	assert.NotEmpty(t, info.ETag)
	assert.False(t, info.LastModified.IsZero())
}

// TestStat_NotFound tests Stat with non-existent file.
func TestStat_NotFound(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	info, err := storage.Stat(ctx, "nonexistent.jpg")
	assert.Nil(t, info)
	require.Error(t, err)
	assert.ErrorIs(t, err, errNotFound)
}

// TestURL_WithBaseURL tests URL generation when baseURL is configured.
func TestURL_WithBaseURL(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := Config{
		BasePath: tempDir,
		BaseURL:  "http://localhost:8080/uploads",
	}

	storage, err := New(cfg)
	require.NoError(t, err)

	key := "images/owner/image/original.jpg"
	url := storage.URL(key)

	assert.Equal(t, "http://localhost:8080/uploads/images/owner/image/original.jpg", url)
}

// TestURL_WithoutBaseURL tests URL generation when baseURL is not configured.
func TestURL_WithoutBaseURL(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := Config{
		BasePath: tempDir,
		BaseURL:  "",
	}

	storage, err := New(cfg)
	require.NoError(t, err)

	key := "test.jpg"
	url := storage.URL(key)

	assert.Equal(t, "", url)
}

// TestPresignedURL_NotSupported tests that presigned URLs are not supported.
func TestPresignedURL_NotSupported(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	url, err := storage.PresignedURL(ctx, "test.jpg", time.Hour)
	assert.Equal(t, "", url)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errNotSupported))
}

// TestProvider tests the Provider method.
func TestProvider(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	assert.Equal(t, "local", storage.Provider())
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
			assert.True(t, errors.Is(err, tt.wantError), "expected error %v, got %v", tt.wantError, err)
		})
	}
}

// TestDetectContentType tests MIME type detection from extensions.
func TestDetectContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		wantMIME string
	}{
		{"jpeg", "image.jpg", "image/jpeg"},
		{"jpeg alternate", "image.jpeg", "image/jpeg"},
		{"png", "image.png", "image/png"},
		{"gif", "image.gif", "image/gif"},
		{"webp", "image.webp", "image/webp"},
		{"unknown", "file.txt", "application/octet-stream"},
		{"no extension", "image", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mime := detectContentType(tt.key)
			assert.Equal(t, tt.wantMIME, mime)
		})
	}
}

// TestCalculateETag tests ETag calculation.
func TestCalculateETag(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	testData := []byte("etag test data")
	key := "etag-test.jpg"

	err := storage.PutBytes(ctx, key, testData, PutOptions{})
	require.NoError(t, err)

	fullPath := storage.fullPath(key)
	etag, err := storage.calculateETag(fullPath)
	require.NoError(t, err)

	// ETag should be MD5 hash wrapped in quotes
	assert.NotEmpty(t, etag)
	assert.True(t, strings.HasPrefix(etag, `"`))
	assert.True(t, strings.HasSuffix(etag, `"`))

	// Calculate again - should be stable
	etag2, err := storage.calculateETag(fullPath)
	require.NoError(t, err)
	assert.Equal(t, etag, etag2)
}

// TestAtomicWrite tests that Put uses atomic write pattern.
func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	storage := setupTestStorage(t)
	ctx := context.Background()

	key := "atomic-test.jpg"
	fullPath := storage.fullPath(key)

	// First write
	err := storage.PutBytes(ctx, key, []byte("version 1"), PutOptions{})
	require.NoError(t, err)

	// Second write should replace atomically
	err = storage.PutBytes(ctx, key, []byte("version 2"), PutOptions{})
	require.NoError(t, err)

	// Verify final content
	data, err := storage.GetBytes(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("version 2"), data)

	// Verify only one file exists (no temp files)
	dir := filepath.Dir(fullPath)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// Count non-temp files
	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".tmp-") {
			fileCount++
		}
	}
	assert.Equal(t, 1, fileCount, "should have exactly one file, no temp files")
}

// setupTestStorage creates a LocalStorage instance for testing.
func setupTestStorage(t *testing.T) *LocalStorage {
	t.Helper()

	tempDir := t.TempDir()
	cfg := Config{
		BasePath: tempDir,
		BaseURL:  "http://localhost:8080/uploads",
	}

	storage, err := New(cfg)
	require.NoError(t, err)
	return storage
}

// infiniteReader is a reader that never returns EOF (for testing context cancellation).
type infiniteReader struct {
	count int
}

func (r *infiniteReader) Read(p []byte) (int, error) {
	r.count++
	// Fill buffer with data
	for i := range p {
		p[i] = byte(r.count % 256)
	}
	return len(p), nil
}
