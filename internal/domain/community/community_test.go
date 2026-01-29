package community_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewGroup(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	name, _ := community.NewGroupName("Test Group")
	slug, _ := community.NewGroupSlug("test-group")
	groupType := community.GroupTypePublic

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		group, err := community.NewGroup(ownerID, name, slug, groupType)
		require.NoError(t, err)
		assert.NotNil(t, group)
		assert.NotEmpty(t, group.ID())
		assert.Equal(t, ownerID, group.OwnerID())
		assert.Equal(t, name, group.Name())
		assert.Equal(t, slug, group.Slug())
		assert.Equal(t, groupType, group.GroupType())
		assert.Equal(t, 1, group.MemberCount()) // Owner is member
		assert.WithinDuration(t, time.Now(), group.CreatedAt(), time.Second)
		assert.Len(t, group.Events(), 1) // GroupCreated event
	})

	t.Run("Error_MissingOwner", func(t *testing.T) {
		t.Parallel()
		_, err := community.NewGroup(identity.UserID{}, name, slug, groupType)
		assert.Error(t, err)
	})
}

func TestReconstructGroup(t *testing.T) {
	t.Parallel()

	id := community.NewGroupID()
	name, _ := community.NewGroupName("Test")
	slug, _ := community.NewGroupSlug("test")
	ownerID := identity.NewUserID()
	now := time.Now()

	group := community.ReconstructGroup(
		id, name, slug, "Desc", community.GroupTypePrivate, ownerID,
		community.DefaultGroupSettings(), 5, 10, 2, nil, now, now,
	)

	assert.Equal(t, id, group.ID())
	assert.Equal(t, name, group.Name())
	assert.Equal(t, "Desc", group.Description())
	assert.Equal(t, 5, group.MemberCount())
	assert.Equal(t, 10, group.ImageCount())
	assert.Equal(t, 2, group.AlbumCount())
}

func TestGroup_UpdateDescription(t *testing.T) {
	t.Parallel()
	g, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePublic)

	err := g.UpdateDescription("New description")
	require.NoError(t, err)
	assert.Equal(t, "New description", g.Description())

	// Too long
	longDesc := make([]byte, community.MaxDescriptionLength+1)
	err = g.UpdateDescription(string(longDesc))
	assert.ErrorIs(t, err, community.ErrGroupDescTooLong)
}

func TestGroup_UpdateSettings(t *testing.T) {
	t.Parallel()
	g, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePublic)

	newSettings, _ := community.NewGroupSettings(true, false, false, 50)
	err := g.UpdateSettings(newSettings)
	require.NoError(t, err)
	assert.Equal(t, newSettings, g.Settings())

	// No change
	err = g.UpdateSettings(newSettings)
	require.NoError(t, err)
}

func TestGroup_Counts(t *testing.T) {
	t.Parallel()
	g, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePublic)
	// Initial member count is 1 (owner)

	g.IncrementMemberCount()
	assert.Equal(t, 2, g.MemberCount())

	g.DecrementMemberCount()
	assert.Equal(t, 1, g.MemberCount())

	g.IncrementImageCount()
	assert.Equal(t, 1, g.ImageCount())

	g.DecrementImageCount()
	assert.Equal(t, 0, g.ImageCount())

	g.IncrementAlbumCount()
	assert.Equal(t, 1, g.AlbumCount())

	g.DecrementAlbumCount()
	assert.Equal(t, 0, g.AlbumCount())
}

func TestGroup_SetCoverImage(t *testing.T) {
	t.Parallel()
	g, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePublic)

	imageID := gallery.NewImageID()
	g.SetCoverImage(&imageID)
	assert.Equal(t, &imageID, g.CoverImageID())

	g.SetCoverImage(nil)
	assert.Nil(t, g.CoverImageID())
}

func TestGroup_TypeChecks(t *testing.T) {
	t.Parallel()
	gPublic, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePublic)
	assert.True(t, gPublic.IsPublic())
	assert.True(t, gPublic.IsDiscoverable())
	assert.True(t, gPublic.AllowsInstantJoin())

	gPrivate, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypePrivate)
	assert.True(t, gPrivate.IsPrivate())
	assert.False(t, gPrivate.IsDiscoverable())
	assert.False(t, gPrivate.AllowsInstantJoin())

	gInvite, _ := community.NewGroup(identity.NewUserID(), createName(), createSlug(), community.GroupTypeInviteOnly)
	assert.True(t, gInvite.IsInviteOnly())
	assert.True(t, gInvite.IsDiscoverable())
	assert.False(t, gInvite.AllowsInstantJoin())
}

func createName() community.GroupName {
	n, _ := community.NewGroupName("Test Group")
	return n
}

func createSlug() community.GroupSlug {
	s, _ := community.NewGroupSlug("test-group")
	return s
}
