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

	metadata, _ := gallery.NewImageMetadata(
		"Title",
		"Description",
		"photo.jpg",
		"image/jpeg",
		1920,
		1080,
		1024000,
		"key",
		"s3",
	)

	t.Run("valid image", func(t *testing.T) {
		t.Parallel()

		ownerID := identity.NewUserID()
		img, err := gallery.NewImage(ownerID, metadata)

		require.NoError(t, err)
		assert.False(t, img.ID().IsZero())
		assert.Equal(t, ownerID, img.OwnerID())
		assert.Equal(t, gallery.VisibilityPrivate, img.Visibility())
		assert.Equal(t, gallery.StatusProcessing, img.Status())
		assert.Empty(t, img.Variants())
		assert.Empty(t, img.Tags())
		assert.Len(t, img.Events(), 1)
	})

	t.Run("zero owner ID", func(t *testing.T) {
		t.Parallel()

		var zeroID identity.UserID
		_, err := gallery.NewImage(zeroID, metadata)

		require.Error(t, err)
	})
}

func TestImage_AddVariant(t *testing.T) {
	t.Parallel()

	variant, _ := gallery.NewImageVariant(
		gallery.VariantThumbnail,
		"thumb.jpg",
		150,
		150,
		1000,
		"jpeg",
	)

	t.Run("add first variant", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)

		err := img.AddVariant(variant)

		require.NoError(t, err)
		assert.Len(t, img.Variants(), 1)
		assert.True(t, img.HasVariant(gallery.VariantThumbnail))
	})

	t.Run("duplicate variant fails", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.AddVariant(variant)
		require.NoError(t, err)

		err = img.AddVariant(variant)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrVariantExists)
	})
}

func TestImage_GetVariant(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	variant, _ := gallery.NewImageVariant(
		gallery.VariantThumbnail,
		"thumb.jpg",
		150,
		150,
		1000,
		"jpeg",
	)
	err := img.AddVariant(variant)
	require.NoError(t, err)

	t.Run("existing variant", func(t *testing.T) {
		t.Parallel()

		v, err := img.GetVariant(gallery.VariantThumbnail)

		require.NoError(t, err)
		assert.Equal(t, gallery.VariantThumbnail, v.VariantType())
	})

	t.Run("non-existing variant", func(t *testing.T) {
		t.Parallel()

		_, err := img.GetVariant(gallery.VariantLarge)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrVariantNotFound)
	})
}

func TestImage_AddTag(t *testing.T) {
	t.Parallel()

	tag1, _ := gallery.NewTag("landscape")
	tag2, _ := gallery.NewTag("nature")

	t.Run("add first tag", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.AddTag(tag1)

		require.NoError(t, err)
		assert.Len(t, img.Tags(), 1)
		assert.True(t, img.HasTag(tag1))
	})

	t.Run("add multiple tags", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.AddTag(tag1)
		require.NoError(t, err)
		err = img.AddTag(tag2)

		require.NoError(t, err)
		assert.Len(t, img.Tags(), 2)
	})

	t.Run("duplicate tag fails", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.AddTag(tag1)
		require.NoError(t, err)

		err = img.AddTag(tag1)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrTagAlreadyExists)
	})

	t.Run("cannot tag deleted image", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.MarkAsDeleted()
		require.NoError(t, err)

		err = img.AddTag(tag1)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrCannotModifyDeleted)
	})
}

func TestImage_RemoveTag(t *testing.T) {
	t.Parallel()

	tag, _ := gallery.NewTag("landscape")

	t.Run("remove existing tag", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.AddTag(tag)
		require.NoError(t, err)

		err = img.RemoveTag(tag)

		require.NoError(t, err)
		assert.Empty(t, img.Tags())
	})

	t.Run("remove non-existing tag is idempotent", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.RemoveTag(tag)

		require.NoError(t, err)
	})
}

func TestImage_UpdateVisibility(t *testing.T) {
	t.Parallel()

	t.Run("change visibility", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.UpdateVisibility(gallery.VisibilityPublic)

		require.NoError(t, err)
		assert.Equal(t, gallery.VisibilityPublic, img.Visibility())
	})

	t.Run("cannot change visibility while processing", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)

		err := img.UpdateVisibility(gallery.VisibilityPublic)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrImageProcessing)
	})

	t.Run("cannot change visibility of deleted image", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.MarkAsDeleted()
		require.NoError(t, err)

		err = img.UpdateVisibility(gallery.VisibilityPublic)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrCannotModifyDeleted)
	})
}

func TestImage_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("delete active image", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.MarkAsDeleted()

		require.NoError(t, err)
		assert.Equal(t, gallery.StatusDeleted, img.Status())
		assert.True(t, img.IsDeleted())
	})

	t.Run("cannot delete flagged image", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.Flag()
		require.NoError(t, err)

		err = img.MarkAsDeleted()

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrCannotDeleteFlagged)
	})

	t.Run("delete is idempotent", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.MarkAsDeleted()
		require.NoError(t, err)

		err = img.MarkAsDeleted()

		require.NoError(t, err)
	})
}

