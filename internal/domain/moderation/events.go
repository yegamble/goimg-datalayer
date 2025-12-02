package moderation

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ReportCreated is emitted when a new content report is created.
type ReportCreated struct {
	shared.BaseEvent
	ReportID    ReportID
	ReporterID  identity.UserID
	ImageID     gallery.ImageID
	Reason      ReportReason
	Description string
}

// NewReportCreated creates a new ReportCreated event.
func NewReportCreated(
	reportID ReportID,
	reporterID identity.UserID,
	imageID gallery.ImageID,
	reason ReportReason,
	description string,
) ReportCreated {
	return ReportCreated{
		BaseEvent:   shared.NewBaseEvent("moderation.report.created", reportID.String()),
		ReportID:    reportID,
		ReporterID:  reporterID,
		ImageID:     imageID,
		Reason:      reason,
		Description: description,
	}
}

// ReportReviewStarted is emitted when a moderator starts reviewing a report.
type ReportReviewStarted struct {
	shared.BaseEvent
	ReportID ReportID
}

// NewReportReviewStarted creates a new ReportReviewStarted event.
func NewReportReviewStarted(reportID ReportID) ReportReviewStarted {
	return ReportReviewStarted{
		BaseEvent: shared.NewBaseEvent("moderation.report.review_started", reportID.String()),
		ReportID:  reportID,
	}
}

// ReportResolved is emitted when a report is resolved with action taken.
type ReportResolved struct {
	shared.BaseEvent
	ReportID   ReportID
	ResolverID identity.UserID
	Resolution string
}

// NewReportResolved creates a new ReportResolved event.
func NewReportResolved(reportID ReportID, resolverID identity.UserID, resolution string) ReportResolved {
	return ReportResolved{
		BaseEvent:  shared.NewBaseEvent("moderation.report.resolved", reportID.String()),
		ReportID:   reportID,
		ResolverID: resolverID,
		Resolution: resolution,
	}
}

// ReportDismissed is emitted when a report is dismissed as invalid.
type ReportDismissed struct {
	shared.BaseEvent
	ReportID   ReportID
	ResolverID identity.UserID
}

// NewReportDismissed creates a new ReportDismissed event.
func NewReportDismissed(reportID ReportID, resolverID identity.UserID) ReportDismissed {
	return ReportDismissed{
		BaseEvent:  shared.NewBaseEvent("moderation.report.dismissed", reportID.String()),
		ReportID:   reportID,
		ResolverID: resolverID,
	}
}

// UserBanned is emitted when a user is banned.
type UserBanned struct {
	shared.BaseEvent
	BanID       BanID
	UserID      identity.UserID
	BannedBy    identity.UserID
	Reason      string
	IsPermanent bool
}

// NewUserBanned creates a new UserBanned event.
func NewUserBanned(
	banID BanID,
	userID identity.UserID,
	bannedBy identity.UserID,
	reason string,
	isPermanent bool,
) UserBanned {
	return UserBanned{
		BaseEvent:   shared.NewBaseEvent("moderation.user.banned", banID.String()),
		BanID:       banID,
		UserID:      userID,
		BannedBy:    bannedBy,
		Reason:      reason,
		IsPermanent: isPermanent,
	}
}

// BanRevoked is emitted when a ban is manually revoked before expiration.
type BanRevoked struct {
	shared.BaseEvent
	BanID     BanID
	UserID    identity.UserID
	RevokedBy identity.UserID
}

// NewBanRevoked creates a new BanRevoked event.
func NewBanRevoked(banID BanID, userID identity.UserID, revokedBy identity.UserID) BanRevoked {
	return BanRevoked{
		BaseEvent: shared.NewBaseEvent("moderation.ban.revoked", banID.String()),
		BanID:     banID,
		UserID:    userID,
		RevokedBy: revokedBy,
	}
}

// BanExpired is emitted when a temporary ban expires naturally.
type BanExpired struct {
	shared.BaseEvent
	BanID  BanID
	UserID identity.UserID
}

// NewBanExpired creates a new BanExpired event.
func NewBanExpired(banID BanID, userID identity.UserID) BanExpired {
	return BanExpired{
		BaseEvent: shared.NewBaseEvent("moderation.ban.expired", banID.String()),
		BanID:     banID,
		UserID:    userID,
	}
}
