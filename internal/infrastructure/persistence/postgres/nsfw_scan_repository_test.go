package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

// createTestImageForNSFW creates a test user and image for NSFW scan tests.
func createTestImageForNSFW(t *testing.T, db *sqlx.DB, imageID gallery.ImageID) {
	t.Helper()
	userID := identity.NewUserID()
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)
}

func TestNSFWScanRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)
	scanID := moderation.NewNSFWScanID()

	// Create a new NSFW scan
	scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)

	// Save the scan
	err := repo.Save(ctx, scan)
	require.NoError(t, err)

	// Retrieve it to verify
	retrieved, err := repo.FindByID(ctx, scanID)
	require.NoError(t, err)
	assert.Equal(t, scanID.String(), retrieved.ID().String())
	assert.Equal(t, imageID.String(), retrieved.ImageID().String())
	assert.Equal(t, moderation.ProviderSightEngine, retrieved.Provider())
	assert.Equal(t, moderation.ScanStatusPending, retrieved.Status())
}

func TestNSFWScanRepository_SaveWithResults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)
	scanID := moderation.NewNSFWScanID()

	// Create and complete a scan
	scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)
	details := moderation.NewNSFWDetails(0.85, 0.1, 0.05, 0.02, 0.01).
		WithSubCategories([]string{"nudity", "explicit"})
	err := scan.Complete(moderation.CategoryNudity, 0.85, details)
	require.NoError(t, err)

	// Save with results
	err = repo.Save(ctx, scan)
	require.NoError(t, err)

	// Retrieve and verify results
	retrieved, err := repo.FindByID(ctx, scanID)
	require.NoError(t, err)
	assert.Equal(t, moderation.ScanStatusCompleted, retrieved.Status())
	assert.Equal(t, moderation.CategoryNudity, retrieved.Category())
	assert.Equal(t, 0.85, retrieved.Score())
	assert.Equal(t, 0.85, retrieved.Details().NudityScore)
	assert.Equal(t, []string{"nudity", "explicit"}, retrieved.Details().SubCategories)
}

func TestNSFWScanRepository_SaveWithError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)
	scanID := moderation.NewNSFWScanID()

	// Create and fail a scan
	scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)
	scan.Fail("API timeout")

	// Save failed scan
	err := repo.Save(ctx, scan)
	require.NoError(t, err)

	// Retrieve and verify error
	retrieved, err := repo.FindByID(ctx, scanID)
	require.NoError(t, err)
	assert.Equal(t, moderation.ScanStatusFailed, retrieved.Status())
	assert.Equal(t, "API timeout", retrieved.ErrorMessage())
}

func TestNSFWScanRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	nonExistentID := moderation.NewNSFWScanID()

	_, err := repo.FindByID(ctx, nonExistentID)
	assert.ErrorIs(t, err, moderation.ErrNSFWScanNotFound)
}

func TestNSFWScanRepository_FindByImageID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)

	// Create and save two scans for the same image
	scan1 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err := repo.Save(ctx, scan1)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Ensure different timestamps

	scan2 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err = repo.Save(ctx, scan2)
	require.NoError(t, err)

	// FindByImageID should return the most recent one
	retrieved, err := repo.FindByImageID(ctx, imageID)
	require.NoError(t, err)
	assert.Equal(t, scan2.ID().String(), retrieved.ID().String())
}

func TestNSFWScanRepository_FindByImageID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	nonExistentImageID := gallery.NewImageID()

	_, err := repo.FindByImageID(ctx, nonExistentImageID)
	assert.ErrorIs(t, err, moderation.ErrNSFWScanNotFound)
}

func TestNSFWScanRepository_FindByImageIDAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)

	// Create and save three scans for the same image
	scan1 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err := repo.Save(ctx, scan1)
	require.NoError(t, err)

	scan2 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err = repo.Save(ctx, scan2)
	require.NoError(t, err)

	scan3 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err = repo.Save(ctx, scan3)
	require.NoError(t, err)

	// FindByImageIDAll should return all scans
	scans, err := repo.FindByImageIDAll(ctx, imageID)
	require.NoError(t, err)
	assert.Len(t, scans, 3)
}

