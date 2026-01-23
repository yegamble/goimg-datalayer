package gallery

import (
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ScanStatus represents the malware scan status of an image.
type ScanStatus string

const (
	// ScanStatusPending indicates the image is waiting to be scanned.
	ScanStatusPending ScanStatus = "pending"

	// ScanStatusClean indicates the image was scanned and no malware was found.
	ScanStatusClean ScanStatus = "clean"

	// ScanStatusInfected indicates the image was scanned and malware was found.
	ScanStatusInfected ScanStatus = "infected"

	// ScanStatusError indicates the scan failed due to a system error.
	ScanStatusError ScanStatus = "error"
)

	// ScanStatusInfected indicates the image was scanned and malware was detected.
	ScanStatusInfected ScanStatus = "infected"

	// ScanStatusError indicates the scan failed due to a technical error.
	ScanStatusError ScanStatus = "error"
)

// AllScanStatuses returns all valid scan status values.
func AllScanStatuses() []ScanStatus {
	return []ScanStatus{
		ScanStatusPending,
		ScanStatusClean,
		ScanStatusInfected,
		ScanStatusError,
	}
}

// ParseScanStatus parses a string into a ScanStatus.
// Returns an error if the string is not a valid scan status value.
func ParseScanStatus(s string) (ScanStatus, error) {
	status := ScanStatus(s)
	switch status {
	case ScanStatusPending, ScanStatusClean, ScanStatusInfected, ScanStatusError:
		return status, nil
	default:
		return "", fmt.Errorf("%w: invalid scan status '%s'", shared.ErrInvalidInput, s)
	}
}

// IsValid returns true if this is a valid scan status.
func (s ScanStatus) IsValid() bool {
	switch s {
	case ScanStatusPending, ScanStatusClean, ScanStatusInfected, ScanStatusError:
		return true
	default:
		return false
	}
}

// String returns the string representation of the scan status.
func (s ScanStatus) String() string {
	return string(s)
}
