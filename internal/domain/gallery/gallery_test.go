package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewImage(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	metadata, _ := gallery.NewImageMetadata("Test Image", "Description", "image/jpeg", 1024, 800, 600)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		img, err := gallery.NewImage(ownerID, metadata)
		require.NoError(t, err)
		assert.NotNil(t, img)
		assert.NotEmpty(t, img.ID())
		assert.Equal(t, ownerID, img.OwnerID())
		assert.Equal(t, metadata, img.Metadata())
		assert.Equal(t, gallery.VisibilityPrivate, img.Visibility())
		assert.Equal(t, gallery.StatusProcessing, img.Status())
		assert.Equal(t, gallery.ScanStatusPending, img.ScanStatus())
		assert.Empty(t, img.Variants())
		assert.Empty(t, img.Tags())
		assert.Zero(t, img.ViewCount())
		assert.Zero(t, img.LikeCount())
		assert.Zero(t, img.CommentCount())
		assert.WithinDuration(t, time.Now(), img.CreatedAt(), time.Second)
		assert.Len(t, img.Events(), 1)
	})

	t.Run("Error_MissingOwner", func(t *testing.T) {
		t.Parallel()
		_, err := gallery.NewImage(identity.UserID{}, metadata)
		assert.Error(t, err)
	})
}

func TestNewImageWithID(t *testing.T) {
	t.Parallel()

	id := gallery.NewImageID()
	ownerID := identity.NewUserID()
	metadata, _ := gallery.NewImageMetadata("Test Image", "Description", "image/jpeg", 1024, 800, 600)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		img, err := gallery.NewImageWithID(id, ownerID, metadata)
		require.NoError(t, err)
		assert.Equal(t, id, img.ID())
	})

	t.Run("Error_MissingID", func(t *testing.T) {
		t.Parallel()
		_, err := gallery.NewImageWithID(gallery.ImageID{}, ownerID, metadata)
		assert.Error(t, err)
	})
}

func TestReconstructImage(t *testing.T) {
	t.Parallel()

	id := gallery.NewImageID()
	ownerID := identity.NewUserID()
	metadata, _ := gallery.NewImageMetadata("Test", "Desc", "image/png", 2048, 100, 100)
	now := time.Now()

	img := gallery.ReconstructImage(
		id, ownerID, metadata, gallery.VisibilityPublic, gallery.StatusActive, gallery.ScanStatusClean,
		nil, nil, nil, 10, 5, 2, now, now,
	)

	assert.Equal(t, id, img.ID())
	assert.Equal(t, ownerID, img.OwnerID())
	assert.Equal(t, gallery.VisibilityPublic, img.Visibility())
	assert.Equal(t, gallery.StatusActive, img.Status())
	assert.Equal(t, int64(10), img.ViewCount())
	assert.Equal(t, int64(5), img.LikeCount())
	assert.Equal(t, int64(2), img.CommentCount())
}

func TestImage_AddVariant(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())

	v, _ := gallery.NewImageVariant(gallery.VariantTypeThumbnail, "path/thumb.jpg", 100, 100, 100)
	err := img.AddVariant(v)
	require.NoError(t, err)
	assert.Len(t, img.Variants(), 1)
	assert.True(t, img.HasVariant(gallery.VariantTypeThumbnail))

	// Duplicate variant
	err = img.AddVariant(v)
	assert.ErrorIs(t, err, gallery.ErrVariantExists)
}

func TestImage_AddTag(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())
	tag, _ := gallery.NewTag("Nature")

	err := img.AddTag(tag)
	require.NoError(t, err)
	assert.True(t, img.HasTag(tag))

	// Duplicate tag
	err = img.AddTag(tag)
	assert.ErrorIs(t, err, gallery.ErrTagAlreadyExists)

	// Max tags
	for i := 0; i < gallery.MaxTagsPerImage; i++ {
		t, _ := gallery.NewTag("Tag" + string(rune(i)))
		_ = img.AddTag(t)
	}
	err = img.AddTag(tag) // Should fail (already exists or max reached if we added unique ones)
	// Wait, we added duplicate "Nature" before loop.
	// Let's reset.
	img, _ = gallery.NewImage(identity.NewUserID(), createMetadata())
	for i := 0; i < gallery.MaxTagsPerImage; i++ {
		// Ensure unique tag names
		tagName := "Tag" + generateRandomString(5) // Simplified unique name
		// Actually, generateRandomString is not available.
		// Just use index.
		tName := "Tag" + string(rune(65+i)) // A, B, C...
		tag, _ := gallery.NewTag(tName)
		_ = img.AddTag(tag)
	}
	newTag, _ := gallery.NewTag("Overflow")
	err = img.AddTag(newTag)
	assert.ErrorIs(t, err, gallery.ErrTooManyTags)
}

func TestImage_RemoveTag(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())
	tag, _ := gallery.NewTag("Nature")
	_ = img.AddTag(tag)

	err := img.RemoveTag(tag)
	require.NoError(t, err)
	assert.False(t, img.HasTag(tag))

	// Remove non-existent
	err = img.RemoveTag(tag)
	require.NoError(t, err)
}

func TestImage_StatusTransitions(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())

	// Mark Active
	err := img.MarkAsActive()
	require.NoError(t, err)
	assert.Equal(t, gallery.StatusActive, img.Status())

	// Flag
	err = img.Flag()
	require.NoError(t, err)
	assert.Equal(t, gallery.StatusFlagged, img.Status())
	assert.True(t, img.IsFlagged())

	// Delete
	err = img.MarkAsDeleted() // Cannot delete flagged
	assert.ErrorIs(t, err, gallery.ErrCannotDeleteFlagged)

	// Unflag (hack via reconstruction or just creating new image for delete test)
	img2, _ := gallery.NewImage(identity.NewUserID(), createMetadata())
	img2.MarkAsActive()
	err = img2.MarkAsDeleted()
	require.NoError(t, err)
	assert.True(t, img2.IsDeleted())
}

func TestImage_UpdateVisibility(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())

	// Processing image cannot change visibility
	err := img.UpdateVisibility(gallery.VisibilityPublic)
	assert.ErrorIs(t, err, gallery.ErrImageProcessing)

	img.MarkAsActive()
	err = img.UpdateVisibility(gallery.VisibilityPublic)
	require.NoError(t, err)
	assert.Equal(t, gallery.VisibilityPublic, img.Visibility())
}

func TestImage_ChangeOwner(t *testing.T) {
	t.Parallel()
	owner1 := identity.NewUserID()
	owner2 := identity.NewUserID()
	img, _ := gallery.NewImage(owner1, createMetadata())

	err := img.ChangeOwner(owner2)
	require.NoError(t, err)
	assert.Equal(t, owner2, img.OwnerID())
	assert.True(t, img.IsOwnedBy(owner2))
}

func TestImage_IPFS(t *testing.T) {
	t.Parallel()
	img, _ := gallery.NewImage(identity.NewUserID(), createMetadata())

	meta, _ := gallery.NewIPFSMetadata("QmHash", 100, time.Now())
	err := img.SetIPFSMetadata(meta)
	require.NoError(t, err)
	assert.True(t, img.HasIPFS())
	assert.Equal(t, meta, *img.IPFSMetadata())

	err = img.ClearIPFSMetadata()
	require.NoError(t, err)
	assert.False(t, img.HasIPFS())
}

func createMetadata() gallery.ImageMetadata {
	m, _ := gallery.NewImageMetadata("Title", "Desc", "image/jpeg", 100, 100, 100)
	return m
}

func generateRandomString(n int) string {
	// Dummy implementation for test
	return "random"
}
