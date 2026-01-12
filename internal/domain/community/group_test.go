package community

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewGroup(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	name, _ := NewGroupName("Photography Lovers")
	slug, _ := NewGroupSlug("photography-lovers")

	t.Run("valid public group", func(t *testing.T) {
		group, err := NewGroup(ownerID, name, slug, GroupTypePublic)

		require.NoError(t, err)
		assert.False(t, group.ID().IsZero())
		assert.Equal(t, name, group.Name())
		assert.Equal(t, slug, group.Slug())
		assert.Equal(t, GroupTypePublic, group.GroupType())
		assert.True(t, group.OwnerID().Equals(ownerID))
		assert.Equal(t, 1, group.MemberCount()) // Owner counted as first member
		assert.Equal(t, 0, group.ImageCount())
		assert.Equal(t, 0, group.AlbumCount())
		assert.Nil(t, group.CoverImageID())
		assert.False(t, group.CreatedAt().IsZero())
		assert.False(t, group.UpdatedAt().IsZero())

		// Check default settings
		assert.False(t, group.Settings().RequireApproval())
		assert.True(t, group.Settings().AllowMemberInvites())
		assert.True(t, group.Settings().AllowMemberAlbums())
		assert.Equal(t, 0, group.Settings().MaxMembers())

		// Check event emitted
		assert.Len(t, group.Events(), 1)
		event, ok := group.Events()[0].(*GroupCreated)
		assert.True(t, ok)
		assert.Equal(t, group.ID(), event.GroupID)
		assert.True(t, event.OwnerID.Equals(ownerID))
	})

	t.Run("invalid - zero owner ID", func(t *testing.T) {
		_, err := NewGroup(identity.UserID{}, name, slug, GroupTypePublic)
		require.Error(t, err)
	})

	t.Run("invalid - empty name", func(t *testing.T) {
		_, err := NewGroup(ownerID, GroupName{}, slug, GroupTypePublic)
		require.ErrorIs(t, err, ErrGroupNameRequired)
	})

	t.Run("invalid - empty slug", func(t *testing.T) {
		_, err := NewGroup(ownerID, name, GroupSlug{}, GroupTypePublic)
		require.ErrorIs(t, err, ErrGroupSlugRequired)
	})

	t.Run("invalid - invalid group type", func(t *testing.T) {
		_, err := NewGroup(ownerID, name, slug, GroupType(999))
		require.ErrorIs(t, err, ErrInvalidGroupType)
	})
}

func TestGroup_UpdateDescription(t *testing.T) {
	t.Parallel()

	group := createTestGroup(t)
	group.ClearEvents()

	t.Run("valid description", func(t *testing.T) {
		err := group.UpdateDescription("A group for photography enthusiasts")
		require.NoError(t, err)
		assert.Equal(t, "A group for photography enthusiasts", group.Description())

		// Check event
		assert.Len(t, group.Events(), 1)
		event, ok := group.Events()[0].(*GroupDescriptionUpdated)
		assert.True(t, ok)
		assert.Equal(t, group.ID(), event.GroupID)
	})

	t.Run("empty description allowed", func(t *testing.T) {
		group.ClearEvents()
		err := group.UpdateDescription("")
		require.NoError(t, err)
		assert.Equal(t, "", group.Description())
	})

	t.Run("too long", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		err := group.UpdateDescription(longDesc)
		require.ErrorIs(t, err, ErrGroupDescTooLong)
	})

	t.Run("no change - no event", func(t *testing.T) {
		// First set the description
		_ = group.UpdateDescription("A group for photography enthusiasts")
		group.ClearEvents()
		// Now update with same description
		err := group.UpdateDescription("A group for photography enthusiasts")
		require.NoError(t, err)
		assert.Len(t, group.Events(), 0) // No event when no change
	})
}

func TestGroup_UpdateSettings(t *testing.T) {
	t.Parallel()

	group := createTestGroup(t)
	group.ClearEvents()

	newSettings, _ := NewGroupSettings(true, false, false, 50)

	err := group.UpdateSettings(newSettings)
	require.NoError(t, err)
	assert.True(t, group.Settings().Equals(newSettings))

	// Check event
	assert.Len(t, group.Events(), 1)
	event, ok := group.Events()[0].(*GroupSettingsUpdated)
	assert.True(t, ok)
	assert.Equal(t, group.ID(), event.GroupID)

	t.Run("no change - no event", func(t *testing.T) {
		group.ClearEvents()
		err := group.UpdateSettings(newSettings)
		require.NoError(t, err)
		assert.Len(t, group.Events(), 0)
	})
}

func TestGroup_SetCoverImage(t *testing.T) {
	t.Parallel()

	group := createTestGroup(t)
	group.ClearEvents()

	imageID := gallery.NewImageID()

	t.Run("set cover image", func(t *testing.T) {
		group.SetCoverImage(&imageID)
		assert.NotNil(t, group.CoverImageID())
		assert.True(t, group.CoverImageID().Equals(imageID))

		// Check event
		assert.Len(t, group.Events(), 1)
		event, ok := group.Events()[0].(*GroupCoverImageChanged)
		assert.True(t, ok)
		assert.Equal(t, group.ID(), event.GroupID)
	})

	t.Run("remove cover image", func(t *testing.T) {
		group.ClearEvents()
		group.SetCoverImage(nil)
		assert.Nil(t, group.CoverImageID())

		// Check event
		assert.Len(t, group.Events(), 1)
	})

	t.Run("no change - same image", func(t *testing.T) {
		group.SetCoverImage(&imageID)
		group.ClearEvents()
		group.SetCoverImage(&imageID)
		assert.Len(t, group.Events(), 0)
	})
}

