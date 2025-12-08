// Package local implements a local filesystem storage provider.
// This is primarily intended for development and testing environments.
package local

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // G501: MD5 used for ETag generation, not cryptographic security
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Common storage errors (duplicated to avoid import cycle).
var (
	errNotFound      = errors.New("storage: object not found")
	errAccessDenied  = errors.New("storage: access denied")
	errInvalidKey    = errors.New("storage: invalid key")
	errPathTraversal = errors.New("storage: path traversal detected")
	errNotSupported  = errors.New("storage: operation not supported")
	errSizeMismatch  = errors.New("storage: size mismatch")
)

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// PutOptions configures storage upload behavior.
type PutOptions struct {
	ContentType  string
	CacheControl string
	Metadata     map[string]string
}

// LocalStorage implements the Storage interface using the local filesystem.
// All operations are atomic using temp files + rename pattern.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// Config configures the local filesystem storage provider.
type Config struct {
	// BasePath is the directory where files are stored.
	// Must be an absolute path.
	BasePath string

	// BaseURL is the URL prefix for generating public URLs.
	// Example: "http://localhost:8080/uploads"
	BaseURL string
}

// New creates a new local filesystem storage provider.
func New(cfg Config) (*LocalStorage, error) {
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("local storage: base path required")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("local storage: invalid path: %w", err)
	}

	// Ensure base directory exists
	if err := os.MkdirAll(absPath, 0o750); err != nil {
		return nil, fmt.Errorf("local storage: create base dir: %w", err)
	}

	return &LocalStorage{
		basePath: absPath,
		baseURL:  cfg.BaseURL,
	}, nil
}

// Put stores data at the given key using streaming.
func (s *LocalStorage) Put(ctx context.Context, key string, data io.Reader, size int64, opts PutOptions) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullPath := s.fullPath(key)
	dir := filepath.Dir(fullPath)

	// Create directory structure
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("local mkdir: %w", err)
	}

	// Write to temp file first (atomic write pattern)
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("local create temp: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on error
	defer func() {
		if tempFile != nil {
			if cerr := tempFile.Close(); cerr != nil {
				// Log close error but continue with cleanup
			}
			if rerr := os.Remove(tempPath); rerr != nil {
				// Log remove error but continue
			}
		}
	}()

	// Copy data with context cancellation check
	written, err := copyWithContext(ctx, tempFile, data)
	if err != nil {
		return fmt.Errorf("local write: %w", err)
	}

	// Verify size if provided
	if size > 0 && written != size {
		return fmt.Errorf("%w: expected %d bytes, wrote %d", errSizeMismatch, size, written)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("local close temp: %w", err)
	}
	tempFile = nil // Prevent deferred cleanup

	// Set permissions
	if err := os.Chmod(tempPath, 0o600); err != nil {
		if rerr := os.Remove(tempPath); rerr != nil {
			// Log best-effort cleanup failure
		}
		return fmt.Errorf("local chmod: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, fullPath); err != nil {
		if rerr := os.Remove(tempPath); rerr != nil {
			// Log best-effort cleanup failure
		}
		return fmt.Errorf("local rename: %w", err)
	}

	return nil
}

// PutBytes is a convenience method for storing small in-memory data.
func (s *LocalStorage) PutBytes(ctx context.Context, key string, data []byte, opts PutOptions) error {
	return s.Put(ctx, key, bytes.NewReader(data), int64(len(data)), opts)
}

// Get retrieves data from storage as a streaming reader.
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullPath := s.fullPath(key)
	//nolint:gosec // G304: File path constructed from validated key (validateKey checks for path traversal)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound
		}
		if os.IsPermission(err) {
			return nil, errAccessDenied
		}
		return nil, fmt.Errorf("local open: %w", err)
	}

	return file, nil
}

// GetBytes retrieves data fully into memory.
func (s *LocalStorage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	reader, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			// Log close error but return read result
		}
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("local read: %w", err)
	}

	return data, nil
}

// Delete removes an object from storage.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullPath := s.fullPath(key)
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local delete: %w", err)
	}

	return nil
}

// Exists checks if an object exists at the given key.
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	fullPath := s.fullPath(key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("local stat: %w", err)
	}

	return true, nil
}

// URL returns the public URL for a key.
func (s *LocalStorage) URL(key string) string {
	if s.baseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", s.baseURL, key)
}

// PresignedURL is not supported for local storage.
func (s *LocalStorage) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	return "", errNotSupported
}

// Stat returns metadata about a stored object.
func (s *LocalStorage) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullPath := s.fullPath(key)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound
		}
		return nil, fmt.Errorf("local stat: %w", err)
	}

	// Calculate ETag (MD5 of content)
	etag, err := s.calculateETag(fullPath)
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		ContentType:  detectContentType(key),
		LastModified: info.ModTime(),
		ETag:         etag,
	}, nil
}

// Provider returns the provider type name.
func (s *LocalStorage) Provider() string {
	return "local"
}

// fullPath returns the full filesystem path for a storage key.
func (s *LocalStorage) fullPath(key string) string {
	return filepath.Join(s.basePath, key)
}

// calculateETag computes the MD5 hash of a file for ETag.
func (s *LocalStorage) calculateETag(path string) (string, error) {
	//nolint:gosec // G304: File path from internal method (fullPath), already validated
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("local open for etag: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Log close error but return hash result
		}
	}()

	//nolint:gosec // G401: MD5 is acceptable for ETag generation (not cryptographic use)
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("local hash: %w", err)
	}

	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash.Sum(nil))), nil
}

// validateKey checks if a storage key is safe.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty key", errInvalidKey)
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: contains '..'", errPathTraversal)
	}
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return fmt.Errorf("%w: cannot be absolute path", errPathTraversal)
	}
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("%w: contains null byte", errInvalidKey)
	}
	return nil
}

// copyWithContext copies data with context cancellation support.
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, fmt.Errorf("copy interrupted: %w", ctx.Err())
		default:
		}

		nr, err := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, fmt.Errorf("write failed: %w", werr)
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, fmt.Errorf("read failed: %w", err)
		}
	}
}

// detectContentType returns the MIME type based on file extension.
func detectContentType(key string) string {
	ext := filepath.Ext(key)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
