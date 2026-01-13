package community

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewGroupAlbum(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()

	t.Run("valid album - public", func(t *testing.T) {
		album, err := NewGroupAlbum(groupID, creatorID, "My Album", true)

		require.NoError(t, err)
		assert.False(t, album.ID().IsZero())
		assert.True(t, album.GroupID().Equals(groupID))
		assert.True(t, album.CreatedBy().Equals(creatorID))
		assert.Equal(t, "My Album", album.Title())
		assert.Equal(t, "", album.Description())
		assert.Nil(t, album.CoverImageID())
		assert.Equal(t, 0, album.ImageCount())
		assert.True(t, album.IsPublic())
		assert.False(t, album.CreatedAt().IsZero())
		assert.False(t, album.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, album.Events(), 1)
		event, ok := album.Events()[0].(*GroupAlbumCreated)
		assert.True(t, ok)
		assert.True(t, event.GroupAlbumID.Equals(album.ID()))
		assert.True(t, event.GroupID.Equals(groupID))
		assert.True(t, event.CreatedBy.Equals(creatorID))
		assert.Equal(t, "My Album", event.Title)
	})

	t.Run("valid album - private", func(t *testing.T) {
		album, err := NewGroupAlbum(groupID, creatorID, "Private Album", false)

		require.NoError(t, err)
		assert.False(t, album.IsPublic())
	})

	t.Run("title with whitespace is trimmed", func(t *testing.T) {
		album, err := NewGroupAlbum(groupID, creatorID, "  Trimmed Album  ", true)

		require.NoError(t, err)
		assert.Equal(t, "Trimmed Album", album.Title())
	})

	t.Run("invalid - zero group ID", func(t *testing.T) {
		_, err := NewGroupAlbum(GroupID{}, creatorID, "My Album", true)
		require.Error(t, err)
	})

	t.Run("invalid - zero creator ID", func(t *testing.T) {
		_, err := NewGroupAlbum(groupID, identity.UserID{}, "My Album", true)
		require.Error(t, err)
	})

	t.Run("invalid - empty title", func(t *testing.T) {
		_, err := NewGroupAlbum(groupID, creatorID, "", true)
		require.ErrorIs(t, err, ErrGroupAlbumTitleRequired)
	})

	t.Run("invalid - whitespace-only title", func(t *testing.T) {
		_, err := NewGroupAlbum(groupID, creatorID, "   ", true)
		require.ErrorIs(t, err, ErrGroupAlbumTitleRequired)
	})

	t.Run("invalid - title too long", func(t *testing.T) {
		longTitle := strings.Repeat("a", MaxAlbumTitleLength+1)
		_, err := NewGroupAlbum(groupID, creatorID, longTitle, true)
		require.ErrorIs(t, err, ErrGroupAlbumTitleTooLong)
	})

	t.Run("valid - title at max length", func(t *testing.T) {
		maxTitle := strings.Repeat("a", MaxAlbumTitleLength)
		album, err := NewGroupAlbum(groupID, creatorID, maxTitle, true)
		require.NoError(t, err)
		assert.Equal(t, MaxAlbumTitleLength, len(album.Title()))
	})
}

func TestReconstructGroupAlbum(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()
	albumID := NewGroupAlbumID()
	imageID := gallery.NewImageID()

	// Create a full album reconstruction
	album := ReconstructGroupAlbum(
		albumID,
		groupID,
		creatorID,
		"Reconstructed Album",
		"A description",
		&imageID,
		10,
		true,
		time.Now().UTC(),
		time.Now().UTC(),
	)

	assert.True(t, album.ID().Equals(albumID))
	assert.True(t, album.GroupID().Equals(groupID))
	assert.True(t, album.CreatedBy().Equals(creatorID))
	assert.Equal(t, "Reconstructed Album", album.Title())
	assert.Equal(t, "A description", album.Description())
	assert.NotNil(t, album.CoverImageID())
	assert.True(t, album.CoverImageID().Equals(imageID))
	assert.Equal(t, 10, album.ImageCount())
	assert.True(t, album.IsPublic())
	assert.Len(t, album.Events(), 0) // No events on reconstruction
}

