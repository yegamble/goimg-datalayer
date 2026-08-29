package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

// createTestGroupAlbum inserts a test group album into the database.
func createTestGroupAlbum(
	t *testing.T,
	db *sqlx.DB,
	albumID community.GroupAlbumID,
	groupID community.GroupID,
	createdBy identity.UserID,
	title string,
	isPublic bool,
) {
	t.Helper()

	_, err := db.Exec(
		`
		INSERT INTO group_albums (
			id, group_id, created_by, title, description, cover_image_id,
			image_count, is_public, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		albumID.String(),
		groupID.String(),
		createdBy.String(),
		title,
		"",
		nil,
		0,
		isPublic,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func TestGroupAlbumRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupAlbumRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	creatorID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test users and group
	createTestUser(t, db, ownerID)
	createTestUser(t, db, creatorID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	t.Run("saves new album successfully", func(t *testing.T) {
		album, err := community.NewGroupAlbum(groupID, creatorID, "My Album", true)
		require.NoError(t, err)

		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Verify album exists
		found, err := repo.FindByID(ctx, album.ID())
		require.NoError(t, err)
		assert.Equal(t, album.ID(), found.ID())
		assert.Equal(t, groupID, found.GroupID())
		assert.Equal(t, creatorID, found.CreatedBy())
		assert.Equal(t, "My Album", found.Title())
		assert.True(t, found.IsPublic())
		assert.Equal(t, 0, found.ImageCount())
	})

	t.Run("updates existing album successfully", func(t *testing.T) {
		album, err := community.NewGroupAlbum(groupID, creatorID, "Original Title", true)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Update title
		err = album.UpdateTitle("Updated Title")
		require.NoError(t, err)

		// Save again
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, album.ID())
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", found.Title())
	})

	t.Run("updates album with description and cover image", func(t *testing.T) {
		imageID := gallery.NewImageID()
		createTestImage(t, db, imageID, creatorID)

		album, err := community.NewGroupAlbum(groupID, creatorID, "Test Album", false)
		require.NoError(t, err)

		// Save initial album
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Update description and cover image
		err = album.UpdateDescription("This is a test album description")
		require.NoError(t, err)
		album.SetCoverImage(&imageID)

		// Save updates
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Verify updates
		found, err := repo.FindByID(ctx, album.ID())
		require.NoError(t, err)
		assert.Equal(t, "This is a test album description", found.Description())
		assert.NotNil(t, found.CoverImageID())
		assert.Equal(t, imageID, *found.CoverImageID())
		assert.False(t, found.IsPublic())
	})

	t.Run("updates album image count", func(t *testing.T) {
		album, err := community.NewGroupAlbum(groupID, creatorID, "Count Test", true)
		require.NoError(t, err)

		// Save initial album
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Increment image count
		album.IncrementImageCount()
		album.IncrementImageCount()

		// Save updates
		err = repo.Save(ctx, album)
		require.NoError(t, err)

		// Verify count
		found, err := repo.FindByID(ctx, album.ID())
		require.NoError(t, err)
		assert.Equal(t, 2, found.ImageCount())
	})

	t.Run("fails when group does not exist", func(t *testing.T) {
		invalidGroupID := community.NewGroupID()

		album, err := community.NewGroupAlbum(invalidGroupID, creatorID, "Test Album", true)
		require.NoError(t, err)

		err = repo.Save(ctx, album)
		assert.Error(t, err) // Foreign key constraint violation
	})

	t.Run("fails when creator does not exist", func(t *testing.T) {
		invalidCreatorID := identity.NewUserID()

		album, err := community.NewGroupAlbum(groupID, invalidCreatorID, "Test Album", true)
		require.NoError(t, err)

		err = repo.Save(ctx, album)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestGroupAlbumRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupAlbumRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	creatorID := identity.NewUserID()
	groupID := community.NewGroupID()
	albumID := community.NewGroupAlbumID()

	// Setup: Create test users, group, and album
	createTestUser(t, db, ownerID)
	createTestUser(t, db, creatorID)
	createTestGroup(t, db, groupID, ownerID, "test-group")
	createTestGroupAlbum(t, db, albumID, groupID, creatorID, "Test Album", true)

	t.Run("finds existing album by ID", func(t *testing.T) {
		album, err := repo.FindByID(ctx, albumID)
		require.NoError(t, err)
		assert.Equal(t, albumID, album.ID())
		assert.Equal(t, groupID, album.GroupID())
		assert.Equal(t, creatorID, album.CreatedBy())
		assert.Equal(t, "Test Album", album.Title())
		assert.True(t, album.IsPublic())
	})

	t.Run("returns ErrGroupAlbumNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewGroupAlbumID()
		_, err := repo.FindByID(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrGroupAlbumNotFound)
	})

	t.Run("reconstructs album with cover image", func(t *testing.T) {
		imageID := gallery.NewImageID()
		albumWithCoverID := community.NewGroupAlbumID()

		createTestImage(t, db, imageID, creatorID)

		// Insert album with cover image
		_, err := db.Exec(
			`
			INSERT INTO group_albums (
				id, group_id, created_by, title, description, cover_image_id,
				image_count, is_public, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			albumWithCoverID.String(),
			groupID.String(),
			creatorID.String(),
			"Album with Cover",
			"Description",
			imageID.String(),
			5,
			false,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		require.NoError(t, err)

		album, err := repo.FindByID(ctx, albumWithCoverID)
		require.NoError(t, err)
		assert.NotNil(t, album.CoverImageID())
		assert.Equal(t, imageID, *album.CoverImageID())
		assert.Equal(t, "Description", album.Description())
		assert.Equal(t, 5, album.ImageCount())
		assert.False(t, album.IsPublic())
	})
}

