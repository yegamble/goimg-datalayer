package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestImageBehavior(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	meta, _ := gallery.NewImageMetadata("Title", "Desc", "file.jpg", "image/jpeg", 800, 600, 1024, "key", "local")
	image, err := gallery.NewImage(ownerID, meta)
	require.NoError(t, err)

	t.Run("NewImage", func(t *testing.T) {
		assert.False(t, image.ID().IsZero())
		assert.Equal(t, ownerID, image.OwnerID())
		assert.Equal(t, gallery.StatusProcessing, image.Status())
		assert.Equal(t, gallery.ScanStatusPending, image.ScanStatus())
		assert.Equal(t, gallery.VisibilityPrivate, image.Visibility())
		assert.Empty(t, image.Variants())
		assert.Empty(t, image.Tags())
		assert.Nil(t, image.IPFSMetadata())
		assert.False(t, image.HasIPFS())

		events := image.Events()
		require.NotEmpty(t, events)
		assert.IsType(t, &gallery.ImageUploaded{}, events[0])
	})

	t.Run("MarkAsActive", func(t *testing.T) {
		image.ClearEvents()
		err := image.MarkAsActive()
		require.NoError(t, err)
		assert.Equal(t, gallery.StatusActive, image.Status())

		// Event emitted
		events := image.Events()
		require.NotEmpty(t, events)
		assert.IsType(t, &gallery.ImageProcessingCompleted{}, events[0])

		// Idempotent
		image.ClearEvents()
		err = image.MarkAsActive()
		require.NoError(t, err)
		assert.Empty(t, image.Events())
	})

	t.Run("UpdateVisibility", func(t *testing.T) {
		image.ClearEvents()

		// Can update active image
		err := image.UpdateVisibility(gallery.VisibilityPublic)
		require.NoError(t, err)
		assert.Equal(t, gallery.VisibilityPublic, image.Visibility())

		events := image.Events()
		require.NotEmpty(t, events)
		assert.IsType(t, &gallery.ImageVisibilityChanged{}, events[0])

		// Cannot update processing image (re-create for test)
		procImg, _ := gallery.NewImage(ownerID, meta)
		err = procImg.UpdateVisibility(gallery.VisibilityPublic)
		assert.ErrorIs(t, err, gallery.ErrImageProcessing)
	})

	t.Run("Scan Status", func(t *testing.T) {
		image.ClearEvents()

		err := image.MarkAsClean()
		require.NoError(t, err)
		assert.Equal(t, gallery.ScanStatusClean, image.ScanStatus())

		err = image.MarkAsInfected()
		require.NoError(t, err)
		assert.Equal(t, gallery.ScanStatusInfected, image.ScanStatus())

		// Cannot activate infected image (need to reset status first, but MarkAsActive checks current status)
		// Let's create a new one to test infection block
		infImg, _ := gallery.NewImage(ownerID, meta)
		_ = infImg.MarkAsInfected()
		err = infImg.MarkAsActive()
		assert.ErrorIs(t, err, gallery.ErrMalwareDetected)

		// SetScanStatus direct
		err = image.SetScanStatus(gallery.ScanStatusClean)
		require.NoError(t, err)
		assert.Equal(t, gallery.ScanStatusClean, image.ScanStatus())

		err = image.SetScanStatus(gallery.ScanStatus("invalid"))
		assert.Error(t, err)
	})

	t.Run("Flag and Delete", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		_ = img.MarkAsActive()
		img.ClearEvents()

		// Flag
		err := img.Flag()
		require.NoError(t, err)
		assert.Equal(t, gallery.StatusFlagged, img.Status())
		assert.IsType(t, &gallery.ImageFlagged{}, img.Events()[0])

		// Cannot delete flagged
		err = img.MarkAsDeleted()
		assert.ErrorIs(t, err, gallery.ErrCannotDeleteFlagged)

		// Unflag (by setting active? No direct method, assume moderation does something else.
		// Actually MarkAsActive sets status to active if not deleted.
		_ = img.MarkAsActive() // Recover

		// Delete
		img.ClearEvents()
		err = img.MarkAsDeleted()
		require.NoError(t, err)
		assert.Equal(t, gallery.StatusDeleted, img.Status())
		assert.IsType(t, &gallery.ImageDeleted{}, img.Events()[0])

		// Cannot modify deleted
		err = img.UpdateVisibility(gallery.VisibilityPublic)
		assert.ErrorIs(t, err, gallery.ErrCannotModifyDeleted)

		err = img.Flag()
		assert.ErrorIs(t, err, gallery.ErrCannotModifyDeleted)
	})

	t.Run("ChangeOwner", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		newOwner := identity.NewUserID()

		err := img.ChangeOwner(newOwner)
		require.NoError(t, err)
		assert.Equal(t, newOwner, img.OwnerID())

		events := img.Events()
		// First is Uploaded, Second is OwnershipChanged
		assert.IsType(t, &gallery.ImageOwnershipChanged{}, events[1])

		// Cannot transfer flagged
		img.Flag()
		err = img.ChangeOwner(ownerID)
		assert.Error(t, err)
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		img.ClearEvents()

		err := img.UpdateMetadata("New Title", "New Desc")
		require.NoError(t, err)
		assert.Equal(t, "New Title", img.Metadata().Title())

		// No change
		img.ClearEvents()
		err = img.UpdateMetadata("New Title", "New Desc")
		require.NoError(t, err)
		assert.Empty(t, img.Events())
	})

	t.Run("Variants", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		variant, _ := gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, 100, 1024, "jpeg")

		err := img.AddVariant(variant)
		require.NoError(t, err)
		assert.Len(t, img.Variants(), 1)
		assert.True(t, img.HasVariant(gallery.VariantThumbnail))

		v, err := img.GetVariant(gallery.VariantThumbnail)
		require.NoError(t, err)
		assert.Equal(t, variant, v)

		_, err = img.GetVariant(gallery.VariantLarge)
		assert.ErrorIs(t, err, gallery.ErrVariantNotFound)

		// Duplicate
		err = img.AddVariant(variant)
		assert.ErrorIs(t, err, gallery.ErrVariantExists)
	})

	t.Run("Tags", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		tag, _ := gallery.NewTag("Nature")

		err := img.AddTag(tag)
		require.NoError(t, err)
		assert.True(t, img.HasTag(tag))

		// Duplicate
		err = img.AddTag(tag)
		assert.ErrorIs(t, err, gallery.ErrTagAlreadyExists)

		// Remove
		err = img.RemoveTag(tag)
		require.NoError(t, err)
		assert.False(t, img.HasTag(tag))

		// Remove non-existent (idempotent)
		err = img.RemoveTag(tag)
		require.NoError(t, err)
	})

	t.Run("IPFS", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		validCID := "Qm12345678901234567890123456789012345678901234"
		ipfsMeta, _ := gallery.NewIPFSMetadata(validCID, false, nil)

		err := img.SetIPFSMetadata(ipfsMeta)
		require.NoError(t, err)
		assert.True(t, img.HasIPFS())
		assert.Equal(t, ipfsMeta, *img.IPFSMetadata())

		// Set same (idempotent/update)
		err = img.SetIPFSMetadata(ipfsMeta)
		require.NoError(t, err)

		// Clear
		err = img.ClearIPFSMetadata()
		require.NoError(t, err)
		assert.False(t, img.HasIPFS())
	})

	t.Run("Metrics", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)

		img.IncrementViews()
		assert.Equal(t, int64(1), img.ViewCount())

		img.SetLikeCount(10)
		assert.Equal(t, int64(10), img.LikeCount())

		img.SetCommentCount(5)
		assert.Equal(t, int64(5), img.CommentCount())
	})

	t.Run("Reconstruct", func(t *testing.T) {
		id := gallery.NewImageID()
		now := time.Now()
		img := gallery.ReconstructImage(
			id, ownerID, meta, gallery.VisibilityPublic, gallery.StatusActive, gallery.ScanStatusClean,
			nil, nil, nil, 10, 5, 2, now, now,
		)

		assert.Equal(t, id, img.ID())
		assert.Equal(t, gallery.StatusActive, img.Status())
		assert.Equal(t, int64(10), img.ViewCount())
		assert.Empty(t, img.Events())
	})

	t.Run("Getters", func(t *testing.T) {
		img, _ := gallery.NewImage(ownerID, meta)
		// Ensure Getters don't panic and return correct initial state
		assert.False(t, img.IsDeleted())
		assert.False(t, img.IsFlagged())
		assert.False(t, img.IsViewable()) // Processing
	})
}

