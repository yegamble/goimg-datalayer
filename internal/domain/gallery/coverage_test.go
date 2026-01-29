package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestImageMetadata(t *testing.T) {
	t.Parallel()

	t.Run("AspectRatio", func(t *testing.T) {
		meta, _ := gallery.NewImageMetadata(
			"Title", "Desc", "file.jpg", "image/jpeg", 1920, 1080, 1024, "key", "local",
		)
		assert.InDelta(t, 1.777, meta.AspectRatio(), 0.001)

		metaZeroHeight, _ := gallery.NewImageMetadata(
			"Title", "Desc", "file.jpg", "image/jpeg", 1920, 0, 1024, "key", "local",
		)
		assert.NotNil(t, metaZeroHeight)
	})

	t.Run("Validation Edge Cases", func(t *testing.T) {
		_, err := gallery.NewImageMetadata(
			"Title", "Desc", "file.jpg", "image/jpeg", 0, 1080, 1024, "key", "local",
		)
		assert.Error(t, err)
	})
}

func TestImageStatus(t *testing.T) {
	t.Parallel()

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, gallery.StatusActive.IsValid())
		assert.False(t, gallery.ImageStatus("invalid").IsValid())
	})

	t.Run("Checks", func(t *testing.T) {
		assert.True(t, gallery.StatusActive.IsViewable())
		assert.False(t, gallery.StatusDeleted.IsViewable())

		assert.True(t, gallery.StatusDeleted.IsDeleted())
		assert.False(t, gallery.StatusActive.IsDeleted())

		assert.True(t, gallery.StatusFlagged.IsFlagged())
		assert.False(t, gallery.StatusActive.IsFlagged())
	})

	t.Run("Parse", func(t *testing.T) {
		status, err := gallery.ParseImageStatus("active")
		require.NoError(t, err)
		assert.Equal(t, gallery.StatusActive, status)

		_, err = gallery.ParseImageStatus("invalid")
		assert.Error(t, err)
	})

	t.Run("All", func(t *testing.T) {
		assert.Len(t, gallery.AllImageStatuses(), 4)
	})
}

