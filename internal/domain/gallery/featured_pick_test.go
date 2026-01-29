package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewFeaturedPick(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	curatorID := identity.NewUserID()
	now := time.Now()
	until := now.Add(24 * time.Hour)

	t.Run("valid pick", func(t *testing.T) {
		pick, err := gallery.NewFeaturedPick(imageID, curatorID, 1, &now, &until)
		require.NoError(t, err)
		assert.NotNil(t, pick)
		assert.False(t, pick.ID().IsZero())
		assert.Equal(t, imageID, pick.ImageID())
		assert.Equal(t, curatorID, pick.CuratorID())
		assert.True(t, pick.IsActive())
	})

	t.Run("invalid image ID", func(t *testing.T) {
		_, err := gallery.NewFeaturedPick(gallery.ImageID{}, curatorID, 1, nil, nil)
		require.Error(t, err)
	})
}

func TestFeaturedPick_IsActive(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	curatorID := identity.NewUserID()
	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	// Active (started in past, ends in future)
	p1, _ := gallery.NewFeaturedPick(imageID, curatorID, 1, &past, &future)
	assert.True(t, p1.IsActive())

	// Inactive (starts in future)
	p2, _ := gallery.NewFeaturedPick(imageID, curatorID, 1, &future, nil)
	assert.False(t, p2.IsActive())

	// Inactive (ended in past)
	p3, _ := gallery.NewFeaturedPick(imageID, curatorID, 1, &past, &past)
	assert.False(t, p3.IsActive())
}