func TestGroup_CountMethods(t *testing.T) {
	t.Parallel()

	group := createTestGroup(t)

	t.Run("member count", func(t *testing.T) {
		initialCount := group.MemberCount()

		group.IncrementMemberCount()
		assert.Equal(t, initialCount+1, group.MemberCount())

		group.IncrementMemberCount()
		assert.Equal(t, initialCount+2, group.MemberCount())

		group.DecrementMemberCount()
		assert.Equal(t, initialCount+1, group.MemberCount())

		// Can't go below zero
		for i := 0; i < 100; i++ {
			group.DecrementMemberCount()
		}
		assert.Equal(t, 0, group.MemberCount())
	})

	t.Run("image count", func(t *testing.T) {
		assert.Equal(t, 0, group.ImageCount())

		group.IncrementImageCount()
		assert.Equal(t, 1, group.ImageCount())

		group.IncrementImageCount()
		assert.Equal(t, 2, group.ImageCount())

		group.DecrementImageCount()
		assert.Equal(t, 1, group.ImageCount())

		group.DecrementImageCount()
		assert.Equal(t, 0, group.ImageCount())

		// Can't go below zero
		group.DecrementImageCount()
		assert.Equal(t, 0, group.ImageCount())
	})

	t.Run("album count", func(t *testing.T) {
		assert.Equal(t, 0, group.AlbumCount())

		group.IncrementAlbumCount()
		assert.Equal(t, 1, group.AlbumCount())

		group.DecrementAlbumCount()
		assert.Equal(t, 0, group.AlbumCount())

		// Can't go below zero
		group.DecrementAlbumCount()
		assert.Equal(t, 0, group.AlbumCount())
	})
}

func TestGroup_CanAcceptNewMembers(t *testing.T) {
	t.Parallel()

	t.Run("no limit", func(t *testing.T) {
		group := createTestGroup(t)
		assert.True(t, group.CanAcceptNewMembers())

		// Even with many members
		for i := 0; i < 1000; i++ {
			group.IncrementMemberCount()
		}
		assert.True(t, group.CanAcceptNewMembers())
	})

	t.Run("with limit - below", func(t *testing.T) {
		group := createTestGroup(t)
		settings, _ := NewGroupSettings(false, true, true, 5)
		_ = group.UpdateSettings(settings)

		assert.True(t, group.CanAcceptNewMembers()) // 1 < 5
	})

	t.Run("with limit - at limit", func(t *testing.T) {
		group := createTestGroup(t)
		settings, _ := NewGroupSettings(false, true, true, 5)
		_ = group.UpdateSettings(settings)

		for i := 1; i < 5; i++ {
			group.IncrementMemberCount()
		}
		assert.False(t, group.CanAcceptNewMembers()) // 5 == 5
	})

	t.Run("with limit - over limit", func(t *testing.T) {
		group := createTestGroup(t)
		settings, _ := NewGroupSettings(false, true, true, 5)
		_ = group.UpdateSettings(settings)

		for i := 1; i < 10; i++ {
			group.IncrementMemberCount()
		}
		assert.False(t, group.CanAcceptNewMembers()) // 10 > 5
	})
}

func TestGroup_HelperMethods(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	otherID := identity.NewUserID()
	group := createTestGroupWithOwner(t, ownerID)

	t.Run("IsOwnedBy", func(t *testing.T) {
		assert.True(t, group.IsOwnedBy(ownerID))
		assert.False(t, group.IsOwnedBy(otherID))
	})

	t.Run("group type checks", func(t *testing.T) {
		publicGroup := createTestGroupWithType(t, GroupTypePublic)
		assert.True(t, publicGroup.IsPublic())
		assert.False(t, publicGroup.IsPrivate())
		assert.False(t, publicGroup.IsInviteOnly())
		assert.True(t, publicGroup.IsDiscoverable())
		assert.True(t, publicGroup.AllowsInstantJoin())

		privateGroup := createTestGroupWithType(t, GroupTypePrivate)
		assert.False(t, privateGroup.IsPublic())
		assert.True(t, privateGroup.IsPrivate())
		assert.False(t, privateGroup.IsInviteOnly())
		assert.False(t, privateGroup.IsDiscoverable())
		assert.False(t, privateGroup.AllowsInstantJoin())

		inviteOnlyGroup := createTestGroupWithType(t, GroupTypeInviteOnly)
		assert.False(t, inviteOnlyGroup.IsPublic())
		assert.False(t, inviteOnlyGroup.IsPrivate())
		assert.True(t, inviteOnlyGroup.IsInviteOnly())
		assert.True(t, inviteOnlyGroup.IsDiscoverable())
		assert.False(t, inviteOnlyGroup.AllowsInstantJoin())
	})
}

func TestGroup_ClearEvents(t *testing.T) {
	t.Parallel()

	group := createTestGroup(t)
	assert.Len(t, group.Events(), 1) // Creation event

	group.ClearEvents()
	assert.Len(t, group.Events(), 0)
}

// Helper functions for tests

func createTestGroup(t *testing.T) *Group {
	t.Helper()
	return createTestGroupWithOwner(t, identity.NewUserID())
}

func createTestGroupWithOwner(t *testing.T, ownerID identity.UserID) *Group {
	t.Helper()
	name, _ := NewGroupName("Test Group")
	slug, _ := NewGroupSlug("test-group")
	group, err := NewGroup(ownerID, name, slug, GroupTypePublic)
	require.NoError(t, err)
	return group
}

func createTestGroupWithType(t *testing.T, groupType GroupType) *Group {
	t.Helper()
	ownerID := identity.NewUserID()
	name, _ := NewGroupName("Test Group")
	slug, _ := NewGroupSlug("test-group")
	group, err := NewGroup(ownerID, name, slug, groupType)
	require.NoError(t, err)
	return group
}
