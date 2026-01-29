package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewVariantConfig(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	name := "Thumbnail"
	width := 150
	height := 150
	quality := 80
	format := "jpeg"

	t.Run("valid config", func(t *testing.T) {
		cfg, err := gallery.NewVariantConfig(userID, name, width, height, quality, format, false)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.False(t, cfg.ID().IsZero())
		assert.Equal(t, userID, cfg.UserID())
		assert.Equal(t, name, cfg.Name())
		assert.Equal(t, width, cfg.Width())
		assert.False(t, cfg.IsSystemPreset())
	})

	t.Run("invalid dimensions", func(t *testing.T) {
		_, err := gallery.NewVariantConfig(userID, name, 0, 0, quality, format, false)
		require.Error(t, err)
	})

	t.Run("invalid quality", func(t *testing.T) {
		_, err := gallery.NewVariantConfig(userID, name, width, height, 101, format, false)
		require.Error(t, err)
	})
}

func TestReconstructVariantConfig(t *testing.T) {
	t.Parallel()

	id := gallery.NewVariantConfigID()
	userID := identity.NewUserID()
	now := time.Now()

	cfg := gallery.ReconstructVariantConfig(
		id, userID, "Test", 100, 100, 80, "png", true, true, now, now,
	)

	assert.Equal(t, id, cfg.ID())
	assert.Equal(t, "Test", cfg.Name())
	assert.True(t, cfg.MaintainAspectRatio())
}
