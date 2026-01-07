package commands_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestFollowUserHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()
	followedUser := testhelpers.ValidActiveUser()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(followedUser, nil)
	mockFollows.On("Exists", mock.Anything, followerID, followedID).
		Return(false, nil)
	mockFollows.On("Save", mock.Anything, mock.AnythingOfType("*identity.Follow")).
		Return(nil)

	cmd := commands.FollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestFollowUserHandler_Handle_AlreadyFollowing(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()
	followedUser := testhelpers.ValidActiveUser()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(followedUser, nil)
	mockFollows.On("Exists", mock.Anything, followerID, followedID).
		Return(true, nil) // Already following

	cmd := commands.FollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.ErrorIs(t, err, identity.ErrFollowAlreadyExists)
	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
	mockFollows.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestFollowUserHandler_Handle_FollowSelf(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	userID := identity.NewUserID()
	user := testhelpers.ValidActiveUser()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).
		Return(user, nil)
	mockFollows.On("Exists", mock.Anything, userID, userID).
		Return(false, nil)

	cmd := commands.FollowUserCommand{
		FollowerID: userID.String(),
		FollowedID: userID.String(), // Same user
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create follow")
	// Domain should reject self-follow
	mockFollows.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestFollowUserHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(nil, identity.ErrUserNotFound)

	cmd := commands.FollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user to follow not found")
	mockUsers.AssertExpectations(t)
	mockFollows.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
	mockFollows.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestFollowUserHandler_Handle_InvalidFollowerID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	cmd := commands.FollowUserCommand{
		FollowerID: "invalid-uuid",
		FollowedID: identity.NewUserID().String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid follower id")
	mockUsers.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
	mockFollows.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
}

func TestFollowUserHandler_Handle_InvalidFollowedID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	cmd := commands.FollowUserCommand{
		FollowerID: identity.NewUserID().String(),
		FollowedID: "invalid-uuid",
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid followed id")
	mockUsers.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
	mockFollows.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
}

func TestFollowUserHandler_Handle_CannotFollowInactiveUser(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)
	logger := zerolog.Nop()

	handler := commands.NewFollowUserHandler(mockFollows, mockUsers, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Create a suspended user
	email, _ := identity.NewEmail("suspended@example.com")
	username, _ := identity.NewUsername("suspended")
	passwordHash, _ := identity.NewPasswordHash("Password123!")
	suspendedUser, _ := identity.NewUser(email, username, passwordHash)
	_ = suspendedUser.Suspend("policy violation")
	suspendedUser.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(suspendedUser, nil)

	cmd := commands.FollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot follow inactive user")
	mockUsers.AssertExpectations(t)
	mockFollows.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
	mockFollows.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}
