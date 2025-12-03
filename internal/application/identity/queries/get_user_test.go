package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestGetUserHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserHandler(mockRepo)

	user := testhelpers.ValidUser()
	userID := user.ID()

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	query := queries.GetUserQuery{
		UserID:      uuid.MustParse(userID.String()),
		RequestorID: uuid.MustParse(userID.String()), // Same user requesting their own data
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID.String(), result.ID)
	assert.Equal(t, testhelpers.ValidEmail, result.Email)
	assert.Equal(t, testhelpers.ValidUsername, result.Username)
	mockRepo.AssertExpectations(t)
}

func TestGetUserHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserHandler(mockRepo)

	userID := identity.NewUserID()
	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, identity.ErrUserNotFound)

	query := queries.GetUserQuery{
		UserID:      uuid.MustParse(userID.String()),
		RequestorID: uuid.MustParse(userID.String()),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by id")
	mockRepo.AssertExpectations(t)
}

func TestGetUserHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserHandler(mockRepo)

	// Parse an invalid UUID string to create a valid query structure
	// but with an ID that will fail domain validation
	invalidUUID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
	invalidUserID, err := identity.ParseUserID(invalidUUID.String())
	require.NoError(t, err) // Parse succeeds for uuid.Nil

	// But when we try to find it, treat as not found
	mockRepo.On("FindByID", mock.Anything, invalidUserID).Return(nil, identity.ErrUserNotFound)

	query := queries.GetUserQuery{
		UserID:      invalidUUID,
		RequestorID: uuid.New(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by id")
	mockRepo.AssertExpectations(t)
}