func TestGroupAlbum_UpdateTitle(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()

	t.Run("successful title update", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)
		album.ClearEvents()

		err := album.UpdateTitle("New Title")

		require.NoError(t, err)
		assert.Equal(t, "New Title", album.Title())
		assert.False(t, album.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, album.Events(), 1)
		event, ok := album.Events()[0].(*GroupAlbumUpdated)
		assert.True(t, ok)
		assert.Equal(t, "Old Title", event.OldTitle)
		assert.Equal(t, "New Title", event.NewTitle)
	})

	t.Run("title trimmed", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)
		album.ClearEvents()

		err := album.UpdateTitle("  New Title  ")

		require.NoError(t, err)
		assert.Equal(t, "New Title", album.Title())
	})

	t.Run("no change - no event", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Same Title", true)
		album.ClearEvents()

		err := album.UpdateTitle("Same Title")

		require.NoError(t, err)
		assert.Len(t, album.Events(), 0)
	})

	t.Run("invalid - empty title", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)

		err := album.UpdateTitle("")
		require.ErrorIs(t, err, ErrGroupAlbumTitleRequired)
		assert.Equal(t, "Old Title", album.Title()) // Unchanged
	})

	t.Run("invalid - whitespace-only title", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)

		err := album.UpdateTitle("   ")
		require.ErrorIs(t, err, ErrGroupAlbumTitleRequired)
	})

	t.Run("invalid - title too long", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)
		longTitle := strings.Repeat("a", MaxAlbumTitleLength+1)

		err := album.UpdateTitle(longTitle)
		require.ErrorIs(t, err, ErrGroupAlbumTitleTooLong)
		assert.Equal(t, "Old Title", album.Title()) // Unchanged
	})

	t.Run("valid - title at max length", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "Old Title", true)
		maxTitle := strings.Repeat("a", MaxAlbumTitleLength)

		err := album.UpdateTitle(maxTitle)
		require.NoError(t, err)
		assert.Equal(t, MaxAlbumTitleLength, len(album.Title()))
	})
}

func TestGroupAlbum_UpdateDescription(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()

	t.Run("successful description update", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		err := album.UpdateDescription("A great collection of photos")

		require.NoError(t, err)
		assert.Equal(t, "A great collection of photos", album.Description())
		assert.False(t, album.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, album.Events(), 1)
		event, ok := album.Events()[0].(*GroupAlbumUpdated)
		assert.True(t, ok)
		assert.NotNil(t, event.NewDescription)
		assert.Equal(t, "A great collection of photos", *event.NewDescription)
	})

	t.Run("description trimmed", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		err := album.UpdateDescription("  Description with spaces  ")

		require.NoError(t, err)
		assert.Equal(t, "Description with spaces", album.Description())
	})

	t.Run("empty description is valid", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		_ = album.UpdateDescription("Old description")
		album.ClearEvents()

		err := album.UpdateDescription("")

		require.NoError(t, err)
		assert.Equal(t, "", album.Description())
		assert.Len(t, album.Events(), 1)
	})

	t.Run("no change - no event", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		_ = album.UpdateDescription("Same description")
		album.ClearEvents()

		err := album.UpdateDescription("Same description")

		require.NoError(t, err)
		assert.Len(t, album.Events(), 0)
	})

	t.Run("invalid - description too long", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		longDesc := strings.Repeat("a", MaxAlbumDescriptionLength+1)

		err := album.UpdateDescription(longDesc)
		require.ErrorIs(t, err, ErrGroupAlbumDescTooLong)
		assert.Equal(t, "", album.Description()) // Unchanged
	})

	t.Run("valid - description at max length", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		maxDesc := strings.Repeat("a", MaxAlbumDescriptionLength)

		err := album.UpdateDescription(maxDesc)
		require.NoError(t, err)
		assert.Equal(t, MaxAlbumDescriptionLength, len(album.Description()))
	})
}

