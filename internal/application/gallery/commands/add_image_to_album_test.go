package commands_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestAddImageToAlbumHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.AddImageToAlbumCommand
		setup   func(t *testing.T, mocks *albumImageTestMocks)
		wantErr string
	}{
		{
			name: "successful add image to album",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				t.Helper()
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)
				image := testhelpers.ValidImage(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.albumImages.On("AddImageToAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "invalid album id",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: "invalid-uuid",
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup:   func(t *testing.T, mocks *albumImageTestMocks) {},
			wantErr: "invalid album id",
		},
		{
			name: "invalid image id",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
			},
			setup:   func(t *testing.T, mocks *albumImageTestMocks) {},
			wantErr: "invalid image id",
		},
		{
			name: "invalid user id",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  "invalid-uuid",
			},
			setup:   func(t *testing.T, mocks *albumImageTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "album not found",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				mocks.albums.On("FindByID", mock.Anything, albumID).
					Return(nil, gallery.ErrAlbumNotFound).Once()
			},
			wantErr: "find album",
		},
		{
			name: "unauthorized - user does not own album",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				t.Helper()
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "unauthorized",
		},
		{
			name: "image not found",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: "find image",
		},
		{
			name: "unauthorized - user does not own image",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				// Create album owned by different user
				differentUserID, _ := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440001")
				album, _ := gallery.NewAlbum(differentUserID, "Test Album")
				// Create image owned by original user
				image := testhelpers.ValidImage(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: "unauthorized",
		},
		{
			name: "failed to add image to album",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)
				image := testhelpers.ValidImage(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.albumImages.On("AddImageToAlbum", mock.Anything, albumID, imageID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "add image to album",
		},
		{
			name: "failed to save album",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)
				image := testhelpers.ValidImage(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.albumImages.On("AddImageToAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "save album",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.AddImageToAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *albumImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)
				image := testhelpers.ValidImage(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.albumImages.On("AddImageToAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).
					Return(fmt.Errorf("event bus unavailable")).Maybe()
			},
			wantErr: "", // Should still succeed
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			mocks := newAlbumImageTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewAddImageToAlbumHandler(
				mocks.albums,
				mocks.images,
				mocks.albumImages,
				mocks.publisher,
				&mocks.logger,
			)

			// Act
			err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			mocks.albums.AssertExpectations(t)
			mocks.images.AssertExpectations(t)
			mocks.albumImages.AssertExpectations(t)
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// albumImageTestMocks holds all mocks needed for album-image testing.
type albumImageTestMocks struct {
	albums      *testhelpers.MockAlbumRepository
	images      *testhelpers.MockImageRepository
	albumImages *testhelpers.MockAlbumImageRepository
	publisher   *testhelpers.MockEventPublisher
	logger      zerolog.Logger
}

// newAlbumImageTestMocks creates a new set of mocks for album-image testing.
func newAlbumImageTestMocks() *albumImageTestMocks {
	return &albumImageTestMocks{
		albums:      new(testhelpers.MockAlbumRepository),
		images:      new(testhelpers.MockImageRepository),
		albumImages: new(testhelpers.MockAlbumImageRepository),
		publisher:   new(testhelpers.MockEventPublisher),
		logger:      zerolog.Nop(),
	}
}