func TestNSFWScanRepository_FindPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()

	// Create pending and completed scans
	imageID1 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID1)
	pendingScan1 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID1, moderation.ProviderSightEngine)
	err := repo.Save(ctx, pendingScan1)
	require.NoError(t, err)

	imageID2 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID2)
	pendingScan2 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID2, moderation.ProviderSightEngine)
	err = repo.Save(ctx, pendingScan2)
	require.NoError(t, err)

	imageID3 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID3)
	completedScan := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID3, moderation.ProviderSightEngine)
	details := moderation.NewNSFWDetails(0.1, 0.1, 0.1, 0.1, 0.1)
	err = completedScan.Complete(moderation.CategorySafe, 0.1, details)
	require.NoError(t, err)
	err = repo.Save(ctx, completedScan)
	require.NoError(t, err)

	// Find pending scans
	pagination, _ := shared.NewPagination(1, 10)
	scans, count, err := repo.FindPending(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, scans, 2)
	assert.Equal(t, int64(2), count)
}

func TestNSFWScanRepository_FindByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()

	// Create scans with different statuses
	imageID1 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID1)
	completedScan1 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID1, moderation.ProviderSightEngine)
	details := moderation.NewNSFWDetails(0.2, 0.1, 0.1, 0.1, 0.1)
	err := completedScan1.Complete(moderation.CategorySafe, 0.2, details)
	require.NoError(t, err)
	err = repo.Save(ctx, completedScan1)
	require.NoError(t, err)

	imageID2 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID2)
	completedScan2 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID2, moderation.ProviderSightEngine)
	err = completedScan2.Complete(moderation.CategorySafe, 0.15, details)
	require.NoError(t, err)
	err = repo.Save(ctx, completedScan2)
	require.NoError(t, err)

	imageID3 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID3)
	failedScan := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID3, moderation.ProviderSightEngine)
	failedScan.Fail("Network error")
	err = repo.Save(ctx, failedScan)
	require.NoError(t, err)

	// Find completed scans
	pagination, _ := shared.NewPagination(1, 10)
	scans, count, err := repo.FindByStatus(ctx, moderation.ScanStatusCompleted, pagination)
	require.NoError(t, err)
	assert.Len(t, scans, 2)
	assert.Equal(t, int64(2), count)

	// Find failed scans
	scans, count, err = repo.FindByStatus(ctx, moderation.ScanStatusFailed, pagination)
	require.NoError(t, err)
	assert.Len(t, scans, 1)
	assert.Equal(t, int64(1), count)
}

func TestNSFWScanRepository_FindNSFWImages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()

	// Create scans with different categories
	imageID1 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID1)
	nsfwScan1 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID1, moderation.ProviderSightEngine)
	details := moderation.NewNSFWDetails(0.9, 0.1, 0.1, 0.1, 0.1)
	err := nsfwScan1.Complete(moderation.CategoryNudity, 0.9, details)
	require.NoError(t, err)
	err = repo.Save(ctx, nsfwScan1)
	require.NoError(t, err)

	imageID2 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID2)
	nsfwScan2 := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID2, moderation.ProviderSightEngine)
	violenceDetails := moderation.NewNSFWDetails(0.1, 0.1, 0.85, 0.1, 0.1)
	err = nsfwScan2.Complete(moderation.CategoryViolence, 0.85, violenceDetails)
	require.NoError(t, err)
	err = repo.Save(ctx, nsfwScan2)
	require.NoError(t, err)

	imageID3 := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID3)
	safeScan := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID3, moderation.ProviderSightEngine)
	safeDetails := moderation.NewNSFWDetails(0.1, 0.1, 0.1, 0.1, 0.1)
	err = safeScan.Complete(moderation.CategorySafe, 0.1, safeDetails)
	require.NoError(t, err)
	err = repo.Save(ctx, safeScan)
	require.NoError(t, err)

	// Find NSFW images (should only return nudity and violence, not safe)
	pagination, _ := shared.NewPagination(1, 10)
	scans, count, err := repo.FindNSFWImages(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, scans, 2)
	assert.Equal(t, int64(2), count)

	// Verify categories
	categories := []moderation.NSFWCategory{scans[0].Category(), scans[1].Category()}
	assert.Contains(t, categories, moderation.CategoryNudity)
	assert.Contains(t, categories, moderation.CategoryViolence)
}

