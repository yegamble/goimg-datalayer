package moderation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewNSFWScan(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	provider := moderation.ProviderSightEngine

	scan := moderation.NewNSFWScan(scanID, imageID, provider)

	assert.Equal(t, scanID, scan.ID())
	assert.Equal(t, imageID, scan.ImageID())
	assert.Equal(t, provider, scan.Provider())
	assert.Equal(t, moderation.ScanStatusPending, scan.Status())
	assert.Equal(t, moderation.CategoryUnknown, scan.Category())
	assert.Equal(t, 0.0, scan.Score())
	assert.True(t, scan.Details().IsEmpty())
	assert.Empty(t, scan.ErrorMessage())
	assert.True(t, scan.ScannedAt().IsZero())
	assert.False(t, scan.CreatedAt().IsZero())
	assert.False(t, scan.UpdatedAt().IsZero())

	// Check that an event was raised
	events := scan.Events()
	require.Len(t, events, 1)
	initiated, ok := events[0].(moderation.NSFWScanInitiatedEvent)
	require.True(t, ok)
	assert.Equal(t, scanID, initiated.ScanID)
	assert.Equal(t, imageID, initiated.ImageID)
	assert.Equal(t, provider, initiated.Provider)
}

func TestReconstructNSFWScan(t *testing.T) {
	scanID := moderation.NewNSFWScanID()
	imageID := gallery.NewImageID()
	provider := moderation.ProviderSightEngine
	status := moderation.ScanStatusCompleted
	category := moderation.CategoryNudity
	score := 0.85
	details := moderation.NewNSFWDetails(0.85, 0.1, 0.05, 0.02, 0.01)
	errorMsg := ""
	scannedAt := time.Now().Add(-time.Hour)
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)

	scan := moderation.ReconstructNSFWScan(
		scanID, imageID, provider, status, category, score, details,
		errorMsg, scannedAt, createdAt, updatedAt,
	)

	assert.Equal(t, scanID, scan.ID())
	assert.Equal(t, imageID, scan.ImageID())
	assert.Equal(t, provider, scan.Provider())
	assert.Equal(t, status, scan.Status())
	assert.Equal(t, category, scan.Category())
	assert.Equal(t, score, scan.Score())
	assert.Equal(t, details, scan.Details())
	assert.Equal(t, errorMsg, scan.ErrorMessage())
	assert.Equal(t, scannedAt, scan.ScannedAt())
	assert.Equal(t, createdAt, scan.CreatedAt())
	assert.Equal(t, updatedAt, scan.UpdatedAt())

	// Reconstruction should not raise events
	events := scan.Events()
	assert.Empty(t, events)
}

func TestNSFWScan_MarkScanning(t *testing.T) {
	t.Run("transitions from pending to scanning", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		scan.ClearEvents() // Clear initialization event

		err := scan.MarkScanning()

		assert.NoError(t, err)
		assert.Equal(t, moderation.ScanStatusScanning, scan.Status())
	})

	t.Run("fails when already completed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategoryNudity, 0.5, details)

		err := scan.MarkScanning()

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyCompleted)
	})

	t.Run("fails when already failed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		_ = scan.Fail("some error")

		err := scan.MarkScanning()

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyFailed)
	})
}

func TestNSFWScan_Complete(t *testing.T) {
	t.Run("successfully completes scan with results", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		scan.ClearEvents() // Clear initialization event

		category := moderation.CategoryNudity
		score := 0.85
		details := moderation.NewNSFWDetails(0.85, 0.1, 0.05, 0.02, 0.01)

		err := scan.Complete(category, score, details)

		assert.NoError(t, err)
		assert.Equal(t, moderation.ScanStatusCompleted, scan.Status())
		assert.Equal(t, category, scan.Category())
		assert.Equal(t, score, scan.Score())
		assert.Equal(t, details.NudityScore, scan.Details().NudityScore)
		assert.False(t, scan.ScannedAt().IsZero())

		// Check completion event
		events := scan.Events()
		require.Len(t, events, 1)
		completed, ok := events[0].(moderation.NSFWScanCompletedEvent)
		require.True(t, ok)
		assert.Equal(t, category, completed.Category)
		assert.Equal(t, score, completed.Score)
		assert.True(t, completed.IsNSFW)
	})

	t.Run("fails with score out of range", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)

		err := scan.Complete(moderation.CategoryNudity, 1.5, details)
		assert.ErrorIs(t, err, moderation.ErrNSFWScoreOutOfRange)

		err = scan.Complete(moderation.CategoryNudity, -0.1, details)
		assert.ErrorIs(t, err, moderation.ErrNSFWScoreOutOfRange)
	})

	t.Run("fails when already completed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategoryNudity, 0.5, details)

		err := scan.Complete(moderation.CategorySafe, 0.1, details)

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyCompleted)
	})

	t.Run("fails when already failed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		_ = scan.Fail("some error")

		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		err := scan.Complete(moderation.CategorySafe, 0.1, details)

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyFailed)
	})
}