func TestGroupAlbumRepository_FindByGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupAlbumRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	creator1ID := identity.NewUserID()
	creator2ID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test users and group
	createTestUser(t, db, ownerID)
	createTestUser(t, db, creator1ID)
	createTestUser(t, db, creator2ID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create multiple albums
	album1ID := community.NewGroupAlbumID()
	album2ID := community.NewGroupAlbumID()
	album3ID := community.NewGroupAlbumID()

	createTestGroupAlbum(t, db, album1ID, groupID, creator1ID, "Album 1", true)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	createTestGroupAlbum(t, db, album2ID, groupID, creator2ID, "Album 2", true)
	time.Sleep(10 * time.Millisecond)
	createTestGroupAlbum(t, db, album3ID, groupID, creator1ID, "Album 3", false)

	t.Run("finds all albums for group", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 10)

		albums, total, err := repo.FindByGroup(ctx, groupID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, albums, 3)

		// Verify order (newest first)
		assert.Equal(t, album3ID, albums[0].ID())
		assert.Equal(t, album2ID, albums[1].ID())
		assert.Equal(t, album1ID, albums[2].ID())
	})

	t.Run("respects pagination limit", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 2)

		albums, total, err := repo.FindByGroup(ctx, groupID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, albums, 2)
	})

	t.Run("respects pagination page", func(t *testing.T) {
		pagination, _ := shared.NewPagination(2, 2)

		albums, total, err := repo.FindByGroup(ctx, groupID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, albums, 1)
		assert.Equal(t, album1ID, albums[0].ID())
	})

	t.Run("returns empty for group with no albums", func(t *testing.T) {
		emptyGroupID := community.NewGroupID()
		createTestGroup(t, db, emptyGroupID, ownerID, "empty-group")

		pagination, _ := shared.NewPagination(1, 10)

		albums, total, err := repo.FindByGroup(ctx, emptyGroupID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, albums)
	})
}

