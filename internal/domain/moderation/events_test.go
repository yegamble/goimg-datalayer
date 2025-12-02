package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewReportCreated(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	reporterID := identity.NewUserID()
	imageID := gallery.NewImageID()
	reason := moderation.ReasonSpam
	description := "This is spam"

	event := moderation.NewReportCreated(reportID, reporterID, imageID, reason, description)

	assert.Equal(t, "moderation.report.created", event.EventType())
	assert.Equal(t, reportID.String(), event.AggregateID())
	assert.Equal(t, reportID, event.ReportID)
	assert.Equal(t, reporterID, event.ReporterID)
	assert.Equal(t, imageID, event.ImageID)
	assert.Equal(t, reason, event.Reason)
	assert.Equal(t, description, event.Description)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewReportReviewStarted(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()

	event := moderation.NewReportReviewStarted(reportID)

	assert.Equal(t, "moderation.report.review_started", event.EventType())
	assert.Equal(t, reportID.String(), event.AggregateID())
	assert.Equal(t, reportID, event.ReportID)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewReportResolved(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	resolverID := identity.NewUserID()
	resolution := "Content removed"

	event := moderation.NewReportResolved(reportID, resolverID, resolution)

	assert.Equal(t, "moderation.report.resolved", event.EventType())
	assert.Equal(t, reportID.String(), event.AggregateID())
	assert.Equal(t, reportID, event.ReportID)
	assert.Equal(t, resolverID, event.ResolverID)
	assert.Equal(t, resolution, event.Resolution)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewReportDismissed(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	resolverID := identity.NewUserID()

	event := moderation.NewReportDismissed(reportID, resolverID)

	assert.Equal(t, "moderation.report.dismissed", event.EventType())
	assert.Equal(t, reportID.String(), event.AggregateID())
	assert.Equal(t, reportID, event.ReportID)
	assert.Equal(t, resolverID, event.ResolverID)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewUserBanned(t *testing.T) {
	t.Parallel()

	banID := moderation.NewBanID()
	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Repeated violations"
	isPermanent := true

	event := moderation.NewUserBanned(banID, userID, bannedBy, reason, isPermanent)

	assert.Equal(t, "moderation.user.banned", event.EventType())
	assert.Equal(t, banID.String(), event.AggregateID())
	assert.Equal(t, banID, event.BanID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, bannedBy, event.BannedBy)
	assert.Equal(t, reason, event.Reason)
	assert.Equal(t, isPermanent, event.IsPermanent)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewBanRevoked(t *testing.T) {
	t.Parallel()

	banID := moderation.NewBanID()
	userID := identity.NewUserID()
	revokedBy := identity.NewUserID()

	event := moderation.NewBanRevoked(banID, userID, revokedBy)

	assert.Equal(t, "moderation.ban.revoked", event.EventType())
	assert.Equal(t, banID.String(), event.AggregateID())
	assert.Equal(t, banID, event.BanID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, revokedBy, event.RevokedBy)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewBanExpired(t *testing.T) {
	t.Parallel()

	banID := moderation.NewBanID()
	userID := identity.NewUserID()

	event := moderation.NewBanExpired(banID, userID)

	assert.Equal(t, "moderation.ban.expired", event.EventType())
	assert.Equal(t, banID.String(), event.AggregateID())
	assert.Equal(t, banID, event.BanID)
	assert.Equal(t, userID, event.UserID)
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}
