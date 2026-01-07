package identity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewFollow(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	t.Run("creates follow with valid inputs", func(t *testing.T) {
		t.Parallel()

		follow, err := identity.NewFollow(followerID, followedID)

		require.NoError(t, err)
		assert.Equal(t, followerID, follow.FollowerID())
		assert.Equal(t, followedID, follow.FollowedID())
		assert.False(t, follow.CreatedAt().IsZero())
		assert.Len(t, follow.Events(), 1)
		assert.WithinDuration(t, time.Now().UTC(), follow.CreatedAt(), 2*time.Second)
	})

	t.Run("emits UserFollowed event", func(t *testing.T) {
		t.Parallel()

		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		events := follow.Events()
		require.Len(t, events, 1)

		event, ok := events[0].(identity.UserFollowed)
		require.True(t, ok)
		assert.Equal(t, "identity.user.followed", event.EventType())
		assert.Equal(t, followerID, event.FollowerID)
		assert.Equal(t, followedID, event.FollowedID)
		assert.False(t, event.OccurredAt().IsZero())
		assert.NotEmpty(t, event.EventID())
		assert.Equal(t, followerID.String(), event.AggregateID())
	})

	t.Run("fails with zero followerID", func(t *testing.T) {
		t.Parallel()

		var zeroID identity.UserID
		_, err := identity.NewFollow(zeroID, followedID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "follower ID is required")
	})

	t.Run("fails with zero followedID", func(t *testing.T) {
		t.Parallel()

		var zeroID identity.UserID
		_, err := identity.NewFollow(followerID, zeroID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "followed ID is required")
	})

	t.Run("fails when trying to follow self", func(t *testing.T) {
		t.Parallel()

		sameID := identity.NewUserID()
		_, err := identity.NewFollow(sameID, sameID)

		require.ErrorIs(t, err, identity.ErrCannotFollowSelf)
	})
}

func TestReconstructFollow(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()
	createdAt := time.Now().UTC().Add(-24 * time.Hour)

	follow := identity.ReconstructFollow(followerID, followedID, createdAt)

	assert.Equal(t, followerID, follow.FollowerID())
	assert.Equal(t, followedID, follow.FollowedID())
	assert.Equal(t, createdAt, follow.CreatedAt())
	assert.Empty(t, follow.Events()) // No events on reconstruction
}

func TestFollow_Getters(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	follow, err := identity.NewFollow(followerID, followedID)
	require.NoError(t, err)

	t.Run("FollowerID returns correct ID", func(t *testing.T) {
		assert.Equal(t, followerID, follow.FollowerID())
		assert.True(t, follow.FollowerID().Equals(followerID))
	})

	t.Run("FollowedID returns correct ID", func(t *testing.T) {
		assert.Equal(t, followedID, follow.FollowedID())
		assert.True(t, follow.FollowedID().Equals(followedID))
	})

	t.Run("CreatedAt returns valid timestamp", func(t *testing.T) {
		assert.False(t, follow.CreatedAt().IsZero())
		assert.True(t, follow.CreatedAt().Before(time.Now().UTC().Add(time.Second)))
	})
}

func TestFollow_Events(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	t.Run("Events returns all domain events", func(t *testing.T) {
		t.Parallel()

		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		events := follow.Events()
		assert.Len(t, events, 1)
	})

	t.Run("ClearEvents removes all events", func(t *testing.T) {
		t.Parallel()

		follow, err := identity.NewFollow(followerID, followedID)
		require.NoError(t, err)

		assert.Len(t, follow.Events(), 1)

		follow.ClearEvents()
		assert.Empty(t, follow.Events())
	})
}

func TestFollow_Matches(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()
	otherID := identity.NewUserID()

	follow, err := identity.NewFollow(followerID, followedID)
	require.NoError(t, err)

	t.Run("matches exact follower and followed IDs", func(t *testing.T) {
		assert.True(t, follow.Matches(followerID, followedID))
	})

	t.Run("does not match different follower ID", func(t *testing.T) {
		assert.False(t, follow.Matches(otherID, followedID))
	})

	t.Run("does not match different followed ID", func(t *testing.T) {
		assert.False(t, follow.Matches(followerID, otherID))
	})

	t.Run("does not match both IDs different", func(t *testing.T) {
		assert.False(t, follow.Matches(otherID, otherID))
	})

	t.Run("does not match reversed IDs", func(t *testing.T) {
		// Follow relationship is directional
		assert.False(t, follow.Matches(followedID, followerID))
	})
}