func TestIPFSMetadata_Validation(t *testing.T) {
	t.Parallel()

	// Valid CIDv0
	validCIDv0 := "Qm12345678901234567890123456789012345678901234"

	t.Run("Valid", func(t *testing.T) {
		now := time.Now()
		m, err := gallery.NewIPFSMetadata(validCIDv0, true, &now)
		require.NoError(t, err)
		assert.Equal(t, validCIDv0, m.CID())
		assert.True(t, m.Pinned())
		assert.NotNil(t, m.PinnedAt())
		assert.Equal(t, "ipfs://"+validCIDv0, m.URI())
		assert.Equal(t, "https://ipfs.io/ipfs/"+validCIDv0, m.GatewayURL("https://ipfs.io"))
	})

	t.Run("Invalid CID", func(t *testing.T) {
		// Empty
		_, err := gallery.NewIPFSMetadata("", false, nil)
		assert.Error(t, err)

		// Short CIDv0
		_, err = gallery.NewIPFSMetadata("QmShort", false, nil)
		assert.Error(t, err)

		// Invalid Prefix
		_, err = gallery.NewIPFSMetadata("InvalidPrefix123456", false, nil)
		assert.Error(t, err)
	})

	t.Run("Methods", func(t *testing.T) {
		m, _ := gallery.NewIPFSMetadata(validCIDv0, false, nil)
		assert.False(t, m.IsZero())

		m2 := m.WithPinned(true)
		assert.True(t, m2.Pinned())
		assert.NotNil(t, m2.PinnedAt())

		assert.True(t, m.Equals(m2)) // Equality checks CID only
	})
}

