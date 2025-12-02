package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewAlbum(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()

	t.Run("valid album", func(t *testing.T) {
		t.Parallel()

		album, err := gallery.NewAlbum(ownerID, "My Album")

		require.NoError(t, err)
		assert.Equal(t, "My Album", album.Title())
		assert.Equal(t, gallery.VisibilityPrivate, album.Visibility())
		assert.Nil(t, album.CoverImageID())
		assert.Equal(t, 0, album.ImageCount())
	})

	t.Run("empty title", func(t *testing.T) {
		t.Parallel()

		_, err := gallery.NewAlbum(ownerID, "")

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrAlbumTitleRequired)
	})
}

func TestAlbum_UpdateTitle(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Original")

	err := album.UpdateTitle("Updated")

	require.NoError(t, err)
	assert.Equal(t, "Updated", album.Title())
}

func TestAlbum_SetCoverImage(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")
	imageID := gallery.NewImageID()

	album.SetCoverImage(&imageID)

	assert.NotNil(t, album.CoverImageID())
	assert.True(t, album.CoverImageID().Equals(imageID))
}

func TestAlbum_ImageCount(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	album.IncrementImageCount()
	album.IncrementImageCount()
	assert.Equal(t, 2, album.ImageCount())

	album.DecrementImageCount()
	assert.Equal(t, 1, album.ImageCount())
}

func TestAlbum_Getters(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	assert.False(t, album.ID().IsZero())
	assert.Equal(t, ownerID, album.OwnerID())
	assert.Equal(t, "", album.Description())
	assert.False(t, album.CreatedAt().IsZero())
	assert.False(t, album.UpdatedAt().IsZero())
	assert.NotNil(t, album.Events())
}

func TestAlbum_UpdateDescription(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	err := album.UpdateDescription("New description")

	require.NoError(t, err)
	assert.Equal(t, "New description", album.Description())
}

func TestAlbum_UpdateVisibility(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	err := album.UpdateVisibility(gallery.VisibilityPublic)

	require.NoError(t, err)
	assert.Equal(t, gallery.VisibilityPublic, album.Visibility())
}

func TestAlbum_IsOwnedBy(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	assert.True(t, album.IsOwnedBy(ownerID))
	assert.False(t, album.IsOwnedBy(identity.NewUserID()))
}

func TestAlbum_IsEmpty(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	assert.True(t, album.IsEmpty())

	album.IncrementImageCount()
	assert.False(t, album.IsEmpty())
}

func TestAlbum_ClearEvents(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	assert.Len(t, album.Events(), 1)

	album.ClearEvents()
	assert.Len(t, album.Events(), 0)
}