func TestFollowEventFields(t *testing.T) {
	t.Parallel()

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	t.Run("UserFollowed event contains correct data", func(t *testing.T) {
		t.Parallel()

		event := identity.NewUserFollowed(followerID, followedID)

		assert.Equal(t, "identity.user.followed", event.EventType())
		assert.Equal(t, followerID, event.FollowerID)
		assert.Equal(t, followedID, event.FollowedID)
		assert.Equal(t, followerID.String(), event.AggregateID())
		assert.NotEmpty(t, event.EventID())
		assert.False(t, event.OccurredAt().IsZero())
		assert.WithinDuration(t, time.Now().UTC(), event.OccurredAt(), 2*time.Second)
	})

	t.Run("UserUnfollowed event contains correct data", func(t *testing.T) {
		t.Parallel()

		event := identity.NewUserUnfollowed(followerID, followedID)

		assert.Equal(t, "identity.user.unfollowed", event.EventType())
		assert.Equal(t, followerID, event.FollowerID)
		assert.Equal(t, followedID, event.FollowedID)
		assert.Equal(t, followerID.String(), event.AggregateID())
		assert.NotEmpty(t, event.EventID())
		assert.False(t, event.OccurredAt().IsZero())
		assert.WithinDuration(t, time.Now().UTC(), event.OccurredAt(), 2*time.Second)
	})

	t.Run("UserFollowed and UserUnfollowed have unique event IDs", func(t *testing.T) {
		t.Parallel()

		event1 := identity.NewUserFollowed(followerID, followedID)
		event2 := identity.NewUserFollowed(followerID, followedID)
		event3 := identity.NewUserUnfollowed(followerID, followedID)

		// Each event should have a unique ID even with same data
		assert.NotEqual(t, event1.EventID(), event2.EventID())
		assert.NotEqual(t, event1.EventID(), event3.EventID())
		assert.NotEqual(t, event2.EventID(), event3.EventID())
	})
}

func TestFollow_MultipleFollowRelationships(t *testing.T) {
	t.Parallel()

	user1 := identity.NewUserID()
	user2 := identity.NewUserID()
	user3 := identity.NewUserID()

	t.Run("allows bidirectional follows between same users", func(t *testing.T) {
		t.Parallel()

		// User1 follows User2
		follow1, err1 := identity.NewFollow(user1, user2)
		require.NoError(t, err1)

		// User2 follows User1 (reverse direction should be allowed)
		follow2, err2 := identity.NewFollow(user2, user1)
		require.NoError(t, err2)

		// These are different relationships
		assert.False(t, follow1.Matches(user2, user1))
		assert.False(t, follow2.Matches(user1, user2))
		assert.True(t, follow1.Matches(user1, user2))
		assert.True(t, follow2.Matches(user2, user1))
	})

	t.Run("allows same follower to follow multiple users", func(t *testing.T) {
		t.Parallel()

		// User1 follows User2
		follow1, err1 := identity.NewFollow(user1, user2)
		require.NoError(t, err1)

		// User1 also follows User3
		follow2, err2 := identity.NewFollow(user1, user3)
		require.NoError(t, err2)

		assert.True(t, follow1.FollowerID().Equals(user1))
		assert.True(t, follow2.FollowerID().Equals(user1))
		assert.False(t, follow1.FollowedID().Equals(follow2.FollowedID()))
	})

	t.Run("allows same user to be followed by multiple users", func(t *testing.T) {
		t.Parallel()

		// User1 follows User3
		follow1, err1 := identity.NewFollow(user1, user3)
		require.NoError(t, err1)

		// User2 also follows User3
		follow2, err2 := identity.NewFollow(user2, user3)
		require.NoError(t, err2)

		assert.True(t, follow1.FollowedID().Equals(user3))
		assert.True(t, follow2.FollowedID().Equals(user3))
		assert.False(t, follow1.FollowerID().Equals(follow2.FollowerID()))
	})
}
