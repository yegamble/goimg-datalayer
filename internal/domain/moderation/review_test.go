package moderation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewReview(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	reviewerID := identity.NewUserID()
	action := moderation.ActionRemove
	notes := "Content violates community guidelines"

	t.Run("valid review", func(t *testing.T) {
		t.Parallel()

		review, err := moderation.NewReview(reportID, reviewerID, action, notes)

		require.NoError(t, err)
		assert.NotNil(t, review)
		assert.False(t, review.ID().IsZero())
		assert.Equal(t, reportID, review.ReportID())
		assert.Equal(t, reviewerID, review.ReviewerID())
		assert.Equal(t, action, review.Action())
		assert.Equal(t, notes, review.Notes())
		assert.False(t, review.CreatedAt().IsZero())
	})

	t.Run("review with empty notes", func(t *testing.T) {
		t.Parallel()

		review, err := moderation.NewReview(reportID, reviewerID, action, "")

		require.NoError(t, err)
		assert.NotNil(t, review)
		assert.Empty(t, review.Notes())
	})

	t.Run("empty report id", func(t *testing.T) {
		t.Parallel()

		var emptyReportID moderation.ReportID
		_, err := moderation.NewReview(emptyReportID, reviewerID, action, notes)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "report id is required")
	})

	t.Run("empty reviewer id", func(t *testing.T) {
		t.Parallel()

		var emptyReviewerID identity.UserID
		_, err := moderation.NewReview(reportID, emptyReviewerID, action, notes)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reviewer id is required")
	})

	t.Run("invalid action", func(t *testing.T) {
		t.Parallel()

		invalidAction := moderation.ReviewAction("invalid")
		_, err := moderation.NewReview(reportID, reviewerID, invalidAction, notes)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrInvalidReviewAction)
	})

	t.Run("notes too long", func(t *testing.T) {
		t.Parallel()

		longNotes := strings.Repeat("a", 2001)
		_, err := moderation.NewReview(reportID, reviewerID, action, longNotes)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrNotesTooLong)
	})

	t.Run("notes at max length", func(t *testing.T) {
		t.Parallel()

		maxNotes := strings.Repeat("a", 2000)
		review, err := moderation.NewReview(reportID, reviewerID, action, maxNotes)

		require.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, maxNotes, review.Notes())
	})
}

func TestReconstructReview(t *testing.T) {
	t.Parallel()

	id := moderation.NewReviewID()
	reportID := moderation.NewReportID()
	reviewerID := identity.NewUserID()
	action := moderation.ActionBan
	notes := "User banned for repeated violations"
	createdAt := time.Now().Add(-1 * time.Hour).UTC()

	review := moderation.ReconstructReview(
		id,
		reportID,
		reviewerID,
		action,
		notes,
		createdAt,
	)

	assert.NotNil(t, review)
	assert.Equal(t, id, review.ID())
	assert.Equal(t, reportID, review.ReportID())
	assert.Equal(t, reviewerID, review.ReviewerID())
	assert.Equal(t, action, review.Action())
	assert.Equal(t, notes, review.Notes())
	assert.Equal(t, createdAt, review.CreatedAt())
}

func TestReview_Getters(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	reviewerID := identity.NewUserID()
	action := moderation.ActionWarn
	notes := "First warning issued"

	review, err := moderation.NewReview(reportID, reviewerID, action, notes)
	require.NoError(t, err)

	t.Run("ID", func(t *testing.T) {
		t.Parallel()
		assert.False(t, review.ID().IsZero())
	})

	t.Run("ReportID", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, reportID, review.ReportID())
	})

	t.Run("ReviewerID", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, reviewerID, review.ReviewerID())
	})

	t.Run("Action", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, action, review.Action())
	})

	t.Run("Notes", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, notes, review.Notes())
	})

	t.Run("CreatedAt", func(t *testing.T) {
		t.Parallel()
		assert.False(t, review.CreatedAt().IsZero())
	})
}

func TestReview_AllActions(t *testing.T) {
	t.Parallel()

	reportID := moderation.NewReportID()
	reviewerID := identity.NewUserID()

	actions := []moderation.ReviewAction{
		moderation.ActionDismiss,
		moderation.ActionWarn,
		moderation.ActionRemove,
		moderation.ActionBan,
	}

	for _, action := range actions {
		action := action
		t.Run(action.String(), func(t *testing.T) {
			t.Parallel()

			review, err := moderation.NewReview(reportID, reviewerID, action, "Test notes")

			require.NoError(t, err)
			assert.Equal(t, action, review.Action())
		})
	}
}
