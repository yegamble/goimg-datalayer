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
)

func TestRemoveImageFromAlbumHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.RemoveImageFromAlbumCommand
		setup   func(t *testing.T, mocks *removeImageTestMocks)
		wantErr string
	}{
		{
			name: "successful remove image from album",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albumImages.On("RemoveImageFromAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "invalid album id",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: "invalid-uuid",
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup:   func(_ *testing.T, _ *removeImageTestMocks) {},
			wantErr: "invalid album id",
		},
		{
			name: "invalid image id",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
			},
			setup:   func(_ *testing.T, _ *removeImageTestMocks) {},
			wantErr: "invalid image id",
		},
		{
			name: "invalid user id",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  "invalid-uuid",
			},
			setup:   func(_ *testing.T, _ *removeImageTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "album not found",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				mocks.albums.On("FindByID", mock.Anything, albumID).
					Return(nil, gallery.ErrAlbumNotFound).Once()
			},
			wantErr: "find album",
		},
		{
			name: "unauthorized - user does not own album",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "unauthorized",
		},
		{
			name: "failed to remove image from album",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albumImages.On("RemoveImageFromAlbum", mock.Anything, albumID, imageID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "remove image from album",
		},
		{
			name: "failed to save album",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albumImages.On("RemoveImageFromAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "save album",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.RemoveImageFromAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *removeImageTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albumImages.On("RemoveImageFromAlbum", mock.Anything, albumID, imageID).Return(nil).Once()
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
			mocks := newRemoveImageTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewRemoveImageFromAlbumHandler(
				mocks.albums,
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
			mocks.albumImages.AssertExpectations(t)
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// removeImageTestMocks holds all mocks needed for remove image testing.
type removeImageTestMocks struct {
	albums      *testhelpers.MockAlbumRepository
	albumImages *testhelpers.MockAlbumImageRepository
	publisher   *testhelpers.MockEventPublisher
	logger      zerolog.Logger
}

// newRemoveImageTestMocks creates a new set of mocks for remove image testing.
func newRemoveImageTestMocks() *removeImageTestMocks {
	return &removeImageTestMocks{
		albums:      new(testhelpers.MockAlbumRepository),
		albumImages: new(testhelpers.MockAlbumImageRepository),
		publisher:   new(testhelpers.MockEventPublisher),
		logger:      zerolog.Nop(),
	}
}
