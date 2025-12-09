// Package storage provides a unified interface for image storage backends.
// It supports multiple providers including local filesystem, S3-compatible
// storage (AWS S3, DigitalOcean Spaces, Backblaze B2, MinIO), and future
// IPFS integration.
package storage

import (
	"context"
	"io"
	"time"
)

// Storage is the interface that all storage providers must implement.
// Implementations must be safe for concurrent use.
type Storage interface {
	// Put stores data at the given key using streaming.
	// For small files, use PutBytes. For large files (>1MB), use Put with io.Reader.
	Put(ctx context.Context, key string, data io.Reader, size int64, opts PutOptions) error

	// PutBytes is a convenience method for storing small in-memory data.
	// Use for thumbnails and small files. For large files, use Put.
	PutBytes(ctx context.Context, key string, data []byte, opts PutOptions) error

	// Get retrieves data from storage as a streaming reader.
	// Callers MUST close the returned ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// GetBytes retrieves data fully into memory.
	// Use only for thumbnails or metadata. For large files, use Get.
	GetBytes(ctx context.Context, key string) ([]byte, error)

	// Delete removes an object from storage.
	// This operation is idempotent - deleting a non-existent key succeeds.
	Delete(ctx context.Context, key string) error

	// Exists checks if an object exists at the given key.
	Exists(ctx context.Context, key string) (bool, error)

	// URL returns the public URL for a key (if applicable).
	// Returns empty string if public URLs are not supported by this provider.
	URL(key string) string

	// PresignedURL generates a temporary signed URL for secure access.
	// duration specifies how long the URL remains valid.
	// Returns ErrNotSupported if presigned URLs are not available.
	PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error)

	// Stat returns metadata about a stored object.
	Stat(ctx context.Context, key string) (*ObjectInfo, error)

	// Provider returns the provider type name (e.g., "local", "s3").
	Provider() string
}

// PutOptions configures storage upload behavior.
type PutOptions struct {
	// ContentType is the MIME type (e.g., "image/jpeg").
	ContentType string

	// CacheControl sets the Cache-Control header for CDN caching.
	// Example: "public, max-age=31536000" for immutable content.
	CacheControl string

	// Metadata contains provider-specific metadata key-value pairs.
	Metadata map[string]string
}

// DefaultPutOptions returns sensible defaults for image storage.
func DefaultPutOptions(contentType string) PutOptions {
	return PutOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000, immutable",
	}
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	// Key is the object's storage key.
	Key string

	// Size is the object size in bytes.
	Size int64

	// ContentType is the MIME type.
	ContentType string

	// LastModified is when the object was last modified.
	LastModified time.Time

	// ETag is the entity tag for cache validation.
	ETag string
}

// ProviderType identifies the storage backend type.
type ProviderType string

const (
	// ProviderLocal uses local filesystem storage.
	ProviderLocal ProviderType = "local"
	// ProviderS3 uses Amazon S3 storage.
	ProviderS3 ProviderType = "s3"
	// ProviderSpaces uses DigitalOcean Spaces storage.
	ProviderSpaces ProviderType = "spaces"
	// ProviderB2 uses Backblaze B2 storage.
	ProviderB2 ProviderType = "b2"
	// ProviderMinIO uses MinIO object storage.
	ProviderMinIO ProviderType = "minio"
	// ProviderIPFS uses IPFS distributed storage (future).
	ProviderIPFS ProviderType = "ipfs"
)
