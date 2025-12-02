package gallery

import "fmt"

// ImageStatus represents the current state of an image in its lifecycle.
type ImageStatus string

const (
	// StatusProcessing indicates the image is being uploaded and processed.
	// Variants are being generated and the image is not yet publicly accessible.
	StatusProcessing ImageStatus = "processing"

	// StatusActive indicates the image has been successfully processed and is accessible
	// according to its visibility settings.
	StatusActive ImageStatus = "active"

	// StatusDeleted indicates the image has been soft-deleted by the owner.
	// The image is not accessible but may still exist in storage for recovery.
	StatusDeleted ImageStatus = "deleted"

	// StatusFlagged indicates the image has been flagged for moderation review.
	// It may have restricted visibility until reviewed.
	StatusFlagged ImageStatus = "flagged"
)

// AllImageStatuses returns all valid image status values.
func AllImageStatuses() []ImageStatus {
	return []ImageStatus{
		StatusProcessing,
		StatusActive,
		StatusDeleted,
		StatusFlagged,
	}
}

// ParseImageStatus parses a string into an ImageStatus.
// Returns an error if the string is not a valid status value.
func ParseImageStatus(s string) (ImageStatus, error) {
	status := ImageStatus(s)
	switch status {
	case StatusProcessing, StatusActive, StatusDeleted, StatusFlagged:
		return status, nil
	default:
		return "", fmt.Errorf("%w: invalid image status '%s'", ErrInvalidImageStatus, s)
	}
}

// IsValid returns true if this is a valid image status.
func (s ImageStatus) IsValid() bool {
	switch s {
	case StatusProcessing, StatusActive, StatusDeleted, StatusFlagged:
		return true
	default:
		return false
	}
}

// IsViewable returns true if images with this status can be viewed.
// Only active images are viewable; processing, deleted, and flagged images are not.
func (s ImageStatus) IsViewable() bool {
	return s == StatusActive
}

// IsDeleted returns true if this status represents a deleted image.
func (s ImageStatus) IsDeleted() bool {
	return s == StatusDeleted
}

// IsFlagged returns true if this status represents a flagged image.
func (s ImageStatus) IsFlagged() bool {
	return s == StatusFlagged
}

// String returns the string representation of the status.
func (s ImageStatus) String() string {
	return string(s)
}
