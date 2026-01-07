package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

func TestFollowRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, followerID)
	createTestUser(t, db, followedID)

	t.Run("saves new follow relationship successfully", func(t *testing.T) {
		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Verify follow relationship exists
		exists, err := repo.Exists(ctx, followerID, followedID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns error when follow already exists", func(t *testing.T) {
		follower2ID := identity.NewUserID()
		followed2ID := identity.NewUserID()

		createTestUser(t, db, follower2ID)
		createTestUser(t, db, followed2ID)

		follow, err := identity.NewFollow(follower2ID, followed2ID)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Try to save again
		err = repo.Save(ctx, follow)
		require.ErrorIs(t, err, identity.ErrFollowAlreadyExists)
	})

	t.Run("fails with invalid follower ID", func(t *testing.T) {
		invalidFollowerID := identity.NewUserID()
		validFollowedID := identity.NewUserID()

		createTestUser(t, db, validFollowedID)

		follow, err := identity.NewFollow(invalidFollowerID, validFollowedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		assert.Error(t, err) // Foreign key constraint violation
	})

	t.Run("fails with invalid followed ID", func(t *testing.T) {
		validFollowerID := identity.NewUserID()
		invalidFollowedID := identity.NewUserID()

		createTestUser(t, db, validFollowerID)

		follow, err := identity.NewFollow(validFollowerID, invalidFollowedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestFollowRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, followerID)
	createTestUser(t, db, followedID)

	t.Run("deletes existing follow relationship", func(t *testing.T) {
		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Delete the follow
		err = repo.Delete(ctx, followerID, followedID)
		require.NoError(t, err)

		// Verify follow no longer exists
		exists, err := repo.Exists(ctx, followerID, followedID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns error when follow does not exist", func(t *testing.T) {
		follower2ID := identity.NewUserID()
		followed2ID := identity.NewUserID()

		createTestUser(t, db, follower2ID)
		createTestUser(t, db, followed2ID)

		err := repo.Delete(ctx, follower2ID, followed2ID)
		require.ErrorIs(t, err, identity.ErrFollowNotFound)
	})

	t.Run("is idempotent - returns error on second delete", func(t *testing.T) {
		follower3ID := identity.NewUserID()
		followed3ID := identity.NewUserID()

		createTestUser(t, db, follower3ID)
		createTestUser(t, db, followed3ID)

		follow, err := identity.NewFollow(follower3ID, followed3ID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Delete once
		err = repo.Delete(ctx, follower3ID, followed3ID)
		require.NoError(t, err)

		// Delete again (should return error)
		err = repo.Delete(ctx, follower3ID, followed3ID)
		require.ErrorIs(t, err, identity.ErrFollowNotFound)
	})
}

func TestFollowRepository_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, followerID)
	createTestUser(t, db, followedID)

	t.Run("returns true when follow exists", func(t *testing.T) {
		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		exists, err := repo.Exists(ctx, followerID, followedID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false when follow does not exist", func(t *testing.T) {
		follower2ID := identity.NewUserID()
		followed2ID := identity.NewUserID()

		createTestUser(t, db, follower2ID)
		createTestUser(t, db, followed2ID)

		exists, err := repo.Exists(ctx, follower2ID, followed2ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns false after delete", func(t *testing.T) {
		follower3ID := identity.NewUserID()
		followed3ID := identity.NewUserID()

		createTestUser(t, db, follower3ID)
		createTestUser(t, db, followed3ID)

		follow, err := identity.NewFollow(follower3ID, followed3ID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		err = repo.Delete(ctx, follower3ID, followed3ID)
		require.NoError(t, err)

		exists, err := repo.Exists(ctx, follower3ID, followed3ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFollowRepository_FindFollowers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followedID := identity.NewUserID()

	// Setup: Create followed user
	createTestUser(t, db, followedID)

	t.Run("returns empty list when user has no followers", func(t *testing.T) {
		userID := identity.NewUserID()
		createTestUser(t, db, userID)

		followers, total, err := repo.FindFollowers(ctx, userID, 20, 0)
		require.NoError(t, err)
		assert.Empty(t, followers)
		assert.Equal(t, 0, total)
	})

	t.Run("returns all followers ordered by created_at DESC", func(t *testing.T) {
		// Create 3 followers
		followerIDs := make([]identity.UserID, 3)
		for i := 0; i < 3; i++ {
			followerIDs[i] = identity.NewUserID()
			createTestUser(t, db, followerIDs[i])

			follow, err := identity.NewFollow(followerIDs[i], followedID)
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		followers, total, err := repo.FindFollowers(ctx, followedID, 20, 0)
		require.NoError(t, err)
		assert.Len(t, followers, 3)
		assert.Equal(t, 3, total)

		// Verify all followers are present
		for i, follow := range followers {
			assert.Equal(t, followedID, follow.FollowedID())
			// Most recent should be first (reverse order)
			expectedFollowerID := followerIDs[2-i]
			assert.Equal(t, expectedFollowerID, follow.FollowerID())
		}
	})

	t.Run("respects pagination limit", func(t *testing.T) {
		followedID2 := identity.NewUserID()
		createTestUser(t, db, followedID2)

		// Create 5 followers
		for i := 0; i < 5; i++ {
			followerID := identity.NewUserID()
			createTestUser(t, db, followerID)

			follow, err := identity.NewFollow(followerID, followedID2)
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		// Get first page (2 items)
		followers, total, err := repo.FindFollowers(ctx, followedID2, 2, 0)
		require.NoError(t, err)
		assert.Len(t, followers, 2)
		assert.Equal(t, 5, total)

		// Get second page (2 items)
		followers, total, err = repo.FindFollowers(ctx, followedID2, 2, 2)
		require.NoError(t, err)
		assert.Len(t, followers, 2)
		assert.Equal(t, 5, total)

		// Get third page (1 item remaining)
		followers, total, err = repo.FindFollowers(ctx, followedID2, 2, 4)
		require.NoError(t, err)
		assert.Len(t, followers, 1)
		assert.Equal(t, 5, total)
	})
}

func TestFollowRepository_FindFollowing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followerID := identity.NewUserID()

	// Setup: Create follower user
	createTestUser(t, db, followerID)

	t.Run("returns empty list when user is not following anyone", func(t *testing.T) {
		userID := identity.NewUserID()
		createTestUser(t, db, userID)

		following, total, err := repo.FindFollowing(ctx, userID, 20, 0)
		require.NoError(t, err)
		assert.Empty(t, following)
		assert.Equal(t, 0, total)
	})

	t.Run("returns all followed users ordered by created_at DESC", func(t *testing.T) {
		// Create 3 users to follow
		followedIDs := make([]identity.UserID, 3)
		for i := 0; i < 3; i++ {
			followedIDs[i] = identity.NewUserID()
			createTestUser(t, db, followedIDs[i])

			follow, err := identity.NewFollow(followerID, followedIDs[i])
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		following, total, err := repo.FindFollowing(ctx, followerID, 20, 0)
		require.NoError(t, err)
		assert.Len(t, following, 3)
		assert.Equal(t, 3, total)

		// Verify all followed users are present
		for i, follow := range following {
			assert.Equal(t, followerID, follow.FollowerID())
			// Most recent should be first (reverse order)
			expectedFollowedID := followedIDs[2-i]
			assert.Equal(t, expectedFollowedID, follow.FollowedID())
		}
	})

	t.Run("respects pagination limit", func(t *testing.T) {
		followerID2 := identity.NewUserID()
		createTestUser(t, db, followerID2)

		// Create 5 users to follow
		for i := 0; i < 5; i++ {
			followedID := identity.NewUserID()
			createTestUser(t, db, followedID)

			follow, err := identity.NewFollow(followerID2, followedID)
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		// Get first page (2 items)
		following, total, err := repo.FindFollowing(ctx, followerID2, 2, 0)
		require.NoError(t, err)
		assert.Len(t, following, 2)
		assert.Equal(t, 5, total)

		// Get second page (2 items)
		following, total, err = repo.FindFollowing(ctx, followerID2, 2, 2)
		require.NoError(t, err)
		assert.Len(t, following, 2)
		assert.Equal(t, 5, total)

		// Get third page (1 item remaining)
		following, total, err = repo.FindFollowing(ctx, followerID2, 2, 4)
		require.NoError(t, err)
		assert.Len(t, following, 1)
		assert.Equal(t, 5, total)
	})
}

func TestFollowRepository_CountFollowers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followedID := identity.NewUserID()

	// Setup: Create followed user
	createTestUser(t, db, followedID)

	t.Run("returns 0 when user has no followers", func(t *testing.T) {
		count, err := repo.CountFollowers(ctx, followedID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns correct count of followers", func(t *testing.T) {
		// Create 3 followers
		for i := 0; i < 3; i++ {
			followerID := identity.NewUserID()
			createTestUser(t, db, followerID)

			follow, err := identity.NewFollow(followerID, followedID)
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		count, err := repo.CountFollowers(ctx, followedID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("updates count after unfollow", func(t *testing.T) {
		followedID2 := identity.NewUserID()
		createTestUser(t, db, followedID2)

		// Create 2 followers
		follower1ID := identity.NewUserID()
		follower2ID := identity.NewUserID()

		createTestUser(t, db, follower1ID)
		createTestUser(t, db, follower2ID)

		follow1, _ := identity.NewFollow(follower1ID, followedID2)
		follow2, _ := identity.NewFollow(follower2ID, followedID2)

		_ = repo.Save(ctx, follow1)
		_ = repo.Save(ctx, follow2)

		// Unfollow one
		_ = repo.Delete(ctx, follower1ID, followedID2)

		// Count should be 1
		count, err := repo.CountFollowers(ctx, followedID2)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestFollowRepository_CountFollowing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()
	followerID := identity.NewUserID()

	// Setup: Create follower user
	createTestUser(t, db, followerID)

	t.Run("returns 0 when user is not following anyone", func(t *testing.T) {
		count, err := repo.CountFollowing(ctx, followerID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns correct count of followed users", func(t *testing.T) {
		// Create 3 users to follow
		for i := 0; i < 3; i++ {
			followedID := identity.NewUserID()
			createTestUser(t, db, followedID)

			follow, err := identity.NewFollow(followerID, followedID)
			require.NoError(t, err)

			err = repo.Save(ctx, follow)
			require.NoError(t, err)
		}

		count, err := repo.CountFollowing(ctx, followerID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("updates count after unfollow", func(t *testing.T) {
		followerID2 := identity.NewUserID()
		createTestUser(t, db, followerID2)

		// Create 2 users to follow
		followed1ID := identity.NewUserID()
		followed2ID := identity.NewUserID()

		createTestUser(t, db, followed1ID)
		createTestUser(t, db, followed2ID)

		follow1, _ := identity.NewFollow(followerID2, followed1ID)
		follow2, _ := identity.NewFollow(followerID2, followed2ID)

		_ = repo.Save(ctx, follow1)
		_ = repo.Save(ctx, follow2)

		// Unfollow one
		_ = repo.Delete(ctx, followerID2, followed1ID)

		// Count should be 1
		count, err := repo.CountFollowing(ctx, followerID2)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestFollowRepository_CascadeDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewFollowRepository(db)

	ctx := context.Background()

	t.Run("deletes all follows when user is deleted (follower)", func(t *testing.T) {
		followerID := identity.NewUserID()
		followedID := identity.NewUserID()

		createTestUser(t, db, followerID)
		createTestUser(t, db, followedID)

		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Delete the follower user
		_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", followerID.String())
		require.NoError(t, err)

		// Verify follow no longer exists
		exists, err := repo.Exists(ctx, followerID, followedID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("deletes all follows when user is deleted (followed)", func(t *testing.T) {
		followerID := identity.NewUserID()
		followedID := identity.NewUserID()

		createTestUser(t, db, followerID)
		createTestUser(t, db, followedID)

		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		err = repo.Save(ctx, follow)
		require.NoError(t, err)

		// Delete the followed user
		_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", followedID.String())
		require.NoError(t, err)

		// Verify follow no longer exists
		exists, err := repo.Exists(ctx, followerID, followedID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