func TestNSFWScan_Fail(t *testing.T) {
	t.Run("successfully marks scan as failed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		scan.ClearEvents() // Clear initialization event

		errorMsg := "API timeout"
		err := scan.Fail(errorMsg)

		assert.NoError(t, err)
		assert.Equal(t, moderation.ScanStatusFailed, scan.Status())
		assert.Equal(t, errorMsg, scan.ErrorMessage())

		// Check failure event
		events := scan.Events()
		require.Len(t, events, 1)
		failed, ok := events[0].(moderation.NSFWScanFailedEvent)
		require.True(t, ok)
		assert.Equal(t, errorMsg, failed.Error)
	})

	t.Run("fails when already completed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategoryNudity, 0.5, details)

		err := scan.Fail("some error")

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyCompleted)
	})

	t.Run("fails when already failed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		_ = scan.Fail("first error")

		err := scan.Fail("second error")

		assert.ErrorIs(t, err, moderation.ErrNSFWScanAlreadyFailed)
	})
}

func TestNSFWScan_IsNSFW(t *testing.T) {
	tests := []struct {
		name     string
		category moderation.NSFWCategory
		expected bool
	}{
		{"safe is not NSFW", moderation.CategorySafe, false},
		{"suggestive is not NSFW", moderation.CategorySuggestive, false},
		{"nudity is NSFW", moderation.CategoryNudity, true},
		{"explicit is NSFW", moderation.CategoryExplicit, true},
		{"violence is NSFW", moderation.CategoryViolence, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scan := moderation.NewNSFWScan(
				moderation.NewNSFWScanID(),
				gallery.NewImageID(),
				moderation.ProviderSightEngine,
			)
			details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
			_ = scan.Complete(tt.category, 0.5, details)

			assert.Equal(t, tt.expected, scan.IsNSFW())
		})
	}

	t.Run("returns false for incomplete scan", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)

		assert.False(t, scan.IsNSFW())
	})
}

func TestNSFWScan_RequiresReview(t *testing.T) {
	t.Run("suggestive requires review", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategorySuggestive, 0.5, details)

		assert.True(t, scan.RequiresReview())
	})

	t.Run("safe does not require review", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.1, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategorySafe, 0.1, details)

		assert.False(t, scan.RequiresReview())
	})
}

func TestNSFWScan_StatusChecks(t *testing.T) {
	t.Run("IsPending", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)

		assert.True(t, scan.IsPending())
		assert.False(t, scan.IsCompleted())
		assert.False(t, scan.IsFailed())
	})

	t.Run("IsCompleted", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		details := moderation.NewNSFWDetails(0.5, 0, 0, 0, 0)
		_ = scan.Complete(moderation.CategorySafe, 0.1, details)

		assert.False(t, scan.IsPending())
		assert.True(t, scan.IsCompleted())
		assert.False(t, scan.IsFailed())
	})

	t.Run("IsFailed", func(t *testing.T) {
		scan := moderation.NewNSFWScan(
			moderation.NewNSFWScanID(),
			gallery.NewImageID(),
			moderation.ProviderSightEngine,
		)
		_ = scan.Fail("error")

		assert.False(t, scan.IsPending())
		assert.False(t, scan.IsCompleted())
		assert.True(t, scan.IsFailed())
	})
}
