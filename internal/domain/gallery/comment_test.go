package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewComment(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	userID := identity.NewUserID()

	t.Run("valid comment", func(t *testing.T) {
		t.Parallel()

		comment, err := gallery.NewComment(imageID, userID, "Great photo!")

		require.NoError(t, err)
		assert.Equal(t, "Great photo!", comment.Content())
		assert.Equal(t, imageID, comment.ImageID())
		assert.Equal(t, userID, comment.UserID())
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		_, err := gallery.NewComment(imageID, userID, "")

		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrCommentRequired)
	})
}

func TestComment_IsAuthoredBy(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	comment, _ := gallery.NewComment(imageID, userID, "Comment")

	assert.True(t, comment.IsAuthoredBy(userID))
	assert.False(t, comment.IsAuthoredBy(identity.NewUserID()))
}

func TestComment_Getters(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	comment, _ := gallery.NewComment(imageID, userID, "Content")

	assert.False(t, comment.ID().IsZero())
	assert.Equal(t, imageID, comment.ImageID())
	assert.Equal(t, userID, comment.UserID())
	assert.Equal(t, "Content", comment.Content())
	assert.False(t, comment.CreatedAt().IsZero())
	assert.Len(t, comment.Events(), 1)
}

func TestComment_ClearEvents(t *testing.T) {
	t.Parallel()

	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	comment, _ := gallery.NewComment(imageID, userID, "Content")

	comment.ClearEvents()
	assert.Empty(t, comment.Events())
}
