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

func TestUpdateAlbumHandler_Handle(t *testing.T) {
	t.Parallel()

	newTitle := "Updated Album Title"
	newDescription := "Updated album description"
	newVisibility := "private"

	tests := []struct {
		name    string
		cmd     commands.UpdateAlbumCommand
		setup   func(t *testing.T, mocks *updateAlbumTestMocks)
		wantErr string
	}{
		{
			name: "successful update - all fields",
			cmd: commands.UpdateAlbumCommand{
				AlbumID:     testhelpers.ValidAlbumID,
				UserID:      testhelpers.ValidUserID,
				Title:       &newTitle,
				Description: &newDescription,
				Visibility:  &newVisibility,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "successful update - title only",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "successful update - description only",
			cmd: commands.UpdateAlbumCommand{
				AlbumID:     testhelpers.ValidAlbumID,
				UserID:      testhelpers.ValidUserID,
				Description: &newDescription,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "successful update - visibility only",
			cmd: commands.UpdateAlbumCommand{
				AlbumID:    testhelpers.ValidAlbumID,
				UserID:     testhelpers.ValidUserID,
				Visibility: &newVisibility,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "no fields updated - early return",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				// No fields to update
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				// No Save should be called
			},
			wantErr: "",
		},
		{
			name: "invalid album id",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup:   func(t *testing.T, mocks *updateAlbumTestMocks) {},
			wantErr: "invalid album id",
		},
		{
			name: "invalid user id",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  "invalid-uuid",
				Title:   &newTitle,
			},
			setup:   func(t *testing.T, mocks *updateAlbumTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "album not found",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				mocks.albums.On("FindByID", mock.Anything, albumID).
					Return(nil, gallery.ErrAlbumNotFound).Once()
			},
			wantErr: "find album",
		},
		{
			name: "unauthorized - user does not own album",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
				Title:   &newTitle,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "unauthorized",
		},
		{
			name: "invalid title",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				Title:   stringPtr(""), // Empty title
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "update title",
		},
		{
			name: "invalid visibility",
			cmd: commands.UpdateAlbumCommand{
				AlbumID:    testhelpers.ValidAlbumID,
				UserID:     testhelpers.ValidUserID,
				Visibility: stringPtr("invalid"),
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "invalid visibility",
		},
		{
			name: "album save failure",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "save album",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.UpdateAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, mocks *updateAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
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
			mocks := newUpdateAlbumTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewUpdateAlbumHandler(
				mocks.albums,
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
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// updateAlbumTestMocks holds all mocks needed for update album testing.
type updateAlbumTestMocks struct {
	albums    *testhelpers.MockAlbumRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newUpdateAlbumTestMocks creates a new set of mocks for update album testing.
func newUpdateAlbumTestMocks() *updateAlbumTestMocks {
	return &updateAlbumTestMocks{
		albums:    new(testhelpers.MockAlbumRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
