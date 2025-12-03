package storage

import (
	"errors"
	"fmt"
)

// Storage error types.
// Use errors.Is() to check for these error types.
var (
	// ErrNotFound indicates the object does not exist.
	ErrNotFound = errors.New("storage: object not found")

	// ErrAccessDenied indicates permission was denied.
	ErrAccessDenied = errors.New("storage: access denied")

	// ErrQuotaExceeded indicates storage quota was exceeded.
	ErrQuotaExceeded = errors.New("storage: quota exceeded")

	// ErrProviderUnavailable indicates the storage backend is unreachable.
	ErrProviderUnavailable = errors.New("storage: provider unavailable")

	// ErrInvalidKey indicates the storage key is malformed.
	ErrInvalidKey = errors.New("storage: invalid key")

	// ErrNotSupported indicates the operation is not supported by this provider.
	ErrNotSupported = errors.New("storage: operation not supported")

	// ErrChecksumMismatch indicates data corruption was detected.
	ErrChecksumMismatch = errors.New("storage: checksum mismatch")

	// ErrSizeMismatch indicates the actual size differs from the declared size.
	ErrSizeMismatch = errors.New("storage: size mismatch")

	// ErrPathTraversal indicates a path traversal attempt was detected.
	ErrPathTraversal = errors.New("storage: path traversal detected")
)

// IsRetryable returns true if the error is transient and the operation can be retried.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrProviderUnavailable)
}

// IsNotFound returns true if the error indicates the object was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAccessDenied returns true if the error indicates access was denied.
func IsAccessDenied(err error) bool {
	return errors.Is(err, ErrAccessDenied)
}

// WrapProviderError wraps provider-specific errors with context.
func WrapProviderError(provider string, operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", provider, operation, err)
}
