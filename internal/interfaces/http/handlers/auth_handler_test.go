package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestAuthHandler_Register(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		registerHandler := commands.NewRegisterUserHandler(
			mockRepo,
			mockPublisher,
			nil,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(registerHandler, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)
		mockRepo.On("FindByUsername", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)
		mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

		authHandler.Register(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.UserDTO
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, reqBody.Email, resp.Email)
		assert.Equal(t, reqBody.Username, resp.Username)
	})

	t.Run("EmailAlreadyExists", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockPublisher := new(testhelpers.MockEventPublisher)
		logger := zerolog.Nop()

		registerHandler := commands.NewRegisterUserHandler(
			mockRepo,
			mockPublisher,
			nil,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(registerHandler, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.RegisterRequest{
			Email:    "existing@example.com",
			Username: "newuser",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		email, err := identity.NewEmail("existing@example.com")
		require.NoError(t, err)
		username, err := identity.NewUsername("existing")
		require.NoError(t, err)
		pwd, err := identity.NewPasswordHash("StrongPassword123!")
		require.NoError(t, err)
		existingUser, err := identity.NewUser(email, username, pwd)
		require.NoError(t, err)

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(existingUser, nil).Once()

		authHandler.Register(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockJWT := new(testhelpers.MockJWTService)
		mockRefresh := new(testhelpers.MockRefreshTokenService)
		mockSession := new(testhelpers.MockSessionStore)
		mockMetrics := new(testhelpers.MockAuthMetricsRecorder)
		logger := zerolog.Nop()

		loginHandler := commands.NewLoginHandler(
			mockRepo,
			mockJWT,
			mockRefresh,
			mockSession,
			mockMetrics,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.LoginRequest{
			Email:    "test@example.com",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		email, err := identity.NewEmail(reqBody.Email)
		require.NoError(t, err)
		username, err := identity.NewUsername("testuser")
		require.NoError(t, err)
		passwordHash, err := identity.NewPasswordHash(reqBody.Password)
		require.NoError(t, err)

		user, err := identity.NewUser(email, username, passwordHash)
		require.NoError(t, err)
		err = user.Activate()
		require.NoError(t, err)

		mockMetrics.On("RecordLoginDelay", mock.Anything).Return()
		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
		mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access_token", nil)
		mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("refresh_token", &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}, nil)
		mockSession.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockJWT.On("GetTokenExpiration", "access_token").Return(time.Now().Add(15*time.Minute), nil)

		authHandler.Login(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockJWT := new(testhelpers.MockJWTService)
		mockRefresh := new(testhelpers.MockRefreshTokenService)
		mockSession := new(testhelpers.MockSessionStore)
		mockMetrics := new(testhelpers.MockAuthMetricsRecorder)
		logger := zerolog.Nop()

		loginHandler := commands.NewLoginHandler(
			mockRepo,
			mockJWT,
			mockRefresh,
			mockSession,
			mockMetrics,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.LoginRequest{
			Email:    "test@example.com",
			Password: "WrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		email, err := identity.NewEmail(reqBody.Email)
		require.NoError(t, err)
		username, err := identity.NewUsername("testuser")
		require.NoError(t, err)
		passwordHash, err := identity.NewPasswordHash("CorrectPassword123!")
		require.NoError(t, err)

		user, err := identity.NewUser(email, username, passwordHash)
		require.NoError(t, err)
		err = user.Activate()
		require.NoError(t, err)

		mockMetrics.On("RecordLoginDelay", mock.Anything).Return()
		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)

		authHandler.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockJWT := new(testhelpers.MockJWTService)
		mockRefresh := new(testhelpers.MockRefreshTokenService)
		mockSession := new(testhelpers.MockSessionStore)
		logger := zerolog.Nop()

		refreshHandler := commands.NewRefreshTokenHandler(
			mockRepo,
			mockJWT,
			mockRefresh,
			mockSession,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, refreshHandler, nil, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.RefreshRequest{
			RefreshToken: "valid_refresh_token",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mockRefresh.On("ValidateToken", mock.Anything, "valid_refresh_token").Return(&services.RefreshTokenMetadata{
			UserID:    uuid.New().String(),
			SessionID: uuid.New().String(),
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)
		mockRefresh.On("DetectAnomalies", mock.Anything, mock.Anything, mock.Anything).Return(false)
		mockRefresh.On("MarkAsUsed", mock.Anything, "valid_refresh_token").Return(nil)

		email, err := identity.NewEmail("test@example.com")
		require.NoError(t, err)
		username, err := identity.NewUsername("testuser")
		require.NoError(t, err)
		pwd, err := identity.NewPasswordHash("StrongPassword123!")
		require.NoError(t, err)

		user, err := identity.NewUser(email, username, pwd)
		require.NoError(t, err)
		err = user.Activate()
		require.NoError(t, err)

		mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(user, nil)
		mockSession.On("Exists", mock.Anything, mock.Anything).Return(true, nil)

		mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("new_access", nil)
		mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("new_refresh", &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}, nil)
		mockSession.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockJWT.On("GetTokenExpiration", "new_access").Return(time.Now().Add(15*time.Minute), nil)

		authHandler.Refresh(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockJWT := new(testhelpers.MockJWTService)
		mockRefresh := new(testhelpers.MockRefreshTokenService)
		mockSession := new(testhelpers.MockSessionStore)
		mockBlacklist := new(testhelpers.MockTokenBlacklist)
		logger := zerolog.Nop()

		logoutHandler := commands.NewLogoutHandler(
			mockRepo,
			mockJWT,
			mockRefresh,
			mockSession,
			mockBlacklist,
			&logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, logoutHandler, nil, nil, nil, nil, nil, logger)

		reqBody := handlers.LogoutRequest{
			RefreshToken: "refresh_token",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer access_token")
		rr := httptest.NewRecorder()

		userID := uuid.New()
		sessionID := uuid.New()
		ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", sessionID, false, true)
		req = req.WithContext(ctx)

		mockJWT.On("ExtractTokenID", "access_token").Return("jti", nil)
		mockJWT.On("GetTokenExpiration", "access_token").Return(time.Now().Add(time.Hour), nil)
		mockBlacklist.On("Add", mock.Anything, "jti", mock.Anything).Return(nil)
		mockRefresh.On("RevokeToken", mock.Anything, "refresh_token").Return(nil)
		mockSession.On("Revoke", mock.Anything, sessionID.String()).Return(nil)

		authHandler.Logout(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})
}
