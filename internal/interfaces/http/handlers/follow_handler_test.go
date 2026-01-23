package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockFollowUserHandler is a mock implementation of FollowUserHandler
type MockFollowUserHandler struct {
	mock.Mock
}

func (m *MockFollowUserHandler) Handle(ctx context.Context, cmd commands.FollowUserCommand) error {
	args := m.Called(ctx, cmd)
	return args.Error(0)
}

// MockUnfollowUserHandler is a mock implementation of UnfollowUserHandler
type MockUnfollowUserHandler struct {
	mock.Mock
}

func (m *MockUnfollowUserHandler) Handle(ctx context.Context, cmd commands.UnfollowUserCommand) error {
	args := m.Called(ctx, cmd)
	return args.Error(0)
}

// MockGetFollowersHandler is a mock implementation of GetFollowersHandler
type MockGetFollowersHandler struct {
	mock.Mock
}

func (m *MockGetFollowersHandler) Handle(ctx context.Context, q queries.GetFollowersQuery) (*dto.FollowersListDTO, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.FollowersListDTO), args.Error(1)
}

// MockGetFollowingHandler is a mock implementation of GetFollowingHandler
type MockGetFollowingHandler struct {
	mock.Mock
}

func (m *MockGetFollowingHandler) Handle(ctx context.Context, q queries.GetFollowingQuery) (*dto.FollowingListDTO, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.FollowingListDTO), args.Error(1)
}

func TestFollowHandler_FollowUser_Success(t *testing.T) {
	// Arrange
	mockFollowHandler := new(MockFollowUserHandler)
	handler := NewFollowHandler(
		mockFollowHandler,
		nil, // unfollow handler not used
		nil, // get followers handler not used
		nil, // get following handler not used
		zerolog.Nop(),
	)

	followerID := uuid.New()
	followedID := uuid.New()

	// Mock expectations
	expectedCmd := commands.FollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}
	mockFollowHandler.On("Handle", mock.Anything, expectedCmd).Return(nil)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodPost, "/users/"+followedID.String()+"/follow", nil)
	rec := httptest.NewRecorder()

	// Add user context (simulating JWT middleware)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, followerID)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRoleKey, "user")
	ctx = context.WithValue(ctx, middleware.SessionIDKey, uuid.New())
	req = req.WithContext(ctx)

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", followedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.FollowUser(rec, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockFollowHandler.AssertExpectations(t)
}

func TestFollowHandler_FollowUser_Unauthorized(t *testing.T) {
	// Arrange
	handler := NewFollowHandler(
		nil, nil, nil, nil,
		zerolog.Nop(),
	)

	req := httptest.NewRequest(http.MethodPost, "/users/123/follow", nil)
	rec := httptest.NewRecorder()

	// No user context (not authenticated)

	// Act
	handler.FollowUser(rec, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFollowHandler_FollowUser_AlreadyFollowing(t *testing.T) {
	// Arrange
	mockFollowHandler := new(MockFollowUserHandler)
	handler := NewFollowHandler(
		mockFollowHandler,
		nil, nil, nil,
		zerolog.Nop(),
	)

	followerID := uuid.New()
	followedID := uuid.New()

	// Mock expectations - return ErrFollowAlreadyExists
	mockFollowHandler.On("Handle", mock.Anything, mock.Anything).
		Return(identity.ErrFollowAlreadyExists)

	// Create request with context and path parameter
	req := httptest.NewRequest(http.MethodPost, "/users/"+followedID.String()+"/follow", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, followerID)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRoleKey, "user")
	ctx = context.WithValue(ctx, middleware.SessionIDKey, uuid.New())
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", followedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.FollowUser(rec, req)

	// Assert
	assert.Equal(t, http.StatusConflict, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, "Conflict", errResp["title"])
	mockFollowHandler.AssertExpectations(t)
}

func TestFollowHandler_FollowUser_CannotFollowSelf(t *testing.T) {
	// Arrange
	mockFollowHandler := new(MockFollowUserHandler)
	handler := NewFollowHandler(
		mockFollowHandler,
		nil, nil, nil,
		zerolog.Nop(),
	)

	userID := uuid.New()

	// Mock expectations - return ErrCannotFollowSelf
	mockFollowHandler.On("Handle", mock.Anything, mock.Anything).
		Return(identity.ErrCannotFollowSelf)

	// Create request with context and path parameter
	req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/follow", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRoleKey, "user")
	ctx = context.WithValue(ctx, middleware.SessionIDKey, uuid.New())
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.FollowUser(rec, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, "Bad Request", errResp["title"])
	mockFollowHandler.AssertExpectations(t)
}

func TestFollowHandler_UnfollowUser_Success(t *testing.T) {
	// Arrange
	mockUnfollowHandler := new(MockUnfollowUserHandler)
	handler := NewFollowHandler(
		nil, // follow handler not used
		mockUnfollowHandler,
		nil, nil,
		zerolog.Nop(),
	)

	followerID := uuid.New()
	followedID := uuid.New()

	// Mock expectations
	expectedCmd := commands.UnfollowUserCommand{
		FollowerID: followerID.String(),
		FollowedID: followedID.String(),
	}
	mockUnfollowHandler.On("Handle", mock.Anything, expectedCmd).Return(nil)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodDelete, "/users/"+followedID.String()+"/follow", nil)
	rec := httptest.NewRecorder()

	// Add user context (simulating JWT middleware)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, followerID)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRoleKey, "user")
	ctx = context.WithValue(ctx, middleware.SessionIDKey, uuid.New())
	req = req.WithContext(ctx)

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", followedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.UnfollowUser(rec, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockUnfollowHandler.AssertExpectations(t)
}

