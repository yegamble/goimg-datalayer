package ipfs

import "errors"

// IPFS-specific error types.
// Use errors.Is() to check for these error types.
var (
	// ErrNotFound indicates the content is not available on the IPFS network.
	ErrNotFound = errors.New("ipfs: content not found")

	// ErrPinFailed indicates the pinning operation failed.
	ErrPinFailed = errors.New("ipfs: pin failed")

	// ErrUnpinFailed indicates the unpinning operation failed.
	ErrUnpinFailed = errors.New("ipfs: unpin failed")

	// ErrNodeUnavailable indicates the IPFS node is not reachable.
	ErrNodeUnavailable = errors.New("ipfs: node unavailable")

	// ErrInvalidCID indicates an invalid content identifier was provided.
	ErrInvalidCID = errors.New("ipfs: invalid CID")

	// ErrTimeout indicates the operation exceeded the configured timeout.
	ErrTimeout = errors.New("ipfs: operation timed out")

	// ErrConfigInvalid indicates the configuration is missing required values.
	ErrConfigInvalid = errors.New("ipfs: API endpoint required")

	// ErrUploadFailed indicates the upload operation failed.
	ErrUploadFailed = errors.New("ipfs: upload failed")

	// ErrNotSupported indicates the operation is not supported by this provider.
	ErrNotSupported = errors.New("ipfs: operation not supported")
)

// IsRetryable returns true if the error is transient and can be retried.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrNodeUnavailable) || errors.Is(err, ErrTimeout)
}

// IsNotFound returns true if the error indicates content was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