func TestNSFWScanRepository_HasActiveScan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)

	// Initially no active scan
	hasActive, err := repo.HasActiveScan(ctx, imageID)
	require.NoError(t, err)
	assert.False(t, hasActive)

	// Create a pending scan
	pendingScan := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
	err = repo.Save(ctx, pendingScan)
	require.NoError(t, err)

	// Now should have active scan
	hasActive, err = repo.HasActiveScan(ctx, imageID)
	require.NoError(t, err)
	assert.True(t, hasActive)

	// Complete the scan
	details := moderation.NewNSFWDetails(0.1, 0.1, 0.1, 0.1, 0.1)
	err = pendingScan.Complete(moderation.CategorySafe, 0.1, details)
	require.NoError(t, err)
	err = repo.Save(ctx, pendingScan)
	require.NoError(t, err)

	// No longer has active scan
	hasActive, err = repo.HasActiveScan(ctx, imageID)
	require.NoError(t, err)
	assert.False(t, hasActive)
}

func TestNSFWScanRepository_SaveUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestImageForNSFW(t, db, imageID)
	scanID := moderation.NewNSFWScanID()

	// Create initial scan
	scan := moderation.NewNSFWScan(scanID, imageID, moderation.ProviderSightEngine)
	err := repo.Save(ctx, scan)
	require.NoError(t, err)

	// Verify initial state
	retrieved, err := repo.FindByID(ctx, scanID)
	require.NoError(t, err)
	assert.Equal(t, moderation.ScanStatusPending, retrieved.Status())

	// Update the scan
	details := moderation.NewNSFWDetails(0.3, 0.1, 0.1, 0.1, 0.1)
	err = retrieved.Complete(moderation.CategorySafe, 0.3, details)
	require.NoError(t, err)
	err = repo.Save(ctx, retrieved)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, scanID)
	require.NoError(t, err)
	assert.Equal(t, moderation.ScanStatusCompleted, updated.Status())
	assert.Equal(t, moderation.CategorySafe, updated.Category())
	assert.Equal(t, 0.3, updated.Score())
}

func TestNSFWScanRepository_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewNSFWScanRepository(db)

	ctx := context.Background()

	// Create 5 pending scans
	for i := 0; i < 5; i++ {
		imageID := gallery.NewImageID()

		// Setup: Create test user and image
		createTestImageForNSFW(t, db, imageID)
		scan := moderation.NewNSFWScan(moderation.NewNSFWScanID(), imageID, moderation.ProviderSightEngine)
		err := repo.Save(ctx, scan)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond) // Ensure different timestamps
	}

	// Test pagination - first page (page 1, 2 items per page)
	pagination1, _ := shared.NewPagination(1, 2)
	scans, count, err := repo.FindPending(ctx, pagination1)
	require.NoError(t, err)
	assert.Len(t, scans, 2)
	assert.Equal(t, int64(5), count)

	// Test pagination - second page (page 2, 2 items per page)
	pagination2, _ := shared.NewPagination(2, 2)
	scans, count, err = repo.FindPending(ctx, pagination2)
	require.NoError(t, err)
	assert.Len(t, scans, 2)
	assert.Equal(t, int64(5), count)

	// Test pagination - third page (page 3, 2 items per page)
	pagination3, _ := shared.NewPagination(3, 2)
	scans, count, err = repo.FindPending(ctx, pagination3)
	require.NoError(t, err)
	assert.Len(t, scans, 1)
	assert.Equal(t, int64(5), count)
}
