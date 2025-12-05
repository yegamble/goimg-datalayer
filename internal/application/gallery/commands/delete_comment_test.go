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

func TestDeleteCommentHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     func(t *testing.T) commands.DeleteCommentCommand
		setup   func(t *testing.T, mocks *deleteCommentTestMocks)
		wantErr string
	}{
		{
			name: "successful deletion by author",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(4), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "successful deletion by moderator",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    "550e8400-e29b-41d4-a716-446655440002", // Moderator user ID
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				moderatorID, _ := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440002")
				commentID := testhelpers.ValidCommentIDParsed()
				imageID := testhelpers.ValidImageIDParsed()

				// Create moderator user with specific ID
				email, _ := identity.NewEmail("moderator@example.com")
				username, _ := identity.NewUsername("moduser123")
				passwordHash, _ := identity.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMye7WdZGIsgbRJHaC0G/YLnQ5zt1g/K7i2")
				moderator := identity.ReconstructUser(
					moderatorID,
					email,
					username,
					passwordHash,
					identity.RoleModerator,
					identity.StatusActive,
					"",
					"",
					testhelpers.ValidTimestamp(),
					testhelpers.ValidTimestamp(),
				)
				comment := testhelpers.ValidComment(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, moderatorID).Return(moderator, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(4), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				mocks.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
		},
		{
			name: "invalid comment id",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: "invalid-uuid",
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup:   func(t *testing.T, mocks *deleteCommentTestMocks) {},
			wantErr: "invalid comment id",
		},
		{
			name: "invalid user id",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    "invalid-uuid",
				}
			},
			setup:   func(t *testing.T, mocks *deleteCommentTestMocks) {},
			wantErr: "invalid user id",
		},
		{
			name: "user not found",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				userID := testhelpers.ValidUserIDParsed()
				mocks.users.On("FindByID", mock.Anything, userID).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: "find user",
		},
		{
			name: "comment not found",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				user := testhelpers.ValidUser(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).
					Return(nil, gallery.ErrCommentNotFound).Once()
			},
			wantErr: "find comment",
		},
		{
			name: "unauthorized - not author and not moderator",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    "550e8400-e29b-41d4-a716-446655440001", // Different user
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				differentUserID, _ := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440001")
				commentID := testhelpers.ValidCommentIDParsed()
				// Create regular user (not moderator)
				email, _ := identity.NewEmail("other@example.com")
				username, _ := identity.NewUsername("otheruser")
				passwordHash, _ := identity.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMye7WdZGIsgbRJHaC0G/YLnQ5zt1g/K7i2")
				user := identity.ReconstructUser(
					differentUserID,
					email,
					username,
					passwordHash,
					identity.RoleUser, // Regular user, not moderator
					identity.StatusActive,
					"",
					"",
					testhelpers.ValidTimestamp(),
					testhelpers.ValidTimestamp(),
				)
				comment := testhelpers.ValidComment(t)

				mocks.users.On("FindByID", mock.Anything, differentUserID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess.Error(),
		},
		{
			name: "comment delete failure",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "delete comment",
		},
		{
			name: "image not found for count update",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: "find image",
		},
		{
			name: "comment count failure",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).
					Return(int64(0), fmt.Errorf("count error")).Once()
			},
			wantErr: "get comment count",
		},
		{
			name: "image save failure after count update",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(4), nil).Once()
				mocks.images.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: "update image comment count",
		},
		{
			name: "event publishing failure - should still succeed",
			cmd: func(t *testing.T) commands.DeleteCommentCommand {
				return commands.DeleteCommentCommand{
					CommentID: testhelpers.ValidCommentID,
					UserID:    testhelpers.ValidUserID,
				}
			},
			setup: func(t *testing.T, mocks *deleteCommentTestMocks) {
				commentID := testhelpers.ValidCommentIDParsed()
				userID := testhelpers.ValidUserIDParsed()
				imageID := testhelpers.ValidImageIDParsed()
				user := testhelpers.ValidUser(t)
				comment := testhelpers.ValidComment(t)
				image := testhelpers.ValidImage(t)

				mocks.users.On("FindByID", mock.Anything, userID).Return(user, nil).Once()
				mocks.comments.On("FindByID", mock.Anything, commentID).Return(comment, nil).Once()
				mocks.comments.On("Delete", mock.Anything, commentID).Return(nil).Once()
				mocks.images.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				mocks.comments.On("CountByImage", mock.Anything, imageID).Return(int64(4), nil).Once()
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
			mocks := newDeleteCommentTestMocks()
			tt.setup(t, mocks)

			handler := commands.NewDeleteCommentHandler(
				mocks.images,
				mocks.comments,
				mocks.users,
				mocks.publisher,
				&mocks.logger,
			)

			// Act
			cmd := tt.cmd(t)
			err := handler.Handle(context.Background(), cmd)

			// Assert
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			mocks.images.AssertExpectations(t)
			mocks.comments.AssertExpectations(t)
			mocks.users.AssertExpectations(t)
			mocks.publisher.AssertExpectations(t)
		})
	}
}

// deleteCommentTestMocks holds all mocks needed for delete comment testing.
type deleteCommentTestMocks struct {
	images    *testhelpers.MockImageRepository
	comments  *testhelpers.MockCommentRepository
	users     *testhelpers.MockUserRepository
	publisher *testhelpers.MockEventPublisher
	logger    zerolog.Logger
}

// newDeleteCommentTestMocks creates a new set of mocks for delete comment testing.
func newDeleteCommentTestMocks() *deleteCommentTestMocks {
	return &deleteCommentTestMocks{
		images:    new(testhelpers.MockImageRepository),
		comments:  new(testhelpers.MockCommentRepository),
		users:     new(testhelpers.MockUserRepository),
		publisher: new(testhelpers.MockEventPublisher),
		logger:    zerolog.Nop(),
	}
}
