package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

func TestCommentRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("saves new comment successfully", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "This is a test comment")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		// Verify comment was saved
		found, err := repo.FindByID(ctx, comment.ID())
		require.NoError(t, err)
		assert.Equal(t, comment.ID(), found.ID())
		assert.Equal(t, comment.UserID(), found.UserID())
		assert.Equal(t, comment.ImageID(), found.ImageID())
		assert.Equal(t, comment.Content(), found.Content())
	})

	t.Run("fails with invalid user ID", func(t *testing.T) {
		invalidUserID := identity.NewUserID()
		validImageID := gallery.NewImageID()

		createTestImage(t, db, validImageID, userID)

		comment, err := gallery.NewComment(validImageID, invalidUserID, "Test comment")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		assert.Error(t, err) // Foreign key constraint violation
	})

	t.Run("fails with invalid image ID", func(t *testing.T) {
		invalidImageID := gallery.NewImageID()

		comment, err := gallery.NewComment(invalidImageID, userID, "Test comment")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestCommentRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("finds existing comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Test comment content")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, comment.ID())
		require.NoError(t, err)
		assert.Equal(t, comment.ID(), found.ID())
		assert.Equal(t, comment.Content(), found.Content())
	})

	t.Run("returns error for non-existent comment", func(t *testing.T) {
		nonExistentID := gallery.NewCommentID()

		found, err := repo.FindByID(ctx, nonExistentID)
		assert.Nil(t, found)
		require.ErrorIs(t, err, gallery.ErrCommentNotFound)
	})

	t.Run("returns error for deleted comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Comment to be deleted")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		// Delete the comment
		err = repo.Delete(ctx, comment.ID())
		require.NoError(t, err)

		// Try to find deleted comment
		found, err := repo.FindByID(ctx, comment.ID())
		assert.Nil(t, found)
		require.ErrorIs(t, err, gallery.ErrCommentNotFound)
	})
}

func TestCommentRepository_FindByImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, ownerID)
	createTestImage(t, db, imageID, ownerID)

	t.Run("returns empty list for image with no comments", func(t *testing.T) {
		emptyImageID := gallery.NewImageID()
		createTestImage(t, db, emptyImageID, ownerID)

		pagination, _ := shared.NewPagination(1, 20)
		comments, total, err := repo.FindByImage(ctx, emptyImageID, pagination)
		require.NoError(t, err)
		assert.Empty(t, comments)
		assert.Equal(t, int64(0), total)
	})

	t.Run("returns comments ordered by creation time (oldest first)", func(t *testing.T) {
		// Create 3 users and have them comment
		userIDs := make([]identity.UserID, 3)
		for i := 0; i < 3; i++ {
			userIDs[i] = identity.NewUserID()
			createTestUser(t, db, userIDs[i])

			comment, err := gallery.NewComment(imageID, userIDs[i], fmt.Sprintf("Comment %d", i+1))
			require.NoError(t, err)

			err = repo.Save(ctx, comment)
			require.NoError(t, err)
		}

		pagination, _ := shared.NewPagination(1, 20)
		comments, total, err := repo.FindByImage(ctx, imageID, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 3)
		assert.Equal(t, int64(3), total)

		// Verify oldest first
		assert.Equal(t, "Comment 1", comments[0].Content())
		assert.Equal(t, "Comment 2", comments[1].Content())
		assert.Equal(t, "Comment 3", comments[2].Content())
	})

	t.Run("respects pagination", func(t *testing.T) {
		imageID2 := gallery.NewImageID()
		createTestImage(t, db, imageID2, ownerID)

		// Create 5 comments
		for i := 0; i < 5; i++ {
			userID := identity.NewUserID()
			createTestUser(t, db, userID)

			comment, err := gallery.NewComment(imageID2, userID, fmt.Sprintf("Comment %d", i+1))
			require.NoError(t, err)

			err = repo.Save(ctx, comment)
			require.NoError(t, err)
		}

		// Get first page (2 items)
		pagination, _ := shared.NewPagination(1, 2)
		comments, total, err := repo.FindByImage(ctx, imageID2, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 2)
		assert.Equal(t, int64(5), total)

		// Get second page (2 items)
		pagination, _ = shared.NewPagination(2, 2)
		comments, total, err = repo.FindByImage(ctx, imageID2, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 2)
		assert.Equal(t, int64(5), total)

		// Get third page (1 item remaining)
		pagination, _ = shared.NewPagination(3, 2)
		comments, total, err = repo.FindByImage(ctx, imageID2, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 1)
		assert.Equal(t, int64(5), total)
	})

	t.Run("excludes deleted comments", func(t *testing.T) {
		imageID3 := gallery.NewImageID()
		createTestImage(t, db, imageID3, ownerID)

		userID := identity.NewUserID()
		createTestUser(t, db, userID)

		// Create 2 comments
		comment1, _ := gallery.NewComment(imageID3, userID, "Comment 1")
		comment2, _ := gallery.NewComment(imageID3, userID, "Comment 2")

		_ = repo.Save(ctx, comment1)
		_ = repo.Save(ctx, comment2)

		// Delete first comment
		_ = repo.Delete(ctx, comment1.ID())

		// Find comments
		pagination, _ := shared.NewPagination(1, 20)
		comments, total, err := repo.FindByImage(ctx, imageID3, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, comment2.ID(), comments[0].ID())
	})
}