func TestGroupAlbumRepository_FindByGroupAndCreator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupAlbumRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	creator1ID := identity.NewUserID()
	creator2ID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test users and group
	createTestUser(t, db, ownerID)
	createTestUser(t, db, creator1ID)
	createTestUser(t, db, creator2ID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create albums from different creators
	album1ID := community.NewGroupAlbumID()
	album2ID := community.NewGroupAlbumID()
	album3ID := community.NewGroupAlbumID()
	album4ID := community.NewGroupAlbumID()

	createTestGroupAlbum(t, db, album1ID, groupID, creator1ID, "Creator1 Album 1", true)
	time.Sleep(10 * time.Millisecond)
	createTestGroupAlbum(t, db, album2ID, groupID, creator2ID, "Creator2 Album 1", true)
	time.Sleep(10 * time.Millisecond)
	createTestGroupAlbum(t, db, album3ID, groupID, creator1ID, "Creator1 Album 2", false)
	time.Sleep(10 * time.Millisecond)
	createTestGroupAlbum(t, db, album4ID, groupID, creator1ID, "Creator1 Album 3", true)

	t.Run("finds albums by specific creator", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 10)

		albums, total, err := repo.FindByGroupAndCreator(ctx, groupID, creator1ID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, albums, 3)

		// All albums should be from creator1
		for _, album := range albums {
			assert.Equal(t, creator1ID, album.CreatedBy())
		}

		// Verify order (newest first)
		assert.Equal(t, album4ID, albums[0].ID())
		assert.Equal(t, album3ID, albums[1].ID())
		assert.Equal(t, album1ID, albums[2].ID())
	})

	t.Run("finds albums by other creator", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 10)

		albums, total, err := repo.FindByGroupAndCreator(ctx, groupID, creator2ID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, albums, 1)
		assert.Equal(t, album2ID, albums[0].ID())
		assert.Equal(t, creator2ID, albums[0].CreatedBy())
	})

	t.Run("returns empty for creator with no albums", func(t *testing.T) {
		noAlbumsCreatorID := identity.NewUserID()
		createTestUser(t, db, noAlbumsCreatorID)

		pagination, _ := shared.NewPagination(1, 10)

		albums, total, err := repo.FindByGroupAndCreator(ctx, groupID, noAlbumsCreatorID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, albums)
	})

	t.Run("respects pagination", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 2)

		albums, total, err := repo.FindByGroupAndCreator(ctx, groupID, creator1ID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, albums, 2)
	})
}

func TestGroupAlbumRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupAlbumRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	creatorID := identity.NewUserID()
	groupID := community.NewGroupID()
	albumID := community.NewGroupAlbumID()

	// Setup: Create test users, group, and album
	createTestUser(t, db, ownerID)
	createTestUser(t, db, creatorID)
	createTestGroup(t, db, groupID, ownerID, "test-group")
	createTestGroupAlbum(t, db, albumID, groupID, creatorID, "Test Album", true)

	t.Run("deletes album successfully", func(t *testing.T) {
		err := repo.Delete(ctx, albumID)
		require.NoError(t, err)

		// Verify album is no longer findable
		_, err = repo.FindByID(ctx, albumID)
		require.ErrorIs(t, err, community.ErrGroupAlbumNotFound)
	})

	t.Run("returns ErrGroupAlbumNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewGroupAlbumID()
		err := repo.Delete(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrGroupAlbumNotFound)
	})

	t.Run("cascades delete to album images", func(t *testing.T) {
		// Create album with images
		albumWithImagesID := community.NewGroupAlbumID()
		imageID := gallery.NewImageID()

		createTestImage(t, db, imageID, creatorID)
		createTestGroupAlbum(t, db, albumWithImagesID, groupID, creatorID, "Album with Images", true)

		// Add image to album
		_, err := db.Exec(`
			INSERT INTO group_album_images (group_album_id, image_id, added_at, added_by)
			VALUES ($1, $2, $3, $4)
		`, albumWithImagesID.String(), imageID.String(), time.Now().UTC(), creatorID.String())
		require.NoError(t, err)

		// Delete album
		err = repo.Delete(ctx, albumWithImagesID)
		require.NoError(t, err)

		// Verify album images were deleted (cascade)
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM group_album_images WHERE group_album_id = $1", albumWithImagesID.String())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
