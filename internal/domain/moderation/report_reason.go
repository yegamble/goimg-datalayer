package moderation

import "fmt"

// ReportReason represents the reason for reporting content.
type ReportReason string

const (
	// ReasonSpam indicates the content is spam or advertisement.
	ReasonSpam ReportReason = "spam"
	// ReasonInappropriate indicates the content is inappropriate or offensive.
	ReasonInappropriate ReportReason = "inappropriate"
	// ReasonCopyright indicates a copyright violation.
	ReasonCopyright ReportReason = "copyright"
	// ReasonHarassment indicates the content contains harassment or bullying.
	ReasonHarassment ReportReason = "harassment"
	// ReasonOther indicates another reason (requires description).
	ReasonOther ReportReason = "other"
)

// ParseReportReason creates a ReportReason from a string value.
// Returns an error if the string is not a valid report reason.
func ParseReportReason(s string) (ReportReason, error) {
	reason := ReportReason(s)
	if !reason.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidReportReason, s)
	}
	return reason, nil
}

// String returns the string representation of the ReportReason.
func (r ReportReason) String() string {
	return string(r)
}

// IsValid returns true if the ReportReason is a valid reason value.
func (r ReportReason) IsValid() bool {
	switch r {
	case ReasonSpam, ReasonInappropriate, ReasonCopyright, ReasonHarassment, ReasonOther:
		return true
	default:
		return false
	}
}
