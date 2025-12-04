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
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// setupTestDB creates a test database using testcontainers.
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	return pgContainer.DB
}

// createTestUser inserts a test user into the database.
func createTestUser(t *testing.T, db *sqlx.DB, userID identity.UserID) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO users (id, email, username, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		userID.String(),
		userID.String()+"@example.com",
		"user"+userID.String()[:8],
		"$argon2id$v=19$m=65536,t=3,p=2$salt$hash",
		"user",
		"active",
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

// createTestImage inserts a test image into the database.
func createTestImage(t *testing.T, db *sqlx.DB, imageID gallery.ImageID, ownerID identity.UserID) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO images (
			id, owner_id, title, description, storage_provider, storage_key,
			original_filename, mime_type, file_size, width, height,
			status, visibility, scan_status, view_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		imageID.String(),
		ownerID.String(),
		"Test Image",
		"Test description",
		"local",
		"test/"+imageID.String()+".jpg",
		"test.jpg",
		"image/jpeg",
		1024*100, // 100KB
		1920,
		1080,
		"active",
		"public",
		"clean",
		0, // view_count
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func TestLikeRepository_Like(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("creates new like successfully", func(t *testing.T) {
		err := repo.Like(ctx, userID, imageID)
		require.NoError(t, err)

		// Verify like exists
		hasLiked, err := repo.HasLiked(ctx, userID, imageID)
		require.NoError(t, err)
		assert.True(t, hasLiked)
	})

	t.Run("is idempotent - can like same image twice", func(t *testing.T) {
		userID2 := identity.NewUserID()
		imageID2 := gallery.NewImageID()

		createTestUser(t, db, userID2)
		createTestImage(t, db, imageID2, userID2)

		// Like twice
		err := repo.Like(ctx, userID2, imageID2)
		require.NoError(t, err)

		err = repo.Like(ctx, userID2, imageID2)
		require.NoError(t, err)

		// Should only count once
		count, err := repo.GetLikeCount(ctx, imageID2)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("fails with invalid user ID", func(t *testing.T) {
		invalidUserID := identity.NewUserID()
		validImageID := gallery.NewImageID()

		createTestImage(t, db, validImageID, userID)

		err := repo.Like(ctx, invalidUserID, validImageID)
		assert.Error(t, err) // Foreign key constraint violation
	})

	t.Run("fails with invalid image ID", func(t *testing.T) {
		invalidImageID := gallery.NewImageID()

		err := repo.Like(ctx, userID, invalidImageID)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestLikeRepository_Unlike(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("removes existing like successfully", func(t *testing.T) {
		// Create like first
		err := repo.Like(ctx, userID, imageID)
		require.NoError(t, err)

		// Remove like
		err = repo.Unlike(ctx, userID, imageID)
		require.NoError(t, err)

		// Verify like doesn't exist
		hasLiked, err := repo.HasLiked(ctx, userID, imageID)
		require.NoError(t, err)
		assert.False(t, hasLiked)
	})

	t.Run("is idempotent - can unlike non-existent like", func(t *testing.T) {
		userID2 := identity.NewUserID()
		imageID2 := gallery.NewImageID()

		createTestUser(t, db, userID2)
		createTestImage(t, db, imageID2, userID2)

		// Unlike without liking first
		err := repo.Unlike(ctx, userID2, imageID2)
		require.NoError(t, err)

		// Verify no like exists
		hasLiked, err := repo.HasLiked(ctx, userID2, imageID2)
		require.NoError(t, err)
		assert.False(t, hasLiked)
	})
}

func TestLikeRepository_HasLiked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("returns true when like exists", func(t *testing.T) {
		err := repo.Like(ctx, userID, imageID)
		require.NoError(t, err)

		hasLiked, err := repo.HasLiked(ctx, userID, imageID)
		require.NoError(t, err)
		assert.True(t, hasLiked)
	})

	t.Run("returns false when like doesn't exist", func(t *testing.T) {
		userID2 := identity.NewUserID()
		imageID2 := gallery.NewImageID()

		createTestUser(t, db, userID2)
		createTestImage(t, db, imageID2, userID2)

		hasLiked, err := repo.HasLiked(ctx, userID2, imageID2)
		require.NoError(t, err)
		assert.False(t, hasLiked)
	})
}

func TestLikeRepository_GetLikeCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test owner and image
	createTestUser(t, db, ownerID)
	createTestImage(t, db, imageID, ownerID)

	t.Run("returns 0 for image with no likes", func(t *testing.T) {
		count, err := repo.GetLikeCount(ctx, imageID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("returns correct count for image with multiple likes", func(t *testing.T) {
		// Create 3 users and have them like the image
		for i := 0; i < 3; i++ {
			userID := identity.NewUserID()
			createTestUser(t, db, userID)
			err := repo.Like(ctx, userID, imageID)
			require.NoError(t, err)
		}

		count, err := repo.GetLikeCount(ctx, imageID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("decreases count after unlike", func(t *testing.T) {
		imageID2 := gallery.NewImageID()
		createTestImage(t, db, imageID2, ownerID)

		// Create and like
		userID := identity.NewUserID()
		createTestUser(t, db, userID)
		err := repo.Like(ctx, userID, imageID2)
		require.NoError(t, err)

		// Unlike
		err = repo.Unlike(ctx, userID, imageID2)
		require.NoError(t, err)

		// Count should be 0
		count, err := repo.GetLikeCount(ctx, imageID2)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestLikeRepository_GetLikedImageIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	ownerID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, userID)
	createTestUser(t, db, ownerID)

	t.Run("returns empty list when user hasn't liked anything", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 20)
		imageIDs, err := repo.GetLikedImageIDs(ctx, userID, pagination)
		require.NoError(t, err)
		assert.Empty(t, imageIDs)
	})

	t.Run("returns liked image IDs in descending order", func(t *testing.T) {
		// Create 3 images and like them
		imageID1 := gallery.NewImageID()
		imageID2 := gallery.NewImageID()
		imageID3 := gallery.NewImageID()

		createTestImage(t, db, imageID1, ownerID)
		createTestImage(t, db, imageID2, ownerID)
		createTestImage(t, db, imageID3, ownerID)

		// Like in order: 1, 2, 3
		err := repo.Like(ctx, userID, imageID1)
		require.NoError(t, err)

		err = repo.Like(ctx, userID, imageID2)
		require.NoError(t, err)

		err = repo.Like(ctx, userID, imageID3)
		require.NoError(t, err)

		// Get liked images (should be in reverse order: 3, 2, 1)
		pagination, _ := shared.NewPagination(1, 20)
		imageIDs, err := repo.GetLikedImageIDs(ctx, userID, pagination)
		require.NoError(t, err)
		require.Len(t, imageIDs, 3)

		// Most recent like should be first
		assert.Equal(t, imageID3.String(), imageIDs[0].String())
		assert.Equal(t, imageID2.String(), imageIDs[1].String())
		assert.Equal(t, imageID1.String(), imageIDs[2].String())
	})

	t.Run("respects pagination", func(t *testing.T) {
		userID2 := identity.NewUserID()
		createTestUser(t, db, userID2)

		// Create 5 images and like them
		for i := 0; i < 5; i++ {
			imageID := gallery.NewImageID()
			createTestImage(t, db, imageID, ownerID)
			err := repo.Like(ctx, userID2, imageID)
			require.NoError(t, err)
		}

		// Get first page (2 items)
		pagination, _ := shared.NewPagination(1, 2)
		imageIDs, err := repo.GetLikedImageIDs(ctx, userID2, pagination)
		require.NoError(t, err)
		assert.Len(t, imageIDs, 2)

		// Get second page (2 items)
		pagination, _ = shared.NewPagination(2, 2)
		imageIDs, err = repo.GetLikedImageIDs(ctx, userID2, pagination)
		require.NoError(t, err)
		assert.Len(t, imageIDs, 2)

		// Get third page (1 item remaining)
		pagination, _ = shared.NewPagination(3, 2)
		imageIDs, err = repo.GetLikedImageIDs(ctx, userID2, pagination)
		require.NoError(t, err)
		assert.Len(t, imageIDs, 1)
	})
}

func TestLikeRepository_CountLikedImagesByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewLikeRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	ownerID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, userID)
	createTestUser(t, db, ownerID)

	t.Run("returns 0 when user hasn't liked anything", func(t *testing.T) {
		count, err := repo.CountLikedImagesByUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("returns correct count of liked images", func(t *testing.T) {
		// Create 3 images and like them
		for i := 0; i < 3; i++ {
			imageID := gallery.NewImageID()
			createTestImage(t, db, imageID, ownerID)
			err := repo.Like(ctx, userID, imageID)
			require.NoError(t, err)
		}

		count, err := repo.CountLikedImagesByUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("updates count after unlike", func(t *testing.T) {
		userID2 := identity.NewUserID()
		createTestUser(t, db, userID2)

		// Create and like 2 images
		imageID1 := gallery.NewImageID()
		imageID2 := gallery.NewImageID()

		createTestImage(t, db, imageID1, ownerID)
		createTestImage(t, db, imageID2, ownerID)

		err := repo.Like(ctx, userID2, imageID1)
		require.NoError(t, err)

		err = repo.Like(ctx, userID2, imageID2)
		require.NoError(t, err)

		// Unlike one
		err = repo.Unlike(ctx, userID2, imageID1)
		require.NoError(t, err)

		// Count should be 1
		count, err := repo.CountLikedImagesByUser(ctx, userID2)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}
