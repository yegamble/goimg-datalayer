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

func TestDeleteAlbumHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.DeleteAlbumCommand
		setup   func(t *testing.T, mocks *deleteAlbumTestMocks)
		wantErr string
	}{
		{
			name: "successful deletion",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Delete", mock.Anything, albumID).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "invalid album id",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
			},
			setup:   func(t *testing.T, mocks *deleteAlbumTestMocks) {},
			wantErr: "invalid album id",
		},
		{
			name: "invalid user id",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  "invalid-uuid",
			},
			setup:   func(t *testing.T, mocks *deleteAlbumTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "album not found - idempotent success",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				mocks.albums.On("FindByID", mock.Anything, albumID).
					Return(nil, gallery.ErrAlbumNotFound).Once()
			},
			wantErr: "", // Idempotent - no error
		},
		{
			name: "unauthorized - user does not own album",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
			},
			wantErr: "unauthorized",
		},
		{
			name: "database error on find",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				mocks.albums.On("FindByID", mock.Anything, albumID).
					Return(nil, fmt.Errorf("database error")).Once()
			},
			wantErr: "find album",
		},
		{
			name: "delete operation failure",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Delete", mock.Anything, albumID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "delete album",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.DeleteAlbumCommand{
				AlbumID: testhelpers.ValidAlbumID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, mocks *deleteAlbumTestMocks) {
				albumID := testhelpers.ValidAlbumIDParsed()
				album := testhelpers.ValidAlbum(t)

				mocks.albums.On("FindByID", mock.Anything, albumID).Return(album, nil).Once()
				mocks.albums.On("Delete", mock.Anything, albumID).Return(nil).Once()
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
			mocks := newDeleteAlbumTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewDeleteAlbumHandler(
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

// deleteAlbumTestMocks holds all mocks needed for delete album testing.
type deleteAlbumTestMocks struct {
	albums    *testhelpers.MockAlbumRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newDeleteAlbumTestMocks creates a new set of mocks for delete album testing.
func newDeleteAlbumTestMocks() *deleteAlbumTestMocks {
	return &deleteAlbumTestMocks{
		albums:    new(testhelpers.MockAlbumRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
