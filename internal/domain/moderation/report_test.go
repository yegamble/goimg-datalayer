package moderation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestNewReport(t *testing.T) {
	t.Parallel()

	reporterID := identity.NewUserID()
	imageID := gallery.NewImageID()
	reason := moderation.ReasonSpam
	description := "This image is spam"

	t.Run("valid report", func(t *testing.T) {
		t.Parallel()

		report, err := moderation.NewReport(reporterID, imageID, reason, description)

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.False(t, report.ID().IsZero())
		assert.Equal(t, reporterID, report.ReporterID())
		assert.Equal(t, imageID, report.ImageID())
		assert.Equal(t, reason, report.Reason())
		assert.Equal(t, description, report.Description())
		assert.Equal(t, moderation.StatusPending, report.Status())
		assert.Nil(t, report.ResolvedBy())
		assert.Nil(t, report.ResolvedAt())
		assert.Empty(t, report.Resolution())
		assert.False(t, report.CreatedAt().IsZero())

		// Should emit ReportCreated event
		events := report.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.report.created", events[0].EventType())
	})

	t.Run("empty reporter id", func(t *testing.T) {
		t.Parallel()

		var emptyReporterID identity.UserID
		_, err := moderation.NewReport(emptyReporterID, imageID, reason, description)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reporter id is required")
	})

	t.Run("empty image id", func(t *testing.T) {
		t.Parallel()

		var emptyImageID gallery.ImageID
		_, err := moderation.NewReport(reporterID, emptyImageID, reason, description)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "image id is required")
	})

	t.Run("invalid reason", func(t *testing.T) {
		t.Parallel()

		invalidReason := moderation.ReportReason("invalid")
		_, err := moderation.NewReport(reporterID, imageID, invalidReason, description)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrInvalidReportReason)
	})

	t.Run("empty description", func(t *testing.T) {
		t.Parallel()

		_, err := moderation.NewReport(reporterID, imageID, reason, "")

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrDescriptionEmpty)
	})

	t.Run("description too long", func(t *testing.T) {
		t.Parallel()

		longDescription := strings.Repeat("a", 1001)
		_, err := moderation.NewReport(reporterID, imageID, reason, longDescription)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrDescriptionTooLong)
	})

	t.Run("description at max length", func(t *testing.T) {
		t.Parallel()

		maxDescription := strings.Repeat("a", 1000)
		report, err := moderation.NewReport(reporterID, imageID, reason, maxDescription)

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, maxDescription, report.Description())
	})
}

func TestReconstructReport(t *testing.T) {
	t.Parallel()

	id := moderation.NewReportID()
	reporterID := identity.NewUserID()
	imageID := gallery.NewImageID()
	reason := moderation.ReasonSpam
	description := "This is spam"
	status := moderation.StatusResolved
	resolverID := identity.NewUserID()
	resolvedAt := time.Now().UTC()
	resolution := "Content removed"
	createdAt := time.Now().Add(-24 * time.Hour).UTC()

	report := moderation.ReconstructReport(
		id,
		reporterID,
		imageID,
		reason,
		description,
		status,
		&resolverID,
		&resolvedAt,
		resolution,
		createdAt,
	)

	assert.NotNil(t, report)
	assert.Equal(t, id, report.ID())
	assert.Equal(t, reporterID, report.ReporterID())
	assert.Equal(t, imageID, report.ImageID())
	assert.Equal(t, reason, report.Reason())
	assert.Equal(t, description, report.Description())
	assert.Equal(t, status, report.Status())
	assert.Equal(t, &resolverID, report.ResolvedBy())
	assert.Equal(t, &resolvedAt, report.ResolvedAt())
	assert.Equal(t, resolution, report.Resolution())
	assert.Equal(t, createdAt, report.CreatedAt())

	// Reconstruction should not emit events
	assert.Empty(t, report.Events())
}

//nolint:funlen // Table-driven test with comprehensive test cases
func TestReport_StartReview(t *testing.T) {
	t.Parallel()

	t.Run("pending to reviewing", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		report.ClearEvents() // Clear creation event

		err := report.StartReview()

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusReviewing, report.Status())

		// Should emit ReportReviewStarted event
		events := report.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.report.review_started", events[0].EventType())
	})

	t.Run("already reviewing is no-op", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		report.ClearEvents()

		err := report.StartReview()

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusReviewing, report.Status())
		assert.Empty(t, report.Events())
	})

	t.Run("cannot start review on resolved report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Resolve(resolverID, "Resolved"))
		report.ClearEvents()

		err := report.StartReview()

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReportInTerminalState)
		assert.Equal(t, moderation.StatusResolved, report.Status())
	})

	t.Run("cannot start review on dismissed report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Dismiss(resolverID))
		report.ClearEvents()

		err := report.StartReview()

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReportInTerminalState)
		assert.Equal(t, moderation.StatusDismissed, report.Status())
	})
	//nolint:funlen // Table-driven test with comprehensive test cases
}