func TestGroupAlbum_SetCoverImage(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()
	imageID1 := gallery.NewImageID()
	imageID2 := gallery.NewImageID()

	t.Run("set cover image", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		album.SetCoverImage(&imageID1)

		assert.NotNil(t, album.CoverImageID())
		assert.True(t, album.CoverImageID().Equals(imageID1))
		assert.False(t, album.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, album.Events(), 1)
		event, ok := album.Events()[0].(*GroupAlbumUpdated)
		assert.True(t, ok)
		assert.NotNil(t, event.NewCoverImageID)
		assert.True(t, event.NewCoverImageID.Equals(imageID1))
	})

	t.Run("change cover image", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.SetCoverImage(&imageID1)
		album.ClearEvents()

		album.SetCoverImage(&imageID2)

		assert.NotNil(t, album.CoverImageID())
		assert.True(t, album.CoverImageID().Equals(imageID2))
		assert.Len(t, album.Events(), 1)
	})

	t.Run("remove cover image", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.SetCoverImage(&imageID1)
		album.ClearEvents()

		album.SetCoverImage(nil)

		assert.Nil(t, album.CoverImageID())
		assert.Len(t, album.Events(), 1)
	})

	t.Run("no change - same image - no event", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.SetCoverImage(&imageID1)
		album.ClearEvents()

		album.SetCoverImage(&imageID1)

		assert.Len(t, album.Events(), 0)
	})

	t.Run("no change - both nil - no event", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		album.SetCoverImage(nil)

		assert.Len(t, album.Events(), 0)
	})
}

func TestGroupAlbum_SetVisibility(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()

	t.Run("change from public to private", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		album.SetVisibility(false)

		assert.False(t, album.IsPublic())
		assert.False(t, album.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, album.Events(), 1)
		event, ok := album.Events()[0].(*GroupAlbumUpdated)
		assert.True(t, ok)
		assert.NotNil(t, event.NewIsPublic)
		assert.False(t, *event.NewIsPublic)
	})

	t.Run("change from private to public", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", false)
		album.ClearEvents()

		album.SetVisibility(true)

		assert.True(t, album.IsPublic())
		assert.Len(t, album.Events(), 1)
	})

	t.Run("no change - no event", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.ClearEvents()

		album.SetVisibility(true)

		assert.Len(t, album.Events(), 0)
	})
}

func TestGroupAlbum_ImageCount(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()

	t.Run("increment image count", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)

		assert.Equal(t, 0, album.ImageCount())

		album.IncrementImageCount()
		assert.Equal(t, 1, album.ImageCount())

		album.IncrementImageCount()
		assert.Equal(t, 2, album.ImageCount())
	})

	t.Run("decrement image count", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.IncrementImageCount()
		album.IncrementImageCount()
		album.IncrementImageCount()

		assert.Equal(t, 3, album.ImageCount())

		album.DecrementImageCount()
		assert.Equal(t, 2, album.ImageCount())

		album.DecrementImageCount()
		assert.Equal(t, 1, album.ImageCount())
	})

	t.Run("decrement when zero does not go negative", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)

		assert.Equal(t, 0, album.ImageCount())

		album.DecrementImageCount()
		assert.Equal(t, 0, album.ImageCount()) // Still zero
	})
}

func TestGroupAlbum_HelperMethods(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()
	otherUserID := identity.NewUserID()

	t.Run("IsOwnedBy - true for creator", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		assert.True(t, album.IsOwnedBy(creatorID))
	})

	t.Run("IsOwnedBy - false for other user", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		assert.False(t, album.IsOwnedBy(otherUserID))
	})

	t.Run("IsEmpty - true when no images", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		assert.True(t, album.IsEmpty())
	})

	t.Run("IsEmpty - false when has images", func(t *testing.T) {
		album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)
		album.IncrementImageCount()
		assert.False(t, album.IsEmpty())
	})
}

func TestGroupAlbum_ClearEvents(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	creatorID := identity.NewUserID()
	album, _ := NewGroupAlbum(groupID, creatorID, "My Album", true)

	assert.Len(t, album.Events(), 1)

	album.ClearEvents()
	assert.Len(t, album.Events(), 0)
}