func TestFollowHandler_UnfollowUser_Idempotent(t *testing.T) {
	// Arrange
	mockUnfollowHandler := new(MockUnfollowUserHandler)
	handler := NewFollowHandler(
		nil,
		mockUnfollowHandler,
		nil, nil,
		zerolog.Nop(),
	)

	followerID := uuid.New()
	followedID := uuid.New()

	// Mock expectations - handler returns nil even if not following
	mockUnfollowHandler.On("Handle", mock.Anything, mock.Anything).Return(nil)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodDelete, "/users/"+followedID.String()+"/follow", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, followerID)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRoleKey, "user")
	ctx = context.WithValue(ctx, middleware.SessionIDKey, uuid.New())
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", followedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.UnfollowUser(rec, req)

	// Assert - should succeed even if not following
	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockUnfollowHandler.AssertExpectations(t)
}

func TestFollowHandler_GetFollowers_Success(t *testing.T) {
	// Arrange
	mockGetFollowersHandler := new(MockGetFollowersHandler)
	handler := NewFollowHandler(
		nil, nil,
		mockGetFollowersHandler,
		nil,
		zerolog.Nop(),
	)

	userID := uuid.New()

	// Mock expectations
	expectedQuery := queries.GetFollowersQuery{
		UserID: userID.String(),
		Limit:  20,
		Offset: 0,
	}

	mockResult := &dto.FollowersListDTO{
		Followers: []dto.FollowUserDTO{
			{
				UserID:      uuid.New().String(),
				Username:    "follower1",
				DisplayName: "Follower One",
				Bio:         "Bio 1",
			},
		},
		TotalCount: 1,
		Offset:     0,
		Limit:      20,
	}

	mockGetFollowersHandler.On("Handle", mock.Anything, expectedQuery).Return(mockResult, nil)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/followers", nil)
	rec := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.GetFollowers(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var result dto.FollowersListDTO
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Followers, 1)
	assert.Equal(t, "follower1", result.Followers[0].Username)
	mockGetFollowersHandler.AssertExpectations(t)
}

