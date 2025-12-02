package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestImageStatus_Functions(t *testing.T) {
	t.Parallel()

	assert.Len(t, gallery.AllImageStatuses(), 4)

	status := gallery.StatusActive
	assert.True(t, status.IsValid())
	assert.True(t, status.IsViewable())
	assert.False(t, status.IsDeleted())
	assert.False(t, status.IsFlagged())
	assert.Equal(t, "active", status.String())

	deleted := gallery.StatusDeleted
	assert.True(t, deleted.IsDeleted())
	assert.False(t, deleted.IsViewable())

	flagged := gallery.StatusFlagged
	assert.True(t, flagged.IsFlagged())
}

func TestParseImageStatus(t *testing.T) {
	t.Parallel()

	status, err := gallery.ParseImageStatus("active")
	assert.NoError(t, err)
	assert.Equal(t, gallery.StatusActive, status)

	_, err = gallery.ParseImageStatus("invalid")
	assert.Error(t, err)
}

func TestVariantType_Functions(t *testing.T) {
	t.Parallel()

	assert.Len(t, gallery.AllVariantTypes(), 5)

	vt := gallery.VariantThumbnail
	assert.True(t, vt.IsValid())
	assert.Equal(t, 150, vt.MaxWidth())
	assert.Equal(t, "thumbnail", vt.String())

	assert.Equal(t, 320, gallery.VariantSmall.MaxWidth())
	assert.Equal(t, 800, gallery.VariantMedium.MaxWidth())
	assert.Equal(t, 1600, gallery.VariantLarge.MaxWidth())
	assert.Equal(t, 0, gallery.VariantOriginal.MaxWidth())
}

func TestParseVariantType(t *testing.T) {
	t.Parallel()

	vt, err := gallery.ParseVariantType("thumbnail")
	assert.NoError(t, err)
	assert.Equal(t, gallery.VariantThumbnail, vt)

	_, err = gallery.ParseVariantType("invalid")
	assert.Error(t, err)
}
