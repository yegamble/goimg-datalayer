package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestUpdateUserHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	user := testhelpers.ValidUser()
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	var savedUser *identity.User
	mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
		savedUser = u
		return true
	})).Return(nil)

	newDisplayName := "New Display Name"
	newBio := "This is my new bio"

	cmd := commands.UpdateUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed, // Same user updating their own profile
		DisplayName: &newDisplayName,
		Bio:         &newBio,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newDisplayName, result.DisplayName)
	assert.Equal(t, newBio, result.Bio)
	assert.NotNil(t, savedUser)
	assert.Equal(t, newDisplayName, savedUser.DisplayName())
	assert.Equal(t, newBio, savedUser.Bio())
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserHandler_Handle_PartialUpdate(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	user := testhelpers.ValidUser()
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).Return(nil)

	newDisplayName := "Only Display Name"

	cmd := commands.UpdateUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		DisplayName: &newDisplayName,
		Bio:         nil, // Not updating bio
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newDisplayName, result.DisplayName)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserHandler_Handle_NoChanges(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	user := testhelpers.ValidUser()
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	cmd := commands.UpdateUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		DisplayName: nil, // No updates
		Bio:         nil,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Save should not be called when no changes
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestUpdateUserHandler_Handle_Unauthorized(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()

	newDisplayName := "Hacker Name"

	cmd := commands.UpdateUserCommand{
		UserID:      userID,
		RequestorID: otherUserID, // Different user trying to update
		DisplayName: &newDisplayName,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestUpdateUserHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	userID := identity.NewUserID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, identity.ErrUserNotFound)

	newDisplayName := "New Name"

	cmd := commands.UpdateUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		DisplayName: &newDisplayName,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by id")
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserHandler_Handle_DisplayNameTooLong(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := commands.NewUpdateUserHandler(mockRepo)

	user := testhelpers.ValidUser()
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	// Create a display name that exceeds 100 characters
	longDisplayName := ""
	for i := 0; i < 101; i++ {
		longDisplayName += "a"
	}

	cmd := commands.UpdateUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		DisplayName: &longDisplayName,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update user profile")
	mockRepo.AssertExpectations(t)
}
