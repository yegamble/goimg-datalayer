package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	domainIdentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// MockSessionStoreForDelete is a mock implementation for delete tests.
type MockSessionStoreForDelete struct {
	revokeAllFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *MockSessionStoreForDelete) Create(ctx context.Context, session *identity.Session) error {
	panic("not implemented")
}

func (m *MockSessionStoreForDelete) Get(ctx context.Context, sessionID uuid.UUID) (*identity.Session, error) {
	panic("not implemented")
}

func (m *MockSessionStoreForDelete) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*identity.Session, error) {
	panic("not implemented")
}

func (m *MockSessionStoreForDelete) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSessionStoreForDelete) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	if m.revokeAllFunc != nil {
		return m.revokeAllFunc(ctx, userID)
	}
	return nil
}

func (m *MockSessionStoreForDelete) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	panic("not implemented")
}

func TestDeleteUserHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	mockSessionStore := &MockSessionStoreForDelete{}
	handler := commands.NewDeleteUserHandler(mockRepo, mockSessionStore)

	password := "ValidPassword123!"
	user := testhelpers.ValidUserWithPassword(password)
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockRepo.On("Delete", mock.Anything, userID).Return(nil)

	revokeAllWasCalled := false
	mockSessionStore.revokeAllFunc = func(ctx context.Context, uid uuid.UUID) error {
		assert.Equal(t, uuidParsed, uid)
		revokeAllWasCalled = true
		return nil
	}

	cmd := commands.DeleteUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		Password:    password,
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, revokeAllWasCalled, "All sessions should be revoked")
	mockRepo.AssertExpectations(t)
}

func TestDeleteUserHandler_Handle_Unauthorized(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	mockSessionStore := &MockSessionStoreForDelete{}
	handler := commands.NewDeleteUserHandler(mockRepo, mockSessionStore)

	userID := uuid.New()
	otherUserID := uuid.New()

	cmd := commands.DeleteUserCommand{
		UserID:      userID,
		RequestorID: otherUserID, // Different user trying to delete
		Password:    "password",
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestDeleteUserHandler_Handle_WrongPassword(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	mockSessionStore := &MockSessionStoreForDelete{}
	handler := commands.NewDeleteUserHandler(mockRepo, mockSessionStore)

	correctPassword := "CorrectPassword123!"
	user := testhelpers.ValidUserWithPassword(correctPassword)
	userID := user.ID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	cmd := commands.DeleteUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		Password:    "WrongPassword123!", // Wrong password
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password verification failed")
	mockRepo.AssertExpectations(t)
}

func TestDeleteUserHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	mockSessionStore := &MockSessionStoreForDelete{}
	handler := commands.NewDeleteUserHandler(mockRepo, mockSessionStore)

	userID := domainIdentity.NewUserID()
	uuidParsed := uuid.MustParse(userID.String())

	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, domainIdentity.ErrUserNotFound)

	cmd := commands.DeleteUserCommand{
		UserID:      uuidParsed,
		RequestorID: uuidParsed,
		Password:    "password",
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by id")
	mockRepo.AssertExpectations(t)
}

func TestDeleteUserHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := new(testhelpers.MockUserRepository)
	mockSessionStore := &MockSessionStoreForDelete{}
	handler := commands.NewDeleteUserHandler(mockRepo, mockSessionStore)

	// uuid.Nil actually parses successfully, so we mock FindByID to fail
	invalidUserID, _ := domainIdentity.ParseUserID(uuid.Nil.String())
	mockRepo.On("FindByID", mock.Anything, invalidUserID).Return(nil, domainIdentity.ErrUserNotFound)

	cmd := commands.DeleteUserCommand{
		UserID:      uuid.Nil,
		RequestorID: uuid.Nil,
		Password:    "password",
	}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find user by id")
	mockRepo.AssertExpectations(t)
}
