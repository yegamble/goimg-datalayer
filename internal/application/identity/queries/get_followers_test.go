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

func TestGetFollowersHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()
	targetUser := testhelpers.ValidActiveUser()

	// Create follower user with specific ID
	followerID := identity.NewUserID()
	followerUser := testhelpers.ValidActiveUserWithIDAndUsername(followerID, "follower@example.com", "follower")

	// Create follow relationship
	follow, _ := identity.NewFollow(followerID, targetUserID)
	follow.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(targetUser, nil)
	mockFollows.On("FindFollowers", mock.Anything, targetUserID, 20, 0).
		Return([]*identity.Follow{follow}, 1, nil)
	mockUsers.On("FindByID", mock.Anything, followerID).
		Return(followerUser, nil)

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Followers, 1)
	assert.Equal(t, followerID.String(), result.Followers[0].UserID)
	assert.Equal(t, "follower", result.Followers[0].Username)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowersHandler_Handle_EmptyList(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()
	targetUser := testhelpers.ValidActiveUser()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(targetUser, nil)
	mockFollows.On("FindFollowers", mock.Anything, targetUserID, 20, 0).
		Return([]*identity.Follow{}, 0, nil)

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Followers)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowersHandler_Handle_MultipleFollowers(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()
	targetUser := testhelpers.ValidActiveUser()

	// Create multiple followers
	follower1ID := identity.NewUserID()
	follower1 := createFollowerUserWithID(follower1ID, "follower1@example.com", "follower1")

	follower2ID := identity.NewUserID()
	follower2 := createFollowerUserWithID(follower2ID, "follower2@example.com", "follower2")

	// Create follow relationships
	follow1, _ := identity.NewFollow(follower1ID, targetUserID)
	follow1.ClearEvents()
	follow2, _ := identity.NewFollow(follower2ID, targetUserID)
	follow2.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(targetUser, nil)
	mockFollows.On("FindFollowers", mock.Anything, targetUserID, 20, 0).
		Return([]*identity.Follow{follow1, follow2}, 2, nil)
	mockUsers.On("FindByID", mock.Anything, follower1ID).
		Return(follower1, nil)
	mockUsers.On("FindByID", mock.Anything, follower2ID).
		Return(follower2, nil)

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Followers, 2)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowersHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(nil, identity.ErrUserNotFound)

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
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
	mockFollows.AssertNotCalled(t, "FindFollowers", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetFollowersHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	query := queries.GetFollowersQuery{
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
	mockFollows.AssertNotCalled(t, "FindFollowers", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetFollowersHandler_Handle_SkipsDeletedFollowers(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()
	targetUser := testhelpers.ValidActiveUser()

	// Create followers
	follower1ID := identity.NewUserID()
	follower1 := createFollowerUserWithID(follower1ID, "follower1@example.com", "follower1")

	follower2ID := identity.NewUserID() // This follower was deleted

	// Create follow relationships
	follow1, _ := identity.NewFollow(follower1ID, targetUserID)
	follow1.ClearEvents()
	follow2, _ := identity.NewFollow(follower2ID, targetUserID)
	follow2.ClearEvents()

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(targetUser, nil)
	mockFollows.On("FindFollowers", mock.Anything, targetUserID, 20, 0).
		Return([]*identity.Follow{follow1, follow2}, 2, nil)
	mockUsers.On("FindByID", mock.Anything, follower1ID).
		Return(follower1, nil)
	mockUsers.On("FindByID", mock.Anything, follower2ID).
		Return(nil, identity.ErrUserNotFound) // Deleted user

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
		Limit:  20,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert - should succeed but only include existing follower
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount) // Total count includes deleted user
	assert.Len(t, result.Followers, 1)    // But only 1 follower in results
	assert.Equal(t, follower1ID.String(), result.Followers[0].UserID)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

func TestGetFollowersHandler_Handle_Pagination(t *testing.T) {
	t.Parallel()

	// Arrange
	mockFollows := new(testhelpers.MockFollowRepository)
	mockUsers := new(testhelpers.MockUserRepository)

	handler := queries.NewGetFollowersHandler(mockFollows, mockUsers)

	targetUserID := identity.NewUserID()
	targetUser := testhelpers.ValidActiveUser()

	// Create follower
	followerID := identity.NewUserID()
	follower := createFollowerUserWithID(followerID, "follower@example.com", "follower")

	// Create follow relationship
	follow, _ := identity.NewFollow(followerID, targetUserID)
	follow.ClearEvents()

	// Mock expectations with pagination
	mockUsers.On("FindByID", mock.Anything, targetUserID).
		Return(targetUser, nil)
	mockFollows.On("FindFollowers", mock.Anything, targetUserID, 10, 20).
		Return([]*identity.Follow{follow}, 100, nil) // Total 100, but only 1 in this page
	mockUsers.On("FindByID", mock.Anything, followerID).
		Return(follower, nil)

	query := queries.GetFollowersQuery{
		UserID: targetUserID.String(),
		Limit:  10,
		Offset: 20,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.TotalCount)
	assert.Len(t, result.Followers, 1)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 20, result.Offset)

	mockUsers.AssertExpectations(t)
	mockFollows.AssertExpectations(t)
}

// Helper function to create a follower user with specific ID
func createFollowerUserWithID(userID identity.UserID, email, username string) *identity.User {
	return testhelpers.ValidActiveUserWithIDAndUsername(userID, email, username)
}
