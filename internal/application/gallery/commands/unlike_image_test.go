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

//nolint:funlen // Table-driven test with comprehensive test cases
func TestUnlikeImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.UnlikeImageCommand
		setup   func(t *testing.T, mocks *unlikeTestMocks)
		wantErr string
	}{
		//nolint:dupl // Test setup intentionally similar across test cases
		{
			name: "successful unlike",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				// GetLikeCount called twice: once before unlike decision, once after unlike
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.likes.On("Unlike", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "not liked - idempotent success",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
			},
			wantErr: "", // No error, idempotent
		},
		{
			name: "invalid user id",
			cmd: commands.UnlikeImageCommand{
				UserID:  "invalid-uuid",
				ImageID: testhelpers.ValidImageID,
			},
			setup:   func(t *testing.T, mocks *unlikeTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "invalid image id",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: "invalid-uuid",
			},
			setup:   func(t *testing.T, mocks *unlikeTestMocks) {},
			wantErr: "invalid image id",
		},
		{
			name: "user not found",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				mocks.users.On("FindByID", mock.Anything, userID).
					Return(nil, fmt.Errorf("user not found")).Once()
			},
			wantErr: "find user",
		},
		{
			name: "image not found",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: "find image",
		},
		{
			name: "has liked check failure",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).
					Return(false, fmt.Errorf("database error")).Once()
			},
			wantErr: "check has liked",
		},
		//nolint:dupl // Test setup intentionally similar across test cases
		{
			name: "unlike failure",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.likes.On("Unlike", mock.Anything, userID, imageID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "remove like",
		},
		{
			name: "like count failure - first call",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				// First GetLikeCount call fails (before unlike decision)
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: "get like count",
		},
		{
			name: "like count failure - second call after unlike",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.likes.On("Unlike", mock.Anything, userID, imageID).Return(nil).Once()
				// Second GetLikeCount call fails (after unlike)
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: "get like count",
		},
		{
			name: "image save failure after unlike",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.likes.On("Unlike", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "update image like count",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.UnlikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *unlikeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.likes.On("Unlike", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
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
			mocks := newUnlikeTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewUnlikeImageHandler(
				mocks.images,
				mocks.likes,
				mocks.users,
				mocks.publisher,
				&mocks.logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.False(t, result.Liked)
				assert.GreaterOrEqual(t, result.LikeCount, int64(0))
			}

			mocks.images.AssertExpectations(t)
			mocks.likes.AssertExpectations(t)
			mocks.users.AssertExpectations(t)
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// unlikeTestMocks holds all mocks needed for unlike testing.
type unlikeTestMocks struct {
	images    *testhelpers.MockImageRepository
	likes     *testhelpers.MockLikeRepository
	users     *testhelpers.MockUserRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newUnlikeTestMocks creates a new set of mocks for unlike testing.
func newUnlikeTestMocks() *unlikeTestMocks {
	return &unlikeTestMocks{
		images:    new(testhelpers.MockImageRepository),
		likes:     new(testhelpers.MockLikeRepository),
		users:     new(testhelpers.MockUserRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