func TestCommentRepository_FindByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	ownerID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, userID)
	createTestUser(t, db, ownerID)

	t.Run("returns empty list when user has no comments", func(t *testing.T) {
		pagination, _ := shared.NewPagination(1, 20)
		comments, total, err := repo.FindByUser(ctx, userID, pagination)
		require.NoError(t, err)
		assert.Empty(t, comments)
		assert.Equal(t, int64(0), total)
	})

	t.Run("returns user's comments across multiple images", func(t *testing.T) {
		// Create 3 images and have user comment on each
		for i := 0; i < 3; i++ {
			imageID := gallery.NewImageID()
			createTestImage(t, db, imageID, ownerID)

			comment, err := gallery.NewComment(imageID, userID, fmt.Sprintf("User comment %d", i+1))
			require.NoError(t, err)

			err = repo.Save(ctx, comment)
			require.NoError(t, err)
		}

		pagination, _ := shared.NewPagination(1, 20)
		comments, total, err := repo.FindByUser(ctx, userID, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 3)
		assert.Equal(t, int64(3), total)
	})

	t.Run("respects pagination", func(t *testing.T) {
		userID2 := identity.NewUserID()
		createTestUser(t, db, userID2)

		// Create 5 comments
		for i := 0; i < 5; i++ {
			imageID := gallery.NewImageID()
			createTestImage(t, db, imageID, ownerID)

			comment, err := gallery.NewComment(imageID, userID2, fmt.Sprintf("Comment %d", i+1))
			require.NoError(t, err)

			err = repo.Save(ctx, comment)
			require.NoError(t, err)
		}

		// Get first page (2 items)
		pagination, _ := shared.NewPagination(1, 2)
		comments, total, err := repo.FindByUser(ctx, userID2, pagination)
		require.NoError(t, err)
		assert.Len(t, comments, 2)
		assert.Equal(t, int64(5), total)
	})
}

func TestCommentRepository_CountByImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, ownerID)
	createTestImage(t, db, imageID, ownerID)

	t.Run("returns 0 for image with no comments", func(t *testing.T) {
		count, err := repo.CountByImage(ctx, imageID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("returns correct count for image with multiple comments", func(t *testing.T) {
		// Create 3 users and have them comment
		for i := 0; i < 3; i++ {
			userID := identity.NewUserID()
			createTestUser(t, db, userID)

			comment, err := gallery.NewComment(imageID, userID, fmt.Sprintf("Comment %d", i+1))
			require.NoError(t, err)

			err = repo.Save(ctx, comment)
			require.NoError(t, err)
		}

		count, err := repo.CountByImage(ctx, imageID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("excludes deleted comments from count", func(t *testing.T) {
		imageID2 := gallery.NewImageID()
		createTestImage(t, db, imageID2, ownerID)

		userID := identity.NewUserID()
		createTestUser(t, db, userID)

		// Create 2 comments
		comment1, _ := gallery.NewComment(imageID2, userID, "Comment 1")
		comment2, _ := gallery.NewComment(imageID2, userID, "Comment 2")

		_ = repo.Save(ctx, comment1)
		_ = repo.Save(ctx, comment2)

		// Delete first comment
		_ = repo.Delete(ctx, comment1.ID())

		// Count should be 1
		count, err := repo.CountByImage(ctx, imageID2)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("soft deletes existing comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Comment to delete")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		// Delete comment
		err = repo.Delete(ctx, comment.ID())
		require.NoError(t, err)

		// Verify comment is not found
		found, err := repo.FindByID(ctx, comment.ID())
		assert.Nil(t, found)
		require.ErrorIs(t, err, gallery.ErrCommentNotFound)
	})

	t.Run("returns error for non-existent comment", func(t *testing.T) {
		nonExistentID := gallery.NewCommentID()

		err := repo.Delete(ctx, nonExistentID)
		require.ErrorIs(t, err, gallery.ErrCommentNotFound)
	})

	t.Run("is idempotent - can delete already deleted comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Comment to delete twice")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		// Delete once
		err = repo.Delete(ctx, comment.ID())
		require.NoError(t, err)

		// Delete again (should return error)
		err = repo.Delete(ctx, comment.ID())
		require.ErrorIs(t, err, gallery.ErrCommentNotFound)
	})
}

func TestCommentRepository_ExistsByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewCommentRepository(db)

	ctx := context.Background()
	userID := identity.NewUserID()
	imageID := gallery.NewImageID()

	// Setup: Create test user and image
	createTestUser(t, db, userID)
	createTestImage(t, db, imageID, userID)

	t.Run("returns true for existing comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Test comment")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		exists, err := repo.ExistsByID(ctx, comment.ID())
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent comment", func(t *testing.T) {
		nonExistentID := gallery.NewCommentID()

		exists, err := repo.ExistsByID(ctx, nonExistentID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns false for deleted comment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, "Comment to be deleted")
		require.NoError(t, err)

		err = repo.Save(ctx, comment)
		require.NoError(t, err)

		// Delete the comment
		err = repo.Delete(ctx, comment.ID())
		require.NoError(t, err)

		// Check existence
		exists, err := repo.ExistsByID(ctx, comment.ID())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
