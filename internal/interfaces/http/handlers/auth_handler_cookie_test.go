package handlers_test

import (
	"bytes"
	"crypto/tls"
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
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
)

func TestAuthHandler_Login_SetsCookie(t *testing.T) {
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

	authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, logger)

	reqBody := handlers.LoginRequest{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Simulate HTTPS request to check Secure flag
	req.TLS = &tls.ConnectionState{}

	rr := httptest.NewRecorder()

	email, _ := identity.NewEmail(reqBody.Email)
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash(reqBody.Password)
	user, _ := identity.NewUser(email, username, passwordHash)
	_ = user.Activate()

	mockMetrics.On("RecordLoginDelay", mock.Anything).Return()
	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
	mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access_token", nil)
	// Return a proper RefreshTokenMetadata object
	mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("refresh_token_value", &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}, nil)
	mockSession.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockJWT.On("GetTokenExpiration", "access_token").Return(time.Now().Add(15*time.Minute), nil)

	authHandler.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check Cookie
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}

	require.NotNil(t, refreshCookie, "Refresh token cookie should be set")
	assert.Equal(t, "refresh_token_value", refreshCookie.Value)
	assert.True(t, refreshCookie.HttpOnly, "Cookie should be HttpOnly")
	assert.Equal(t, "/api/v1/auth/refresh", refreshCookie.Path, "Cookie path should be restricted")
	assert.Equal(t, http.SameSiteStrictMode, refreshCookie.SameSite, "Cookie should be SameSite=Strict")
	// Since we mocked TLS, Secure should be true (if implemented correctly)
	assert.True(t, refreshCookie.Secure, "Cookie should be Secure")
}

func TestAuthHandler_Refresh_ReadsCookie(t *testing.T) {
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

	authHandler := handlers.NewAuthHandler(nil, nil, refreshHandler, nil, nil, logger)

	// Empty body to force reading from cookie
	reqBody := handlers.RefreshRequest{
		RefreshToken: "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set the cookie
	cookie := &http.Cookie{
		Name:  "refresh_token",
		Value: "cookie_refresh_token",
	}
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()

	// Mock expecting the token from the cookie
	mockRefresh.On("ValidateToken", mock.Anything, "cookie_refresh_token").Return(&services.RefreshTokenMetadata{
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	mockRefresh.On("DetectAnomalies", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockRefresh.On("MarkAsUsed", mock.Anything, "cookie_refresh_token").Return(nil)

	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	pwd, _ := identity.NewPasswordHash("StrongPassword123!")
	user, _ := identity.NewUser(email, username, pwd)
	_ = user.Activate()

	mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(user, nil)
	mockSession.On("Exists", mock.Anything, mock.Anything).Return(true, nil)

	mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("new_access", nil)
	mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("new_refresh", &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}, nil)
	mockSession.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockJWT.On("GetTokenExpiration", "new_access").Return(time.Now().Add(15*time.Minute), nil)

	authHandler.Refresh(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
