package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestFeaturedPickID(t *testing.T) {
	t.Parallel()

	id := gallery.NewFeaturedPickID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())

	parsed, err := gallery.ParseFeaturedPickID(id.String())
	assert.NoError(t, err)
	assert.True(t, id.Equals(parsed))

	id2 := gallery.NewFeaturedPickID()
	assert.False(t, id.Equals(id2))

	_, err = gallery.ParseFeaturedPickID("invalid")
	assert.Error(t, err)
}
