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

// Test constants.
const (
	ValidUserID            = "550e8400-e29b-41d4-a716-446655440000"
	ValidImageID           = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	ValidFilename          = "test-image.jpg"
	ValidTitle             = "Test Image"
	ValidDescription       = "This is a test image"
	ValidMimeType          = "image/jpeg"
	ValidWidth             = 1920
	ValidHeight            = 1080
	ValidFileSize    int64 = 512000 // 500KB
)

// TestSuite provides mock dependencies for testing.
type TestSuite struct {
	ImageRepo      *MockImageRepository
	Storage        *MockStorage
	JobEnqueuer    *MockJobEnqueuer
	EventPublisher *MockEventPublisher
	Logger         zerolog.Logger
}

// NewTestSuite creates a new test suite with mocked dependencies.
func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	return &TestSuite{
		ImageRepo:      new(MockImageRepository),
		Storage:        new(MockStorage),
		JobEnqueuer:    new(MockJobEnqueuer),
		EventPublisher: new(MockEventPublisher),
		Logger:         zerolog.Nop(), // No-op logger for tests
	}
}

// AssertExpectations verifies all mock expectations were met.
func (s *TestSuite) AssertExpectations(t *testing.T) {
	t.Helper()

	s.ImageRepo.AssertExpectations(t)
	s.Storage.AssertExpectations(t)
	s.JobEnqueuer.AssertExpectations(t)
	s.EventPublisher.AssertExpectations(t)
}

// ValidUserIDParsed returns a parsed UserID for testing.
func ValidUserIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidUserID)
	return userID
}

// ValidImageIDParsed returns a parsed ImageID for testing.
func ValidImageIDParsed() gallery.ImageID {
	imageID, _ := gallery.ParseImageID(ValidImageID)
	return imageID
}

// ValidImage creates a valid Image aggregate for testing.
func ValidImage(t *testing.T) *gallery.Image {
	t.Helper()

	ownerID := ValidUserIDParsed()
	metadata := ValidImageMetadata(t)

	image, err := gallery.NewImage(ownerID, metadata)
	require.NoError(t, err)

	// Mark as active so it's viewable
	require.NoError(t, image.MarkAsActive())

	// Set visibility to public for easier testing
	require.NoError(t, image.UpdateVisibility(gallery.VisibilityPublic))

	return image
}

// ValidImageMetadata creates valid ImageMetadata for testing.
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

// ValidFileReader returns a mock file reader for testing uploads.
func ValidFileReader() io.Reader {
	// Return a reader with some fake JPEG data
	return bytes.NewReader(make([]byte, ValidFileSize))
}

// ValidTag creates a valid Tag for testing.
func ValidTag(t *testing.T, name string) gallery.Tag {
	t.Helper()

	tag, err := gallery.NewTag(name)
	require.NoError(t, err)

	return tag
}

// ValidAlbum creates a valid Album aggregate for testing.
func ValidAlbum(t *testing.T) *gallery.Album {
	t.Helper()

	ownerID := ValidUserIDParsed()
	album, err := gallery.NewAlbum(ownerID, "Test Album")
	require.NoError(t, err)

	// Set description and visibility
	require.NoError(t, album.UpdateDescription("Test album description"))
	require.NoError(t, album.UpdateVisibility(gallery.VisibilityPublic))

	return album
}

// ValidAlbumID returns a valid album ID string for testing.
const ValidAlbumID = "8c9e6679-7425-40de-944b-e07fc1f90ae8"

// ValidAlbumIDParsed returns a parsed AlbumID for testing.
func ValidAlbumIDParsed() gallery.AlbumID {
	albumID, _ := gallery.ParseAlbumID(ValidAlbumID)
	return albumID
}

// ValidComment creates a valid Comment aggregate for testing.
func ValidComment(t *testing.T) *gallery.Comment {
	t.Helper()

	userID := ValidUserIDParsed()
	imageID := ValidImageIDParsed()
	comment, err := gallery.NewComment(imageID, userID, "This is a test comment")
	require.NoError(t, err)

	return comment
}

// ValidCommentID returns a valid comment ID string for testing.
const ValidCommentID = "9c9e6679-7425-40de-944b-e07fc1f90ae9"

// ValidCommentIDParsed returns a parsed CommentID for testing.
func ValidCommentIDParsed() gallery.CommentID {
	commentID, _ := gallery.ParseCommentID(ValidCommentID)
	return commentID
}

// ValidUser creates a valid User aggregate for testing.
func ValidUser(t *testing.T) *identity.User {
	t.Helper()

	userID := ValidUserIDParsed()
	email, err := identity.NewEmail("test@example.com")
	require.NoError(t, err)

	username, err := identity.NewUsername("testuser")
	require.NoError(t, err)

	passwordHash, err := identity.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMye7WdZGIsgbRJHaC0G/YLnQ5zt1g/K7i2") // "password"
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
		ValidTimestamp(),
		ValidTimestamp(),
	)

	return user
}

// ValidModeratorUser creates a valid User aggregate with moderator role for testing.
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
		ValidTimestamp(),
		ValidTimestamp(),
	)

	return user
}

// ValidTimestamp returns a valid timestamp for testing.
func ValidTimestamp() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}
