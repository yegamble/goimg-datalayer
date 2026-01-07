package queries_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestGetFollowingHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Create followed user with specific ID
	followedID := identity.NewUserID()
	followedUser := testhelpers.ValidActiveUserWithIDAndUsername(followedID, "followed@example.com", "followed")

	// Create follow relationship
	follow, _ := identity.NewFollow(followerUserID, followedID)
	follow.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 20, 0).
		Return([]*identity.Follow{follow}, 1, nil)
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(followedUser, nil)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Following, 1)
	assert.Equal(t, followedID.String(), result.Following[0].UserID)
	assert.Equal(t, "followed", result.Following[0].Username)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowingHandler_Handle_EmptyList(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 20, 0).
		Return([]*identity.Follow{}, 0, nil)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Following)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowingHandler_Handle_MultipleFollowing(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Create multiple followed users
	followed1ID := identity.NewUserID()
	followed1 := createFollowerUserWithID(followed1ID, "followed1@example.com", "followed1")

	followed2ID := identity.NewUserID()
	followed2 := createFollowerUserWithID(followed2ID, "followed2@example.com", "followed2")

	// Create follow relationships
	follow1, _ := identity.NewFollow(followerUserID, followed1ID)
	follow1.ClearEvents()
	follow2, _ := identity.NewFollow(followerUserID, followed2ID)
	follow2.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 20, 0).
		Return([]*identity.Follow{follow1, follow2}, 2, nil)
	mockUsers.On("FindByID", mock.Anything, followed1ID).
		Return(followed1, nil)
	mockUsers.On("FindByID", mock.Anything, followed2ID).
		Return(followed2, nil)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Following, 2)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowingHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(nil, identity.ErrUserNotFound)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user")

	mockUsers.AssertExpectations(t)
	mockFollows.AssertNotCalled(t, "FindFollowing", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetFollowingHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	query := queries.GetFollowingQuery{
		UserID: "invalid-uuid",
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user id")

	mockUsers.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
	mockFollows.AssertNotCalled(t, "FindFollowing", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetFollowingHandler_Handle_SkipsDeletedFollowedUsers(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Create followed users
	followed1ID := identity.NewUserID()
	followed1 := createFollowerUserWithID(followed1ID, "followed1@example.com", "followed1")

	followed2ID := identity.NewUserID() // This user was deleted

	// Create follow relationships
	follow1, _ := identity.NewFollow(followerUserID, followed1ID)
	follow1.ClearEvents()
	follow2, _ := identity.NewFollow(followerUserID, followed2ID)
	follow2.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 20, 0).
		Return([]*identity.Follow{follow1, follow2}, 2, nil)
	mockUsers.On("FindByID", mock.Anything, followed1ID).
		Return(followed1, nil)
	mockUsers.On("FindByID", mock.Anything, followed2ID).
		Return(nil, identity.ErrUserNotFound) // Deleted user

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert - should succeed but only include existing followed user
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)   // Total count includes deleted user
	assert.Len(t, result.Following, 1)      // But only 1 followed user in results
	assert.Equal(t, followed1ID.String(), result.Following[0].UserID)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowingHandler_Handle_Pagination(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Create followed user
	followedID := identity.NewUserID()
	followed := createFollowerUserWithID(followedID, "followed@example.com", "followed")

	// Create follow relationship
	follow, _ := identity.NewFollow(followerUserID, followedID)
	follow.ClearEvents()

	// Mock expectations with pagination
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 10, 20).
		Return([]*identity.Follow{follow}, 100, nil) // Total 100, but only 1 in this page
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(followed, nil)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  10,
		Offset: 20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.TotalCount)
	assert.Len(t, result.Following, 1)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 20, result.Offset)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowingHandler_Handle_IncludesUserProfileData(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowingHandler(mockFollows, mockUsers)

	followerUserID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUser()

	// Create followed user with profile data
	followedID := identity.NewUserID()
	followedUser := testhelpers.ValidActiveUserWithIDAndUsername(followedID, "followed@example.com", "followed")
	_ = followedUser.UpdateProfile("Followed User", "This is my bio")
	followedUser.ClearEvents()

	// Create follow relationship
	follow, _ := identity.NewFollow(followerUserID, followedID)
	follow.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, followerUserID).
		Return(followerUser, nil)
	mockFollows.On("FindFollowing", mock.Anything, followerUserID, 20, 0).
		Return([]*identity.Follow{follow}, 1, nil)
	mockUsers.On("FindByID", mock.Anything, followedID).
		Return(followedUser, nil)

	query := queries.GetFollowingQuery{
		UserID: followerUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Following, 1)
	assert.Equal(t, followedID.String(), result.Following[0].UserID)
	assert.Equal(t, "followed", result.Following[0].Username)
	assert.Equal(t, "Followed User", result.Following[0].DisplayName)
	assert.Equal(t, "This is my bio", result.Following[0].Bio)
	assert.False(t, result.Following[0].FollowedAt.IsZero())

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}
