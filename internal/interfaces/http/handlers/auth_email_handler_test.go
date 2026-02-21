package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestAuthHandler_ForgotPassword(t *testing.T) {
	t.Run("Success_AlwaysReturns200", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockTokenRepo := new(testhelpers.MockTokenRepository)
		mockEmailSender := new(testhelpers.MockEmailSender)
		logger := zerolog.Nop()

		forgotHandler := commands.NewRequestPasswordResetHandler(
			mockRepo, mockTokenRepo, mockEmailSender, &logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, forgotHandler, nil, nil, nil, logger)

		reqBody := handlers.ForgotPasswordRequest{Email: "nonexistent@example.com"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)

		authHandler.ForgotPassword(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("BadRequest_InvalidBody", func(t *testing.T) {
		logger := zerolog.Nop()
		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader([]byte(`{"email":""}`)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		authHandler.ForgotPassword(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	t.Run("BadRequest_InvalidBody", func(t *testing.T) {
		logger := zerolog.Nop()
		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		authHandler.ResetPassword(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidToken_ReturnsError", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockTokenRepo := new(testhelpers.MockTokenRepository)
		mockSession := new(testhelpers.MockSessionStore)
		logger := zerolog.Nop()

		resetHandler := commands.NewResetPasswordHandler(
			mockTokenRepo, mockRepo, mockSession, logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, resetHandler, nil, nil, logger)

		reqBody := handlers.ResetPasswordRequest{
			Token:       "invalid-token",
			NewPassword: "NewStrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mockTokenRepo.On("FindValidPasswordResetToken", mock.Anything, "invalid-token").
			Return(nil, identity.ErrTokenNotFound)

		authHandler.ResetPassword(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	t.Run("BadRequest_InvalidBody", func(t *testing.T) {
		logger := zerolog.Nop()
		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		authHandler.VerifyEmail(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidToken_ReturnsError", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockTokenRepo := new(testhelpers.MockTokenRepository)
		logger := zerolog.Nop()

		verifyHandler := commands.NewVerifyEmailHandler(
			mockTokenRepo, mockRepo, logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, verifyHandler, logger)

		reqBody := handlers.VerifyEmailRequest{Token: "bad-token"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mockTokenRepo.On("FindValidEmailVerificationToken", mock.Anything, "bad-token").
			Return(nil, identity.ErrTokenNotFound)

		authHandler.VerifyEmail(rr, req)

		assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusUnauthorized)
	})
}

func TestAuthHandler_ResendVerification(t *testing.T) {
	t.Run("Unauthorized_NoContext", func(t *testing.T) {
		logger := zerolog.Nop()
		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", nil)
		rr := httptest.NewRecorder()

		authHandler.ResendVerification(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Success_WithAuthContext", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockTokenRepo := new(testhelpers.MockTokenRepository)
		mockEmailSender := new(testhelpers.MockEmailSender)
		logger := zerolog.Nop()

		sendHandler := commands.NewSendVerificationEmailHandler(
			mockRepo, mockTokenRepo, mockEmailSender, logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, sendHandler, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", nil)
		rr := httptest.NewRecorder()

		userID := uuid.New()
		sessionID := uuid.New()
		ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", sessionID, false, true)
		req = req.WithContext(ctx)

		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("testuser")
		pwd, _ := identity.NewPasswordHash("StrongPassword123!")
		user, _ := identity.NewUser(email, username, pwd)

		mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(user, nil)
		mockTokenRepo.On("CreateEmailVerificationToken", mock.Anything, mock.Anything, mock.Anything).Return("verify-token", nil)
		mockEmailSender.On("SendVerificationEmail", mock.Anything, "test@example.com", "verify-token").Return(nil)

		authHandler.ResendVerification(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Error_ReturnsInternalServerError", func(t *testing.T) {
		mockRepo := new(testhelpers.MockUserRepository)
		mockTokenRepo := new(testhelpers.MockTokenRepository)
		mockEmailSender := new(testhelpers.MockEmailSender)
		logger := zerolog.Nop()

		sendHandler := commands.NewSendVerificationEmailHandler(
			mockRepo, mockTokenRepo, mockEmailSender, logger,
		)

		authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, sendHandler, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", nil)
		rr := httptest.NewRecorder()

		userID := uuid.New()
		sessionID := uuid.New()
		ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", sessionID, false, true)
		req = req.WithContext(ctx)

		mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)

		authHandler.ResendVerification(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
