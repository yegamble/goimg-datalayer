package moderation

import "errors"

// Domain-specific errors for the Moderation bounded context.
var (
	// ErrReportNotFound indicates a report was not found.
	ErrReportNotFound = errors.New("report not found")
	// ErrReportAlreadyResolved indicates a report has already been resolved.
	ErrReportAlreadyResolved = errors.New("report already resolved")
	// ErrReportAlreadyDismissed indicates a report has already been dismissed.
	ErrReportAlreadyDismissed = errors.New("report already dismissed")
	// ErrReportInTerminalState indicates a report is in a terminal state and cannot be modified.
	ErrReportInTerminalState = errors.New("report is in terminal state")
	// ErrReportNotReviewing indicates the report must be in reviewing state for this operation.
	ErrReportNotReviewing = errors.New("report is not in reviewing state")

	// ErrBanNotFound indicates a ban was not found.
	ErrBanNotFound = errors.New("ban not found")
	// ErrUserAlreadyBanned indicates a user already has an active ban.
	ErrUserAlreadyBanned = errors.New("user is already banned")
	// ErrBanAlreadyRevoked indicates a ban has already been revoked.
	ErrBanAlreadyRevoked = errors.New("ban already revoked")
	// ErrBanExpired indicates a ban has already expired.
	ErrBanExpired = errors.New("ban has expired")
	// ErrBanNotActive indicates the ban is not currently active.
	ErrBanNotActive = errors.New("ban is not active")

	// ErrReviewNotFound indicates a review was not found.
	ErrReviewNotFound = errors.New("review not found")

	// ErrInvalidReportReason indicates an invalid report reason was provided.
	ErrInvalidReportReason = errors.New("invalid report reason")
	// ErrInvalidReportStatus indicates an invalid report status was provided.
	ErrInvalidReportStatus = errors.New("invalid report status")
	// ErrInvalidReviewAction indicates an invalid review action was provided.
	ErrInvalidReviewAction = errors.New("invalid review action")

	// ErrDescriptionEmpty indicates the description cannot be empty.
	ErrDescriptionEmpty = errors.New("description cannot be empty")
	// ErrDescriptionTooLong indicates the description exceeds the maximum length.
	ErrDescriptionTooLong = errors.New("description exceeds 1000 characters")
	// ErrReasonRequired indicates a ban reason is required.
	ErrReasonRequired = errors.New("ban reason is required")
	// ErrReasonTooLong indicates the reason exceeds the maximum length.
	ErrReasonTooLong = errors.New("reason exceeds 500 characters")
	// ErrResolutionRequired indicates a resolution is required when resolving a report.
	ErrResolutionRequired = errors.New("resolution is required")
	// ErrResolutionTooLong indicates the resolution exceeds the maximum length.
	ErrResolutionTooLong = errors.New("resolution exceeds 1000 characters")
	// ErrNotesTooLong indicates review notes exceed the maximum length.
	ErrNotesTooLong = errors.New("notes exceed 2000 characters")
)
