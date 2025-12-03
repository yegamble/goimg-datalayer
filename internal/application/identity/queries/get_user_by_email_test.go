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

func TestGetUserByEmailHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserByEmailHandler(mockRepo)

	user := testhelpers.ValidUser()
	email := testhelpers.ValidEmailVO()

	mockRepo.On("FindByEmail", mock.Anything, email).Return(user, nil)

	query := queries.GetUserByEmailQuery{
		Email: testhelpers.ValidEmail,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testhelpers.ValidEmail, result.Email)
	assert.Equal(t, testhelpers.ValidUsername, result.Username)
	mockRepo.AssertExpectations(t)
}

func TestGetUserByEmailHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserByEmailHandler(mockRepo)

	email := testhelpers.AlternateEmail()
	mockRepo.On("FindByEmail", mock.Anything, email).Return(nil, identity.ErrUserNotFound)

	query := queries.GetUserByEmailQuery{
		Email: "alternate@example.com",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by email")
	mockRepo.AssertExpectations(t)
}

func TestGetUserByEmailHandler_Handle_InvalidEmail(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserByEmailHandler(mockRepo)

	query := queries.GetUserByEmailQuery{
		Email: "not-an-email",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestGetUserByEmailHandler_Handle_EmptyEmail(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	handler := queries.NewGetUserByEmailHandler(mockRepo)

	query := queries.GetUserByEmailQuery{
		Email: "",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}
