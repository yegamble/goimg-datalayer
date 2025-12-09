//nolint:testpackage // White-box testing required for internal mocks
package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// MockAlbumRepository is a mock implementation of gallery.AlbumRepository.
type MockAlbumRepository struct {
	mock.Mock
}

func (m *MockAlbumRepository) NextID() gallery.AlbumID {
	args := m.Called()
	return args.Get(0).(gallery.AlbumID)
}

func (m *MockAlbumRepository) FindByID(ctx context.Context, id gallery.AlbumID) (*gallery.Album, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	return args.Get(0).(*gallery.Album), args.Error(1) //nolint:wrapcheck
}

func (m *MockAlbumRepository) FindByOwner(ctx context.Context, ownerID identity.UserID) ([]*gallery.Album, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	return args.Get(0).([]*gallery.Album), args.Error(1) //nolint:wrapcheck
}

func (m *MockAlbumRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Album, int64, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*gallery.Album), args.Get(1).(int64), args.Error(2)
}

func (m *MockAlbumRepository) Save(ctx context.Context, album *gallery.Album) error {
	args := m.Called(ctx, album)
	return args.Error(0)
}

func (m *MockAlbumRepository) Delete(ctx context.Context, id gallery.AlbumID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlbumRepository) ExistsByID(ctx context.Context, id gallery.AlbumID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockUserRepository is a mock implementation of identity.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) NextID() identity.UserID {
	args := m.Called()
	return args.Get(0).(identity.UserID)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) ExistsByID(ctx context.Context, id identity.UserID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

//nolint:dupl // Test setup intentionally similar across test cases
func TestCreateAlbumHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("password123")
	now := time.Now()
	user := identity.ReconstructUser(userID, email, username, passwordHash, identity.RoleUser, identity.StatusActive, "", "", now, now)

	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "Test Album",
		Description: "Test description",
		Visibility:  "public",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockAlbums.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Album")).Return(nil)
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, albumID)

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCreateAlbumHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	cmd := CreateAlbumCommand{
		UserID:      "invalid-uuid",
		Title:       "Test Album",
		Description: "",
		Visibility:  "",
	}

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, albumID)
	assert.Contains(t, err.Error(), "invalid user id")

	mockUsers.AssertNotCalled(t, "FindByID")
	mockAlbums.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateAlbumHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "Test Album",
		Description: "",
		Visibility:  "",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(nil, identity.ErrUserNotFound)

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, albumID)
	assert.Contains(t, err.Error(), "find user")

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateAlbumHandler_Handle_InvalidTitle(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("password123")
	now := time.Now()
	user := identity.ReconstructUser(userID, email, username, passwordHash, identity.RoleUser, identity.StatusActive, "", "", now, now)

	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "", // Empty title
		Description: "",
		Visibility:  "",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil)

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, albumID)
	assert.Contains(t, err.Error(), "create album")

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateAlbumHandler_Handle_SaveError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("password123")
	now := time.Now()
	user := identity.ReconstructUser(userID, email, username, passwordHash, identity.RoleUser, identity.StatusActive, "", "", now, now)

	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "Test Album",
		Description: "",
		Visibility:  "",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockAlbums.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Album")).
		Return(errors.New("database error"))

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, albumID)
	assert.Contains(t, err.Error(), "save album")

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertExpectations(t)
	mockPublisher.AssertNotCalled(t, "Publish")
}

//nolint:dupl // Test setup intentionally similar across test cases
func TestCreateAlbumHandler_Handle_WithDescription(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("password123")
	now := time.Now()
	user := identity.ReconstructUser(userID, email, username, passwordHash, identity.RoleUser, identity.StatusActive, "", "", now, now)

	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "Test Album",
		Description: "A detailed description of the album",
		Visibility:  "private",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockAlbums.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Album")).Return(nil)
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, albumID)

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCreateAlbumHandler_Handle_EventPublishingError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockAlbums := new(MockAlbumRepository)
	mockUsers := new(MockUserRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zerolog.Nop()

	handler := NewCreateAlbumHandler(mockAlbums, mockUsers, mockPublisher, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("password123")
	now := time.Now()
	user := identity.ReconstructUser(userID, email, username, passwordHash, identity.RoleUser, identity.StatusActive, "", "", now, now)

	cmd := CreateAlbumCommand{
		UserID:      userID.String(),
		Title:       "Test Album",
		Description: "",
		Visibility:  "",
	}

	// Mock expectations
	mockUsers.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockAlbums.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Album")).Return(nil)
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event publishing error"))

	// Act
	albumID, err := handler.Handle(context.Background(), cmd)

	// Assert - Event publishing errors should NOT fail the operation
	require.NoError(t, err)
	assert.NotEmpty(t, albumID)

	mockUsers.AssertExpectations(t)
	mockAlbums.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}