func TestReport_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("resolve report with valid resolution", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		report.ClearEvents()

		resolverID := identity.NewUserID()
		resolution := "Content removed for violating guidelines"
		beforeResolve := time.Now().UTC()

		err := report.Resolve(resolverID, resolution)

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusResolved, report.Status())
		assert.NotNil(t, report.ResolvedBy())
		assert.Equal(t, resolverID, *report.ResolvedBy())
		assert.NotNil(t, report.ResolvedAt())
		assert.True(t, report.ResolvedAt().After(beforeResolve) || report.ResolvedAt().Equal(beforeResolve))
		assert.Equal(t, resolution, report.Resolution())

		// Should emit ReportResolved event
		events := report.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.report.resolved", events[0].EventType())
	})

	t.Run("can resolve from pending status", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		report.ClearEvents()

		resolverID := identity.NewUserID()
		resolution := "Quick resolution"

		err := report.Resolve(resolverID, resolution)

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusResolved, report.Status())
	})

	t.Run("empty resolver id", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())

		var emptyResolverID identity.UserID
		err := report.Resolve(emptyResolverID, "Resolution")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolver id is required")
		assert.Equal(t, moderation.StatusReviewing, report.Status())
	})

	t.Run("empty resolution", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())

		resolverID := identity.NewUserID()
		err := report.Resolve(resolverID, "")

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrResolutionRequired)
		assert.Equal(t, moderation.StatusReviewing, report.Status())
	})

	t.Run("resolution too long", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())

		resolverID := identity.NewUserID()
		longResolution := strings.Repeat("a", 1001)
		err := report.Resolve(resolverID, longResolution)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrResolutionTooLong)
		assert.Equal(t, moderation.StatusReviewing, report.Status())
	})

	t.Run("resolution at max length", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())

		resolverID := identity.NewUserID()
		maxResolution := strings.Repeat("a", 1000)
		err := report.Resolve(resolverID, maxResolution)

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusResolved, report.Status())
	})

	t.Run("cannot resolve already resolved report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Resolve(resolverID, "First resolution"))
		report.ClearEvents()

		err := report.Resolve(resolverID, "Second resolution")

		require.Error(t, err)
		require.ErrorIs(t, err, moderation.ErrReportAlreadyResolved)
		assert.Equal(t, moderation.StatusResolved, report.Status())
		assert.Equal(t, "First resolution", report.Resolution())
	})

	t.Run("cannot resolve dismissed report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Dismiss(resolverID))
		report.ClearEvents()

		err := report.Resolve(resolverID, "Resolution")

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReportInTerminalState)
		assert.Equal(t, moderation.StatusDismissed, report.Status())
	})
}

func TestReport_Dismiss(t *testing.T) {
	t.Parallel()

	t.Run("dismiss report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		report.ClearEvents()

		resolverID := identity.NewUserID()
		beforeDismiss := time.Now().UTC()

		err := report.Dismiss(resolverID)

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusDismissed, report.Status())
		assert.NotNil(t, report.ResolvedBy())
		assert.Equal(t, resolverID, *report.ResolvedBy())
		assert.NotNil(t, report.ResolvedAt())
		assert.True(t, report.ResolvedAt().After(beforeDismiss) || report.ResolvedAt().Equal(beforeDismiss))
		assert.Equal(t, "Report dismissed", report.Resolution())

		// Should emit ReportDismissed event
		events := report.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.report.dismissed", events[0].EventType())
	})

	t.Run("can dismiss from pending status", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		report.ClearEvents()

		resolverID := identity.NewUserID()

		err := report.Dismiss(resolverID)

		require.NoError(t, err)
		assert.Equal(t, moderation.StatusDismissed, report.Status())
	})

	t.Run("empty resolver id", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())

		var emptyResolverID identity.UserID
		err := report.Dismiss(emptyResolverID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolver id is required")
		assert.Equal(t, moderation.StatusReviewing, report.Status())
	})

	t.Run("cannot dismiss already dismissed report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Dismiss(resolverID))
		report.ClearEvents()

		err := report.Dismiss(resolverID)

		require.ErrorIs(t, err, moderation.ErrReportAlreadyDismissed)
		assert.Equal(t, moderation.StatusDismissed, report.Status())
	})

	t.Run("cannot dismiss resolved report", func(t *testing.T) {
		t.Parallel()

		report := createTestReport(t)
		require.NoError(t, report.StartReview())
		resolverID := identity.NewUserID()
		require.NoError(t, report.Resolve(resolverID, "Resolution"))
		report.ClearEvents()

		err := report.Dismiss(resolverID)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReportInTerminalState)
		assert.Equal(t, moderation.StatusResolved, report.Status())
	})
}

func TestReport_ClearEvents(t *testing.T) {
	t.Parallel()

	report := createTestReport(t)
	assert.NotEmpty(t, report.Events())

	report.ClearEvents()

	assert.Empty(t, report.Events())
}

// Helper function to create a test report.
func createTestReport(t *testing.T) *moderation.Report {
	t.Helper()

	reporterID := identity.NewUserID()
	imageID := gallery.NewImageID()
	reason := moderation.ReasonSpam
	description := "This is a test report"

	report, err := moderation.NewReport(reporterID, imageID, reason, description)
	require.NoError(t, err)

	return report
}
