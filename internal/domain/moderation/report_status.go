package moderation

import "fmt"

// ReportStatus represents the current status of a content report.
type ReportStatus string

const (
	// StatusPending indicates the report is awaiting review.
	StatusPending ReportStatus = "pending"
	// StatusReviewing indicates the report is being actively reviewed.
	StatusReviewing ReportStatus = "reviewing"
	// StatusResolved indicates the report was resolved with action taken.
	StatusResolved ReportStatus = "resolved"
	// StatusDismissed indicates the report was dismissed as invalid.
	StatusDismissed ReportStatus = "dismissed"
)

// ParseReportStatus creates a ReportStatus from a string value.
// Returns an error if the string is not a valid report status.
func ParseReportStatus(s string) (ReportStatus, error) {
	status := ReportStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidReportStatus, s)
	}
	return status, nil
}

// String returns the string representation of the ReportStatus.
func (s ReportStatus) String() string {
	return string(s)
}

// IsValid returns true if the ReportStatus is a valid status value.
func (s ReportStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusReviewing, StatusResolved, StatusDismissed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the ReportStatus is a terminal state.
// Terminal states (resolved, dismissed) cannot transition to other states.
func (s ReportStatus) IsTerminal() bool {
	return s == StatusResolved || s == StatusDismissed
}
