package gallery_test

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)


func TestNewFeaturedPickWithSchedule(t *testing.T) {
	imageID := gallery.NewImageID()
	adminID := identity.NewUserID()
	now := time.Now().UTC()
	until := now.Add(24 * time.Hour)

	pick, err := gallery.NewFeaturedPickWithSchedule(imageID, adminID, "test reason", 1, now, &until)
	require.NoError(t, err)
	assert.NotNil(t, pick)
	assert.Equal(t, imageID, pick.ImageID())
	assert.Equal(t, adminID, pick.FeaturedBy())
	assert.Equal(t, "test reason", pick.Reason())
	assert.Equal(t, 1, pick.DisplayOrder())
	assert.Equal(t, now, pick.FeaturedFrom())
	assert.NotNil(t, pick.FeaturedUntil())
	assert.Equal(t, until, *pick.FeaturedUntil())

	// Test invalid parameters
	_, err = gallery.NewFeaturedPickWithSchedule(gallery.ImageID{}, adminID, "test", 1, now, nil)
	assert.ErrorIs(t, err, gallery.ErrInvalidMetadata)

	_, err = gallery.NewFeaturedPickWithSchedule(imageID, identity.UserID{}, "test", 1, now, nil)
	assert.ErrorIs(t, err, gallery.ErrInvalidMetadata)

	untilPast := now.Add(-24 * time.Hour)
	_, err = gallery.NewFeaturedPickWithSchedule(imageID, adminID, "test", 1, now, &untilPast)
	assert.ErrorIs(t, err, gallery.ErrInvalidMetadata)
}