func TestVariantConfig(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	name := "Thumbnail"
	format := gallery.FormatJPEG
	cropMode := gallery.CropModeFill

	t.Run("NewVariantConfig", func(t *testing.T) {
		config, err := gallery.NewVariantConfig(userID, name, 200, 200, format, 80, cropMode)
		require.NoError(t, err)

		assert.False(t, config.ID().IsZero())
		assert.Equal(t, userID, config.UserID())
		assert.Equal(t, "thumbnail", config.Name())
		assert.Equal(t, 200, config.MaxWidth())
		assert.Equal(t, 200, config.MaxHeight())
		assert.Equal(t, format, config.Format())
		assert.Equal(t, 80, config.Quality())
		assert.Equal(t, cropMode, config.CropMode())
		assert.False(t, config.IsPreset())
		assert.False(t, config.CreatedAt().IsZero())
		assert.False(t, config.UpdatedAt().IsZero())

		events := config.Events()
		require.Len(t, events, 1)
		assert.IsType(t, &gallery.VariantConfigCreated{}, events[0])
		assert.Equal(t, "gallery.variant_config.created", events[0].EventType())
	})

	t.Run("Validation Errors", func(t *testing.T) {
		_, err := gallery.NewVariantConfig(identity.UserID{}, name, 200, 200, format, 80, cropMode)
		assert.Error(t, err)

		_, err = gallery.NewVariantConfig(userID, "", 200, 200, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, "a", 200, 200, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, "too-long-name-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 200, 200, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, "invalid name!", 200, 200, format, 80, cropMode)
		assert.Error(t, err)

		_, err = gallery.NewVariantConfig(userID, name, 0, 200, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, name, 9000, 200, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, name, 200, 0, format, 80, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, name, 200, 9000, format, 80, cropMode)
		assert.Error(t, err)

		_, err = gallery.NewVariantConfig(userID, name, 200, 200, format, 0, cropMode)
		assert.Error(t, err)
		_, err = gallery.NewVariantConfig(userID, name, 200, 200, format, 101, cropMode)
		assert.Error(t, err)

		_, err = gallery.NewVariantConfig(userID, name, 200, 200, gallery.OutputFormat("invalid"), 80, cropMode)
		assert.Error(t, err)

		_, err = gallery.NewVariantConfig(userID, name, 200, 200, format, 80, gallery.CropMode("invalid"))
		assert.Error(t, err)
	})

	t.Run("Update", func(t *testing.T) {
		config, _ := gallery.NewVariantConfig(userID, name, 200, 200, format, 80, cropMode)
		config.ClearEvents()

		newFormat := gallery.FormatPNG
		newCrop := gallery.CropModeFit

		err := config.Update(400, 400, newFormat, 90, newCrop)
		require.NoError(t, err)

		assert.Equal(t, 400, config.MaxWidth())
		assert.Equal(t, 400, config.MaxHeight())
		assert.Equal(t, newFormat, config.Format())
		assert.Equal(t, 90, config.Quality())
		assert.Equal(t, newCrop, config.CropMode())

		events := config.Events()
		require.Len(t, events, 1)
		assert.IsType(t, &gallery.VariantConfigUpdated{}, events[0])
		assert.Equal(t, "gallery.variant_config.updated", events[0].EventType())
	})

	t.Run("Update Validation", func(t *testing.T) {
		config, _ := gallery.NewVariantConfig(userID, name, 200, 200, format, 80, cropMode)

		err := config.Update(0, 400, format, 90, cropMode)
		assert.Error(t, err)
		err = config.Update(9000, 400, format, 90, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 0, format, 90, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 9000, format, 90, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 400, format, 0, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 400, format, 101, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 400, gallery.OutputFormat("invalid"), 90, cropMode)
		assert.Error(t, err)
		err = config.Update(400, 400, format, 90, gallery.CropMode("invalid"))
		assert.Error(t, err)
	})

	t.Run("ReconstructVariantConfig", func(t *testing.T) {
		id := gallery.NewVariantConfigID()
		now := time.Now()
		config := gallery.ReconstructVariantConfig(
			id, userID, name, 300, 300, format, 85, cropMode, true, now, now,
		)

		assert.Equal(t, id, config.ID())
		assert.Equal(t, userID, config.UserID())
		assert.Equal(t, name, config.Name())
		assert.Equal(t, 300, config.MaxWidth())
		assert.Equal(t, 85, config.Quality())
		assert.True(t, config.IsPreset())
		assert.Equal(t, now, config.CreatedAt())
		assert.Empty(t, config.Events())
	})

	t.Run("IsOwnedBy", func(t *testing.T) {
		config, _ := gallery.NewVariantConfig(userID, name, 200, 200, format, 80, cropMode)
		assert.True(t, config.IsOwnedBy(userID))
		assert.False(t, config.IsOwnedBy(identity.NewUserID()))
	})

	t.Run("Event Types", func(t *testing.T) {
		created := &gallery.VariantConfigCreated{}
		assert.Equal(t, "gallery.variant_config.created", created.EventType())

		updated := &gallery.VariantConfigUpdated{}
		assert.Equal(t, "gallery.variant_config.updated", updated.EventType())

		deleted := &gallery.VariantConfigDeleted{}
		assert.Equal(t, "gallery.variant_config.deleted", deleted.EventType())
	})
}

func TestVariantConfigID(t *testing.T) {
	t.Parallel()

	t.Run("NewVariantConfigID", func(t *testing.T) {
		id := gallery.NewVariantConfigID()
		assert.False(t, id.IsZero())
		assert.NotNil(t, id.UUID())
	})

	t.Run("ParseVariantConfigID", func(t *testing.T) {
		original := gallery.NewVariantConfigID()
		parsed, err := gallery.ParseVariantConfigID(original.String())
		require.NoError(t, err)
		assert.Equal(t, original, parsed)

		_, err = gallery.ParseVariantConfigID("invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("Equals", func(t *testing.T) {
		id1 := gallery.NewVariantConfigID()
		id2 := gallery.NewVariantConfigID()
		assert.True(t, id1.Equals(id1))
		assert.False(t, id1.Equals(id2))
	})
}

func TestScanStatus(t *testing.T) {
	t.Parallel()

	t.Run("AllScanStatuses", func(t *testing.T) {
		statuses := gallery.AllScanStatuses()
		assert.Contains(t, statuses, gallery.ScanStatusPending)
		assert.Contains(t, statuses, gallery.ScanStatusClean)
		assert.Contains(t, statuses, gallery.ScanStatusInfected)
		assert.Contains(t, statuses, gallery.ScanStatusError)
	})

	t.Run("ParseScanStatus", func(t *testing.T) {
		status, err := gallery.ParseScanStatus("clean")
		require.NoError(t, err)
		assert.Equal(t, gallery.ScanStatusClean, status)

		_, err = gallery.ParseScanStatus("invalid")
		assert.Error(t, err)
	})

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, gallery.ScanStatusClean.IsValid())
		assert.False(t, gallery.ScanStatus("invalid").IsValid())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "pending", gallery.ScanStatusPending.String())
	})
}