func TestImageVariant_Validation(t *testing.T) {
	t.Parallel()

	t.Run("NewImageVariant", func(t *testing.T) {
		// Valid
		v, err := gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, 100, 1024, "jpeg")
		require.NoError(t, err)
		assert.Equal(t, gallery.VariantThumbnail, v.VariantType())
		assert.Equal(t, "jpeg", v.Format())

		// Invalid Type
		_, err = gallery.NewImageVariant(gallery.VariantType("invalid"), "key", 100, 100, 1024, "jpeg")
		assert.Error(t, err)

		// Invalid Key
		_, err = gallery.NewImageVariant(gallery.VariantThumbnail, "", 100, 100, 1024, "jpeg")
		assert.Error(t, err)

		// Invalid Dimensions
		_, err = gallery.NewImageVariant(gallery.VariantThumbnail, "key", 0, 100, 1024, "jpeg")
		assert.Error(t, err)
		_, err = gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, -1, 1024, "jpeg")
		assert.Error(t, err)

		// Invalid Size
		_, err = gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, 100, 0, "jpeg")
		assert.Error(t, err)

		// Invalid Format (empty)
		_, err = gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, 100, 1024, "")
		assert.Error(t, err)
	})

	t.Run("AspectRatio", func(t *testing.T) {
		v, _ := gallery.NewImageVariant(gallery.VariantThumbnail, "key", 100, 50, 1024, "jpeg")
		assert.Equal(t, 2.0, v.AspectRatio())
	})
}