func TestImage_UpdateMetadata(t *testing.T) {
	t.Parallel()

	t.Run("update title and description", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)

		err = img.UpdateMetadata("New Title", "New Description")

		require.NoError(t, err)
		assert.Equal(t, "New Title", img.Metadata().Title())
		assert.Equal(t, "New Description", img.Metadata().Description())
	})

	t.Run("cannot update deleted image", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)
		err := img.MarkAsActive()
		require.NoError(t, err)
		err = img.MarkAsDeleted()
		require.NoError(t, err)

		err = img.UpdateMetadata("New Title", "New Description")

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrCannotModifyDeleted)
	})
}

func TestImage_EngagementMetrics(t *testing.T) {
	t.Parallel()

	t.Run("increment views", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)

		img.IncrementViews()
		img.IncrementViews()

		assert.Equal(t, int64(2), img.ViewCount())
	})

	t.Run("set like count", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)

		img.SetLikeCount(10)

		assert.Equal(t, int64(10), img.LikeCount())
	})

	t.Run("set comment count", func(t *testing.T) {
		t.Parallel()
		img := createTestImage(t)

		img.SetCommentCount(5)

		assert.Equal(t, int64(5), img.CommentCount())
	})
}

func createTestImage(t *testing.T) *gallery.Image {
	t.Helper()

	metadata, err := gallery.NewImageMetadata(
		"Test Image",
		"Description",
		"test.jpg",
		"image/jpeg",
		1920,
		1080,
		1024000,
		"key",
		"s3",
	)
	require.NoError(t, err)

	ownerID := identity.NewUserID()
	img, err := gallery.NewImage(ownerID, metadata)
	require.NoError(t, err)
	img.ClearEvents()

	return img
}

func TestImage_Getters(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	require.NoError(t, img.MarkAsActive())

	assert.False(t, img.ID().IsZero())
	assert.False(t, img.OwnerID().IsZero())
	assert.NotNil(t, img.Metadata())
	assert.Equal(t, gallery.StatusActive, img.Status())
	assert.Equal(t, gallery.VisibilityPrivate, img.Visibility())
	assert.Empty(t, img.Variants())
	assert.Empty(t, img.Tags())
	assert.Equal(t, int64(0), img.ViewCount())
	assert.Equal(t, int64(0), img.LikeCount())
	assert.Equal(t, int64(0), img.CommentCount())
	assert.False(t, img.CreatedAt().IsZero())
	assert.False(t, img.UpdatedAt().IsZero())
}

func TestImage_IsViewable(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)

	assert.False(t, img.IsViewable())

	require.NoError(t, img.MarkAsActive())
	assert.True(t, img.IsViewable())
}

func TestImage_IsOwnedBy(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)

	assert.True(t, img.IsOwnedBy(img.OwnerID()))
	assert.False(t, img.IsOwnedBy(identity.NewUserID()))
}

func TestImage_Flag(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	require.NoError(t, img.MarkAsActive())

	err := img.Flag()

	require.NoError(t, err)
	assert.True(t, img.IsFlagged())
}

func TestReconstructImage(t *testing.T) {
	t.Parallel()

	metadata, _ := gallery.NewImageMetadata(
		"Title",
		"Desc",
		"file.jpg",
		"image/jpeg",
		1920,
		1080,
		1000,
		"key",
		"s3",
	)

	img := gallery.ReconstructImage(
		gallery.NewImageID(),
		identity.NewUserID(),
		metadata,
		gallery.VisibilityPublic,
		gallery.StatusActive,
		[]gallery.ImageVariant{},
		[]gallery.Tag{},
		10,
		5,
		3,
		time.Now(),
		time.Now(),
	)

	assert.NotNil(t, img)
	assert.Equal(t, gallery.VisibilityPublic, img.Visibility())
	assert.Equal(t, int64(10), img.ViewCount())
}

func TestImage_EventManagement(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)

	// Events were cleared in createTestImage, create a new one
	_ = img.MarkAsActive()
	assert.Len(t, img.Events(), 1)

	img.ClearEvents()
	assert.Empty(t, img.Events())
}

func TestImage_SetLikeCount_Negative(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	img.SetLikeCount(-10)

	assert.Equal(t, int64(0), img.LikeCount())
}

func TestImage_SetCommentCount_Negative(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	img.SetCommentCount(-5)

	assert.Equal(t, int64(0), img.CommentCount())
}

func TestImage_UpdateVisibility_NoChange(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	_ = img.MarkAsActive()

	err := img.UpdateVisibility(gallery.VisibilityPrivate)

	require.NoError(t, err)
	assert.Equal(t, gallery.VisibilityPrivate, img.Visibility())
}

func TestImage_UpdateMetadata_NoChange(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	_ = img.MarkAsActive()

	originalTitle := img.Metadata().Title()
	originalDesc := img.Metadata().Description()

	err := img.UpdateMetadata(originalTitle, originalDesc)

	require.NoError(t, err)
	assert.Equal(t, originalTitle, img.Metadata().Title())
}

func TestImage_MarkAsActive_Idempotent(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	err := img.MarkAsActive()
	require.NoError(t, err)

	err = img.MarkAsActive()

	require.NoError(t, err)
	assert.Equal(t, gallery.StatusActive, img.Status())
}

func TestImage_Flag_Idempotent(t *testing.T) {
	t.Parallel()

	img := createTestImage(t)
	err := img.MarkAsActive()
	require.NoError(t, err)
	err = img.Flag()
	require.NoError(t, err)

	err = img.Flag()

	require.NoError(t, err)
	assert.True(t, img.IsFlagged())
}
