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

func TestAddCommentHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.AddCommentCommand
		setup   func(t *testing.T, mocks *commentTestMocks)
		wantErr error
		assert  func(t *testing.T, mocks *commentTestMocks, commentID string, err error)
	}{
		{
			name: "successful comment addition",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "This is a great image!",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).Return(nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(5), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
				mocks.publisher.AssertExpectations(t)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.AddCommentCommand{
				UserID:  "invalid-uuid",
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user id")
				assert.Empty(t, commentID)
			},
		},
		{
			name: "invalid image id",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: "invalid-uuid",
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image id")
				assert.Empty(t, commentID)
			},
		},
		{
			name: "user not found",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				mocks.users.On("FindByID", mock.Anything, userID).
					Return(nil, fmt.Errorf("user not found")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "find user")
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
			},
		},
		{
			name: "image not found",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "find image")
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - private image not owned by user",
			cmd: commands.AddCommentCommand{
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				differentUserID, _ := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440001")
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)
				// Set image to private
				_ = image.UpdateVisibility(gallery.VisibilityPrivate)

				mocks.users.On("FindByID", mock.Anything, differentUserID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, testhelpers.ValidImageIDParsed()).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
			},
		},
		{
			name: "empty content after sanitization",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "   ", // Only whitespace
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrCommentRequired,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrCommentRequired)
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
			},
		},
		{
			name: "comment too long",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: string(make([]byte, gallery.MaxCommentLength+1)), // Exceeds max length
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrCommentTooLong,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrCommentTooLong)
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
			},
		},
		{
			name: "sanitizes HTML content",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "<script>alert('xss')</script>Nice picture!",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				// HTML should be stripped, only "Nice picture!" remains
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).Return(nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(1), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
				mocks.publisher.AssertExpectations(t)
			},
		},
		{
			name: "comment save failure",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save comment")
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
			},
		},
		{
			name: "comment count failure",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).Return(nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "get comment count")
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
			},
		},
		{
			name: "image save failure after comment",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).Return(nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(5), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "update image comment count")
				assert.Empty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
			},
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: commands.AddCommentCommand{
				UserID:  testhelpers.ValidUserID,
				ImageID: testhelpers.ValidImageID,
				Content: "Test comment",
			},
			setup: func(t *testing.T, mocks *commentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Comment")).Return(nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(5), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).
					Return(fmt.Errorf("event bus unavailable")).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, mocks *commentTestMocks, commentID string, err error) {
				// Should still succeed even if event publishing fails
				require.NoError(t, err)
				assert.NotEmpty(t, commentID)
				mocks.users.AssertExpectations(t)
				mocks.images.AssertExpectations(t)
				mocks.comments.AssertExpectations(t)
				mocks.publisher.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			mocks := newCommentTestMocks()
			if tt.setup != nil {
				tt.setup(t, mocks)
			}

			handler := commands.NewAddCommentHandler(
				mocks.images,
				mocks.comments,
				mocks.users,
				mocks.publisher,
				&mocks.logger,
			)

			// Act
			commentID, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			switch {
			case tt.assert != nil:
				tt.assert(t, mocks, commentID, err)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, commentID)
			default:
				require.NoError(t, err)
				assert.NotEmpty(t, commentID)
			}
		})
	}
}

// commentTestMocks holds all mocks needed for comment testing.
type commentTestMocks struct {
	images    *testhelpers.MockImageRepository
	comments  *testhelpers.MockCommentRepository
	users     *testhelpers.MockUserRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newCommentTestMocks creates a new set of mocks for comment testing.
func newCommentTestMocks() *commentTestMocks {
	return &commentTestMocks{
		images:    new(testhelpers.MockImageRepository),
		comments:  new(testhelpers.MockCommentRepository),
		users:     new(testhelpers.MockUserRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
