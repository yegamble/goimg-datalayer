package testhelpers

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	ValidUserID            = "550e8400-e29b-41d4-a716-446655440000"
	ValidImageID           = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	ValidFilename          = "test-image.jpg"
	ValidTitle             = "Test Image"
	ValidDescription       = "This is a test image"
	ValidMimeType          = "image/jpeg"
	ValidWidth             = 1920
	ValidHeight            = 1080
	ValidFileSize    int64 = 512000
)

type TestSuite struct {
	ImageRepo       *MockImageRepository
	Storage         *MockStorage
	JobEnqueuer     *MockJobEnqueuer
	EventPublisher  *MockEventPublisher
	IPFSService     *MockIPFSService
	StorageProvider *MockStorageProvider
	Logger          zerolog.Logger
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	return &TestSuite{
		ImageRepo:       new(MockImageRepository),
		Storage:         new(MockStorage),
		JobEnqueuer:     new(MockJobEnqueuer),
		EventPublisher:  new(MockEventPublisher),
		IPFSService:     new(MockIPFSService),
		StorageProvider: new(MockStorageProvider),
		Logger:          zerolog.Nop(),
	}
}

func (s *TestSuite) AssertExpectations(t *testing.T) {
	t.Helper()

	s.ImageRepo.AssertExpectations(t)
	s.Storage.AssertExpectations(t)
	s.JobEnqueuer.AssertExpectations(t)
	s.EventPublisher.AssertExpectations(t)
	s.IPFSService.AssertExpectations(t)
	s.StorageProvider.AssertExpectations(t)
}

func ValidUserIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidUserID)
	return userID
}

func ValidImageIDParsed() gallery.ImageID {
	imageID, _ := gallery.ParseImageID(ValidImageID)
	return imageID
}

func ValidImage(t *testing.T) *gallery.Image {
	t.Helper()

	ownerID := ValidUserIDParsed()
	metadata := ValidImageMetadata(t)

	image, err := gallery.NewImage(ownerID, metadata)
	require.NoError(t, err)

	require.NoError(t, image.MarkAsActive())

	require.NoError(t, image.UpdateVisibility(gallery.VisibilityPublic))

	return image
}

func ValidImageMetadata(t *testing.T) gallery.ImageMetadata {
	t.Helper()

	metadata, err := gallery.NewImageMetadata(
		ValidTitle,
		ValidDescription,
		ValidFilename,
		ValidMimeType,
		ValidWidth,
		ValidHeight,
		ValidFileSize,
		"images/test/original",
		"local",
	)
	require.NoError(t, err)

	return metadata
}

func ValidFileReader() io.Reader {
	return bytes.NewReader(make([]byte, ValidFileSize))
}

func ValidTag(t *testing.T, name string) gallery.Tag {
	t.Helper()

	tag, err := gallery.NewTag(name)
	require.NoError(t, err)

	return tag
}

const ValidIPFSCID = "QmTzQ1JRkWErjk39mryYw2WVPhE8u1S6aLNpT3EEDwzJ1X"

func ValidImageWithIPFS(t *testing.T) *gallery.Image {
	t.Helper()

	image := ValidImage(t)

	pinnedAt := ValidTimestamp()
	ipfsMeta, err := gallery.NewIPFSMetadata(
		ValidIPFSCID,
		true,
		&pinnedAt,
	)
	require.NoError(t, err)
	require.NoError(t, image.SetIPFSMetadata(ipfsMeta))

	return image
}

func ValidAlbum(t *testing.T) *gallery.Album {
	t.Helper()

	ownerID := ValidUserIDParsed()
	album, err := gallery.NewAlbum(ownerID, "Test Album")
	require.NoError(t, err)

	require.NoError(t, album.UpdateDescription("Test album description"))
	require.NoError(t, album.UpdateVisibility(gallery.VisibilityPublic))

	return album
}

const ValidAlbumID = "8c9e6679-7425-40de-944b-e07fc1f90ae8"

func ValidAlbumIDParsed() gallery.AlbumID {
	albumID, _ := gallery.ParseAlbumID(ValidAlbumID)
	return albumID
}

func ValidComment(t *testing.T) *gallery.Comment {
	t.Helper()

	userID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	comment, err := gallery.NewComment(imageID, userID, "This is a test comment")
	require.NoError(t, err)

	return comment
}

const ValidCommentID = "9c9e6679-7425-40de-944b-e07fc1f90ae9"

func ValidCommentIDParsed() gallery.CommentID {
	commentID, _ := gallery.ParseCommentID(ValidCommentID)
	return commentID
}

func ValidUser(t *testing.T) *identity.User {
	t.Helper()

	userID := ValidUserIDParsed()
	email, err := identity.NewEmail("test@example.com")
	require.NoError(t, err)

	username, err := identity.NewUsername("testuser")
	require.NoError(t, err)

	passwordHash, err := identity.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMye7WdZGIsgbRJHaC0G/YLnQ5zt1g/K7i2")
	require.NoError(t, err)

	user := identity.ReconstructUser(
		userID,
		email,
		username,
		passwordHash,
		identity.RoleUser,
		identity.StatusActive,
		"",
		"",
		0,
		ValidTimestamp(),
		ValidTimestamp(),
		identity.UserTypeRegistered,
		nil,
		nil,
		false,
		nil,
	)

	return user
}

func ValidModeratorUser(t *testing.T) *identity.User {
	t.Helper()

	userID := identity.NewUserID()
	email, err := identity.NewEmail("moderator@example.com")
	require.NoError(t, err)

	username, err := identity.NewUsername("moduser123")
	require.NoError(t, err)

	passwordHash, err := identity.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMye7WdZGIsgbRJHaC0G/YLnQ5zt1g/K7i2")
	require.NoError(t, err)

	user := identity.ReconstructUser(
		userID,
		email,
		username,
		passwordHash,
		identity.RoleModerator,
		identity.StatusActive,
		"",
		"",
		0,
		ValidTimestamp(),
		ValidTimestamp(),
		identity.UserTypeRegistered,
		nil,
		nil,
		false,
		nil,
	)

	return user
}

func ValidTimestamp() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}