func TestFollowHandler_GetFollowers_WithPagination(t *testing.T) {
	// Arrange
	mockGetFollowersHandler := new(MockGetFollowersHandler)
	handler := NewFollowHandler(
		nil, nil,
		mockGetFollowersHandler,
		nil,
		zerolog.Nop(),
	)

	userID := uuid.New()

	// Mock expectations with custom pagination
	expectedQuery := queries.GetFollowersQuery{
		UserID: userID.String(),
		Limit:  10,
		Offset: 5,
	}

	mockResult := &dto.FollowersListDTO{
		Followers:  []dto.FollowUserDTO{},
		TotalCount: 50,
		Offset:     5,
		Limit:      10,
	}

	mockGetFollowersHandler.On("Handle", mock.Anything, expectedQuery).Return(mockResult, nil)

	// Create request with pagination query params
	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/followers?limit=10&offset=5", nil)
	rec := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.GetFollowers(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var result dto.FollowersListDTO
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 50, result.TotalCount)
	assert.Equal(t, 5, result.Offset)
	assert.Equal(t, 10, result.Limit)
	mockGetFollowersHandler.AssertExpectations(t)
}

func TestFollowHandler_GetFollowing_Success(t *testing.T) {
	// Arrange
	mockGetFollowingHandler := new(MockGetFollowingHandler)
	handler := NewFollowHandler(
		nil, nil, nil,
		mockGetFollowingHandler,
		zerolog.Nop(),
	)

	userID := uuid.New()

	// Mock expectations
	expectedQuery := queries.GetFollowingQuery{
		UserID: userID.String(),
		Limit:  20,
		Offset: 0,
	}

	mockResult := &dto.FollowingListDTO{
		Following: []dto.FollowUserDTO{
			{
				UserID:      uuid.New().String(),
				Username:    "followed1",
				DisplayName: "Followed One",
				Bio:         "Bio 1",
			},
			{
				UserID:      uuid.New().String(),
				Username:    "followed2",
				DisplayName: "Followed Two",
				Bio:         "Bio 2",
			},
		},
		TotalCount: 2,
		Offset:     0,
		Limit:      20,
	}

	mockGetFollowingHandler.On("Handle", mock.Anything, expectedQuery).Return(mockResult, nil)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/following", nil)
	rec := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.GetFollowing(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var result dto.FollowingListDTO
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Following, 2)
	assert.Equal(t, "followed1", result.Following[0].Username)
	assert.Equal(t, "followed2", result.Following[1].Username)
	mockGetFollowingHandler.AssertExpectations(t)
}

func TestFollowHandler_GetFollowers_UserNotFound(t *testing.T) {
	// Arrange
	mockGetFollowersHandler := new(MockGetFollowersHandler)
	handler := NewFollowHandler(
		nil, nil,
		mockGetFollowersHandler,
		nil,
		zerolog.Nop(),
	)

	userID := uuid.New()

	// Mock expectations - return user not found error
	mockGetFollowersHandler.On("Handle", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)

	// Create request with path parameter
	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/followers", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Act
	handler.GetFollowers(rec, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, "Not Found", errResp["title"])
	mockGetFollowersHandler.AssertExpectations(t)
}

func TestParsePaginationParams_DefaultValues(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Act
	limit, offset := parsePaginationParams(req)

	// Assert
	assert.Equal(t, 20, limit)
	assert.Equal(t, 0, offset)
}

func TestParsePaginationParams_CustomValues(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test?limit=50&offset=10", nil)

	// Act
	limit, offset := parsePaginationParams(req)

	// Assert
	assert.Equal(t, 50, limit)
	assert.Equal(t, 10, offset)
}

func TestParsePaginationParams_MaxLimit(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test?limit=200", nil)

	// Act
	limit, offset := parsePaginationParams(req)

	// Assert - should cap at 100
	assert.Equal(t, 100, limit)
	assert.Equal(t, 0, offset)
}

func TestParsePaginationParams_MinLimit(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test?limit=0", nil)

	// Act
	limit, offset := parsePaginationParams(req)

	// Assert - should enforce min of 1
	assert.Equal(t, 1, limit)
	assert.Equal(t, 0, offset)
}

func TestParsePaginationParams_NegativeOffset(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test?offset=-5", nil)

	// Act
	limit, offset := parsePaginationParams(req)

	// Assert - negative offset should be ignored
	assert.Equal(t, 20, limit)
	assert.Equal(t, 0, offset)
}
