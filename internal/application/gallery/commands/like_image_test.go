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

func TestLikeImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.LikeImageCommand
		setup   func(t *testing.T, mocks *likeTestMocks)
		wantErr string
	}{
		{
			name: "successful like",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				// GetLikeCount called twice: once before like decision, once after like
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.likes.On("Like", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "already liked - idempotent success",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(true, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
			},
			wantErr: "", // No error, idempotent
		},
		{
			name: "invalid user id",
			cmd: commands.LikeImageCommand{
				UserID:  "invalid-uuid",
				ImageID: testhelpers.ValidImageID,
			},
			setup:   func(t *testing.T, mocks *likeTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "invalid image id",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: "invalid-uuid",
			},
			setup:   func(t *testing.T, mocks *likeTestMocks) {},
			wantErr: "invalid image id",
		},
		{
			name: "user not found",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				mocks.users.On("FindByID", mock.Anything, userID).
					Return(nil, fmt.Errorf("user not found")).Once()
			},
			wantErr: "find user",
		},
		{
			name: "image not found",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
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
			name: "unauthorized - private image not owned by user",
			cmd: commands.LikeImageCommand{
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				differentUserID, _ := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440001")
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)
				_ = image.UpdateVisibility(gallery.VisibilityPrivate)

				mocks.users.On("FindByID", mock.Anything, differentUserID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess.Error(),
		},
		{
			name: "has liked check failure",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
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
		{
			name: "like creation failure",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.likes.On("Like", mock.Anything, userID, imageID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "create like",
		},
		{
			name: "like count failure - first call",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				// First GetLikeCount call fails (before like decision)
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: "get like count",
		},
		{
			name: "like count failure - second call after like",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.likes.On("Like", mock.Anything, userID, imageID).Return(nil).Once()
				// Second GetLikeCount call fails (after like)
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: "get like count",
		},
		{
			name: "image save failure after like",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.likes.On("Like", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "update image like count",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.LikeImageCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
			},
			setup: func(t *testing.T, mocks *likeTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.likes.On("HasLiked", mock.Anything, userID, imageID).Return(false, nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(9), nil).Once()
				mocks.likes.On("Like", mock.Anything, userID, imageID).Return(nil).Once()
				mocks.likes.On("GetLikeCount", mock.Anything, imageID).Return(int64(10), nil).Once()
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
			mocks := newLikeTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewLikeImageHandler(
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
				assert.True(t, result.Liked)
				assert.Greater(t, result.LikeCount, int64(0))
			}

			mocks.images.AssertExpectations(t)
			mocks.likes.AssertExpectations(t)
			mocks.users.AssertExpectations(t)
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// likeTestMocks holds all mocks needed for like testing.
type likeTestMocks struct {
	images    *testhelpers.MockImageRepository
	likes     *testhelpers.MockLikeRepository
	users     *testhelpers.MockUserRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newLikeTestMocks creates a new set of mocks for like testing.
func newLikeTestMocks() *likeTestMocks {
	return &likeTestMocks{
		images:    new(testhelpers.MockImageRepository),
		likes:     new(testhelpers.MockLikeRepository),
		users:     new(testhelpers.MockUserRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