func TestVariantType(t *testing.T) {
	t.Parallel()

	t.Run("MaxWidth", func(t *testing.T) {
		assert.Equal(t, 150, gallery.VariantThumbnail.MaxWidth())
		assert.Equal(t, 320, gallery.VariantSmall.MaxWidth())
		assert.Equal(t, 800, gallery.VariantMedium.MaxWidth())
		assert.Equal(t, 1600, gallery.VariantLarge.MaxWidth())
		assert.Equal(t, 0, gallery.VariantOriginal.MaxWidth())
		assert.Equal(t, 0, gallery.VariantType("invalid").MaxWidth())
	})

	t.Run("All", func(t *testing.T) {
		assert.NotEmpty(t, gallery.AllVariantTypes())
	})

	t.Run("Parse", func(t *testing.T) {
		vt, err := gallery.ParseVariantType("thumbnail")
		require.NoError(t, err)
		assert.Equal(t, gallery.VariantThumbnail, vt)

		_, err = gallery.ParseVariantType("invalid")
		assert.Error(t, err)
	})

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, gallery.VariantThumbnail.IsValid())
		assert.False(t, gallery.VariantType("invalid").IsValid())
	})
}

func TestTag(t *testing.T) {
	t.Parallel()

	t.Run("NewTag", func(t *testing.T) {
		tag, err := gallery.NewTag("Nature")
		require.NoError(t, err)
		// Tags are normalized to lowercase
		assert.Equal(t, "nature", tag.Name())
		assert.Equal(t, "nature", tag.Slug())

		_, err = gallery.NewTag("")
		assert.Error(t, err)

		_, err = gallery.NewTag("a") // Too short?
	})

	t.Run("MustNewTag", func(t *testing.T) {
		require.NotPanics(t, func() {
			gallery.MustNewTag("Nature")
		})
		require.Panics(t, func() {
			gallery.MustNewTag("")
		})
	})

	t.Run("Equals", func(t *testing.T) {
		t1, _ := gallery.NewTag("Nature")
		t2, _ := gallery.NewTag("Nature")
		t3, _ := gallery.NewTag("Urban")

		assert.True(t, t1.Equals(t2))
		assert.False(t, t1.Equals(t3))
	})

	t.Run("String", func(t *testing.T) {
		tag, _ := gallery.NewTag("Nature")
		assert.Equal(t, "nature", tag.String())
	})
}

func TestComment(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	content := "This is a comment"

	t.Run("NewComment", func(t *testing.T) {
		comment, err := gallery.NewComment(imageID, userID, content)
		require.NoError(t, err)

		assert.False(t, comment.ID().IsZero())
		assert.Equal(t, imageID, comment.ImageID())
		assert.Equal(t, userID, comment.UserID())
		assert.Equal(t, content, comment.Content())
		assert.False(t, comment.CreatedAt().IsZero())

		events := comment.Events()
		require.Len(t, events, 1)
	})

	t.Run("Validation", func(t *testing.T) {
		_, err := gallery.NewComment(gallery.ImageID{}, userID, content)
		assert.Error(t, err)

		_, err = gallery.NewComment(imageID, identity.UserID{}, content)
		assert.Error(t, err)

		_, err = gallery.NewComment(imageID, userID, "")
		assert.Error(t, err)
	})

	t.Run("Reconstruct", func(t *testing.T) {
		id := gallery.NewCommentID()
		now := time.Now()

		comment := gallery.ReconstructComment(id, imageID, userID, content, now)
		assert.Equal(t, id, comment.ID())
		assert.Equal(t, now, comment.CreatedAt())
		assert.Empty(t, comment.Events())
	})

	t.Run("AuthoredBy", func(t *testing.T) {
		comment, _ := gallery.NewComment(imageID, userID, content)
		assert.True(t, comment.IsAuthoredBy(userID))
		assert.False(t, comment.IsAuthoredBy(identity.NewUserID()))
	})

	t.Run("ClearEvents", func(t *testing.T) {
		comment, _ := gallery.NewComment(imageID, userID, content)
		assert.NotEmpty(t, comment.Events())
		comment.ClearEvents()
		assert.Empty(t, comment.Events())
	})
}

func TestAlbum_Hierarchy(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	album, _ := gallery.NewAlbum(ownerID, "Album")

	t.Run("Root", func(t *testing.T) {
		assert.True(t, album.IsRoot())
		assert.False(t, album.HasParent())
		assert.Nil(t, album.ParentID())
	})

	t.Run("SetParent", func(t *testing.T) {
		parentID := gallery.NewAlbumID()

		album.SetParent(&parentID)
		assert.False(t, album.IsRoot())
		assert.True(t, album.HasParent())
		assert.Equal(t, parentID, *album.ParentID())

		// Event
		events := album.Events()
		assert.NotEmpty(t, events)

		// Same parent (no-op)
		album.ClearEvents()
		album.SetParent(&parentID)
		assert.Empty(t, album.Events())

		// Remove parent
		album.SetParent(nil)
		assert.True(t, album.IsRoot())
	})
}
