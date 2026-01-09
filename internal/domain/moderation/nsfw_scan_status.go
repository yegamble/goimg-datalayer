package moderation

import "fmt"

// NSFWScanStatus represents the current status of an NSFW scan.
type NSFWScanStatus string

const (
	// ScanStatusPending indicates the scan is queued but not started.
	ScanStatusPending NSFWScanStatus = "pending"
	// ScanStatusScanning indicates the scan is currently in progress.
	ScanStatusScanning NSFWScanStatus = "scanning"
	// ScanStatusCompleted indicates the scan completed successfully.
	ScanStatusCompleted NSFWScanStatus = "completed"
	// ScanStatusFailed indicates the scan failed (API error, timeout, etc.).
	ScanStatusFailed NSFWScanStatus = "failed"
)

// ParseNSFWScanStatus creates an NSFWScanStatus from a string value.
// Returns an error if the string is not a valid status.
func ParseNSFWScanStatus(s string) (NSFWScanStatus, error) {
	status := NSFWScanStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidNSFWScanStatus, s)
	}
	return status, nil
}

// String returns the string representation of the NSFWScanStatus.
func (s NSFWScanStatus) String() string {
	return string(s)
}

// IsValid returns true if the NSFWScanStatus is a valid status value.
func (s NSFWScanStatus) IsValid() bool {
	switch s {
	case ScanStatusPending, ScanStatusScanning, ScanStatusCompleted, ScanStatusFailed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status is a terminal state.
// Terminal states (completed, failed) cannot transition to other states.
func (s NSFWScanStatus) IsTerminal() bool {
	return s == ScanStatusCompleted || s == ScanStatusFailed
}

// IsActive returns true if the scan is in an active state (pending or scanning).
func (s NSFWScanStatus) IsActive() bool {
	return s == ScanStatusPending || s == ScanStatusScanning
}
