package gallery_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestReconstructAlbum(t *testing.T) {
	t.Parallel()

	albumID := gallery.NewAlbumID()
	ownerID := identity.NewUserID()
	imageID := gallery.NewImageID()
	now := time.Now()

	album := gallery.ReconstructAlbum(
		albumID,
		ownerID,
		"Title",
		"Description",
		gallery.VisibilityPublic,
		&imageID,
		5,
		now,
		now,
	)

	assert.NotNil(t, album)
	assert.Equal(t, albumID, album.ID())
	assert.Equal(t, ownerID, album.OwnerID())
	assert.Equal(t, "Title", album.Title())
	assert.Equal(t, "Description", album.Description())
	assert.Equal(t, 5, album.ImageCount())
}

func TestReconstructComment(t *testing.T) {
	t.Parallel()

	commentID := gallery.NewCommentID()
	imageID := gallery.NewImageID()
	userID := identity.NewUserID()
	now := time.Now()

	comment := gallery.ReconstructComment(
		commentID,
		imageID,
		userID,
		"Content",
		now,
	)

	assert.NotNil(t, comment)
	assert.Equal(t, commentID, comment.ID())
	assert.Equal(t, imageID, comment.ImageID())
	assert.Equal(t, userID, comment.UserID())
	assert.Equal(t, "Content", comment.Content())
}
