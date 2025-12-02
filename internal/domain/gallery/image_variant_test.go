package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestNewImageVariant(t *testing.T) {
	t.Parallel()

	t.Run("valid variant", func(t *testing.T) {
		t.Parallel()

		variant, err := gallery.NewImageVariant(
			gallery.VariantThumbnail,
			"thumb.jpg",
			150,
			150,
			1000,
			"jpeg",
		)

		require.NoError(t, err)
		assert.Equal(t, gallery.VariantThumbnail, variant.VariantType())
		assert.Equal(t, 150, variant.Width())
		assert.Equal(t, 150, variant.Height())
	})

	t.Run("invalid variant type", func(t *testing.T) {
		t.Parallel()

		_, err := gallery.NewImageVariant(
			"invalid",
			"file.jpg",
			100,
			100,
			1000,
			"jpeg",
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrInvalidVariantType)
	})

	t.Run("zero dimensions", func(t *testing.T) {
		t.Parallel()

		_, err := gallery.NewImageVariant(
			gallery.VariantThumbnail,
			"file.jpg",
			0,
			100,
			1000,
			"jpeg",
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrInvalidVariantData)
	})
}

func TestImageVariant_AspectRatio(t *testing.T) {
	t.Parallel()

	variant, _ := gallery.NewImageVariant(
		gallery.VariantThumbnail,
		"file.jpg",
		1920,
		1080,
		1000,
		"jpeg",
	)

	ratio := variant.AspectRatio()
	assert.InDelta(t, 1.777, ratio, 0.001)
}

func TestImageVariant_Getters(t *testing.T) {
	t.Parallel()

	variant, _ := gallery.NewImageVariant(
		gallery.VariantThumbnail,
		"thumb.jpg",
		150,
		100,
		5000,
		"jpeg",
	)

	assert.Equal(t, gallery.VariantThumbnail, variant.VariantType())
	assert.Equal(t, "thumb.jpg", variant.StorageKey())
	assert.Equal(t, 150, variant.Width())
	assert.Equal(t, 100, variant.Height())
	assert.Equal(t, int64(5000), variant.FileSize())
	assert.Equal(t, "jpeg", variant.Format())
}

func TestImageVariant_FormatNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputFormat  string
		expectFormat string
	}{
		{"jpg lowercase", "jpg", "jpeg"},
		{"JPG uppercase", "JPG", "jpeg"},
		{"JPEG uppercase", "JPEG", "jpeg"},
		{"png lowercase", "png", "png"},
		{"PNG uppercase", "PNG", "png"},
		{"gif lowercase", "gif", "gif"},
		{"GIF uppercase", "GIF", "gif"},
		{"webp lowercase", "webp", "webp"},
		{"WebP mixed", "WebP", "webp"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			variant, err := gallery.NewImageVariant(
				gallery.VariantThumbnail,
				"file.jpg",
				100,
				100,
				1000,
				tt.inputFormat,
			)

			require.NoError(t, err)
			assert.Equal(t, tt.expectFormat, variant.Format())
		})
	}
}
