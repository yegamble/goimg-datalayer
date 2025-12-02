package moderation

import (
	"context"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ReportRepository defines the interface for persisting Report aggregates.
// Implementations must be provided by the infrastructure layer.
type ReportRepository interface {
	// NextID generates the next available ReportID.
	// This is used by the application layer when creating new reports.
	NextID() ReportID

	// FindByID retrieves a report by its unique identifier.
	// Returns ErrReportNotFound if the report does not exist.
	FindByID(ctx context.Context, id ReportID) (*Report, error)

	// FindPending retrieves all reports with pending status.
	// Results are paginated according to the provided pagination parameters.
	// Returns the reports, total count, and any error.
	FindPending(ctx context.Context, pagination shared.Pagination) ([]*Report, int64, error)

	// FindByImage retrieves all reports for a specific image.
	// This is useful for checking if an image has been reported multiple times.
	FindByImage(ctx context.Context, imageID gallery.ImageID) ([]*Report, error)

	// FindByReporter retrieves all reports submitted by a specific user.
	// Results are paginated according to the provided pagination parameters.
	// Returns the reports, total count, and any error.
	FindByReporter(ctx context.Context, reporterID identity.UserID, pagination shared.Pagination) ([]*Report, int64, error)

	// Save persists a report to the data store.
	// This handles both creation and updates.
	// Domain events should be published after successful save.
	Save(ctx context.Context, report *Report) error
}

// BanRepository defines the interface for persisting Ban aggregates.
// Implementations must be provided by the infrastructure layer.
type BanRepository interface {
	// NextID generates the next available BanID.
	// This is used by the application layer when creating new bans.
	NextID() BanID

	// FindByID retrieves a ban by its unique identifier.
	// Returns ErrBanNotFound if the ban does not exist.
	FindByID(ctx context.Context, id BanID) (*Ban, error)

	// FindByUserID retrieves the most recent ban for a specific user.
	// Returns ErrBanNotFound if the user has no bans.
	// Note: A user can only have one active ban at a time, but may have historical bans.
	FindByUserID(ctx context.Context, userID identity.UserID) (*Ban, error)

	// FindActiveBans retrieves all currently active bans.
	// This includes permanent bans and temporary bans that have not expired or been revoked.
	FindActiveBans(ctx context.Context) ([]*Ban, error)

	// FindExpiredBans retrieves all bans that have naturally expired.
	// This is useful for cleanup and notification purposes.
	FindExpiredBans(ctx context.Context) ([]*Ban, error)

	// IsUserBanned checks if a user currently has an active ban.
	// This is a convenience method for authorization checks.
	IsUserBanned(ctx context.Context, userID identity.UserID) (bool, error)

	// Save persists a ban to the data store.
	// This handles both creation and updates.
	// Domain events should be published after successful save.
	Save(ctx context.Context, ban *Ban) error
}

// ReviewRepository defines the interface for persisting Review entities.
// Implementations must be provided by the infrastructure layer.
type ReviewRepository interface {
	// NextID generates the next available ReviewID.
	// This is used by the application layer when creating new reviews.
	NextID() ReviewID

	// FindByID retrieves a review by its unique identifier.
	// Returns ErrReviewNotFound if the review does not exist.
	FindByID(ctx context.Context, id ReviewID) (*Review, error)

	// FindByReport retrieves all reviews for a specific report.
	// Reviews are returned in chronological order (oldest first).
	// This provides a complete audit trail for a report.
	FindByReport(ctx context.Context, reportID ReportID) ([]*Review, error)

	// FindByReviewer retrieves all reviews performed by a specific moderator.
	// Results are paginated according to the provided pagination parameters.
	// Returns the reviews, total count, and any error.
	FindByReviewer(ctx context.Context, reviewerID identity.UserID, pagination shared.Pagination) ([]*Review, int64, error)

	// Save persists a review to the data store.
	// Reviews are immutable, so this only handles creation.
	Save(ctx context.Context, review *Review) error
}
