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

func TestUnfollowUserHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	logger := zerolog.Nop()

	handler := commands.NewUnfollowUserHandler(mockFollows, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Mock expectations
	mockFollows.On("Delete", mock.Anything, followerID, followedID).
		Return(nil)

	cmd := commands.UnfollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	mockFollows.AssertExpectations(t)
}

func TestUnfollowUserHandler_Handle_Idempotent_NotFollowing(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	logger := zerolog.Nop()

	handler := commands.NewUnfollowUserHandler(mockFollows, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// Mock expectations - follow relationship doesn't exist
	mockFollows.On("Delete", mock.Anything, followerID, followedID).
		Return(identity.ErrFollowNotFound)

	cmd := commands.UnfollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert - should succeed (idempotent operation)
	require.NoError(t, err)
	mockFollows.AssertExpectations(t)
}

func TestUnfollowUserHandler_Handle_InvalidFollowerID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	logger := zerolog.Nop()

	handler := commands.NewUnfollowUserHandler(mockFollows, &logger)

	cmd := commands.UnfollowUserCommand{
		FollowerID: "invalid-uuid",
		FollowedID: identity.NewUserID().String(),
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid follower id")
	mockFollows.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

func TestUnfollowUserHandler_Handle_InvalidFollowedID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	logger := zerolog.Nop()

	handler := commands.NewUnfollowUserHandler(mockFollows, &logger)

	cmd := commands.UnfollowUserCommand{
		FollowerID: identity.NewUserID().String(),
		FollowedID: "invalid-uuid",
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid followed id")
	mockFollows.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

func TestUnfollowUserHandler_Handle_MultipleCallsSucceed(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	logger := zerolog.Nop()

	handler := commands.NewUnfollowUserHandler(mockFollows, &logger)

	followerID := identity.NewUserID()
	followedID := identity.NewUserID()

	// First call succeeds
	mockFollows.On("Delete", mock.Anything, followerID, followedID).
		Return(nil).
		Once()

	// Second call returns not found (already deleted)
	mockFollows.On("Delete", mock.Anything, followerID, followedID).
		Return(identity.ErrFollowNotFound).
		Once()

	cmd := commands.UnfollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}

	// Act - First call
	err1 := handler.Handle(context.Background(), cmd)
	require.NoError(t, err1)

	// Act - Second call (idempotent)
	err2 := handler.Handle(context.Background(), cmd)
	require.NoError(t, err2)

	// Assert
	mockFollows.AssertExpectations(t)
}
