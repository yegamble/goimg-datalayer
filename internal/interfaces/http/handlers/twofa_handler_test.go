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
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func setupTOTPService(t *testing.T) *security.TOTPService {
	key := make([]byte, 32)
	encryptor, err := security.NewSecretEncryptor(key)
	require.NoError(t, err)

	service, err := security.NewTOTPService(security.DefaultTOTPConfig(), encryptor)
	require.NoError(t, err)

	return service
}

func TestTwoFAHandler_Setup(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockTOTPRepo := new(testhelpers.MockTOTPRepository)
		mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
		totpService := setupTOTPService(t)
		logger := zerolog.Nop()

		setupHandler := commands.NewSetup2FAHandler(
			mockUserRepo,
			mockTOTPRepo,
			mockBackupRepo,
			totpService,
			&logger,
		)

		handler := handlers.NewTwoFAHandler(setupHandler, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
		userID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		// Mock expectations
		mockTOTPRepo.On("IsEnabled", mock.Anything, userID).Return(false, nil)

		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("user")
		pwd, _ := identity.NewPasswordHash("hash")

		user := identity.ReconstructUser(
			userID, email, username, pwd, identity.RoleUser, identity.StatusActive,
			"User", "Bio", 0, time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
		)

		mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		mockTOTPRepo.On("Save", mock.Anything, userID, mock.Anything).Return(nil)
		mockBackupRepo.On("SaveAll", mock.Anything, userID, mock.Anything).Return(nil)

		handler.Setup(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.Setup2FAResponseDTO
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Secret)
		assert.NotEmpty(t, resp.BackupCodes)
	})
}

func TestTwoFAHandler_Verify(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockTOTPRepo := new(testhelpers.MockTOTPRepository)
		totpService := setupTOTPService(t)
		logger := zerolog.Nop()

		verifyHandler := commands.NewVerify2FAHandler(
			mockUserRepo,
			mockTOTPRepo,
			totpService,
			&logger,
		)

		handler := handlers.NewTwoFAHandler(nil, verifyHandler, nil, nil, nil, nil, logger)

		// Generate a valid code
		setupRes, _ := totpService.GenerateSecret("test@example.com")
		code, _ := totpService.GenerateCurrentCode(setupRes.EncryptedSecret)

		reqBody := handlers.Verify2FARequest{Code: code}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
		userID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		// Mock expectations
		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("user")
		pwd, _ := identity.NewPasswordHash("hash")
		user := identity.ReconstructUser(
			userID, email, username, pwd, identity.RoleUser, identity.StatusActive,
			"User", "Bio", 0, time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
		)
		mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		// Setup secret
		secret, _ := identity.NewTOTPSecret(setupRes.EncryptedSecret, "test@example.com")
		mockTOTPRepo.On("FindByUserID", mock.Anything, userID).Return(&secret, nil)

		mockTOTPRepo.On("Save", mock.Anything, userID, mock.Anything).Return(nil)

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("InvalidCode", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockTOTPRepo := new(testhelpers.MockTOTPRepository)
		totpService := setupTOTPService(t)
		logger := zerolog.Nop()

		verifyHandler := commands.NewVerify2FAHandler(
			mockUserRepo,
			mockTOTPRepo,
			totpService,
			&logger,
		)

		handler := handlers.NewTwoFAHandler(nil, verifyHandler, nil, nil, nil, nil, logger)

		reqBody := handlers.Verify2FARequest{Code: "invalid"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
		userID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		// Setup secret
		setupRes, _ := totpService.GenerateSecret("test@example.com")
		secret, _ := identity.NewTOTPSecret(setupRes.EncryptedSecret, "test@example.com")

		// User mock
		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("user")
		pwd, _ := identity.NewPasswordHash("hash")
		user := identity.ReconstructUser(
			userID, email, username, pwd, identity.RoleUser, identity.StatusActive,
			"User", "Bio", 0, time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
		)
		mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		mockTOTPRepo.On("FindByUserID", mock.Anything, userID).Return(&secret, nil)

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestTwoFAHandler_Disable(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(testhelpers.MockUserRepository)
		mockTOTPRepo := new(testhelpers.MockTOTPRepository)
		mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
		logger := zerolog.Nop()

		disableHandler := commands.NewDisable2FAHandler(
			mockUserRepo,
			mockTOTPRepo,
			mockBackupRepo,
			nil, // totpService not needed
			&logger,
		)

		handler := handlers.NewTwoFAHandler(nil, nil, disableHandler, nil, nil, nil, logger)

		reqBody := handlers.Disable2FARequest{Password: "StrongPassword123!"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/2fa/disable", bytes.NewReader(body))
		userID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("user")
		pwd, err := identity.NewPasswordHash("StrongPassword123!")
		require.NoError(t, err)
		user := identity.ReconstructUser(
			userID, email, username, pwd, identity.RoleUser, identity.StatusActive,
			"User", "Bio", 0, time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
		)
		mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		secret, _ := identity.NewTOTPSecret([]byte("secret"), "test@example.com")
		secret.Enable()
		mockTOTPRepo.On("FindByUserID", mock.Anything, userID).Return(&secret, nil)

		mockTOTPRepo.On("Delete", mock.Anything, userID).Return(nil)
		mockBackupRepo.On("DeleteByUserID", mock.Anything, userID).Return(nil)

		handler.Disable(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestTwoFAHandler_Status(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockTOTPRepo := new(testhelpers.MockTOTPRepository)
		mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
		logger := zerolog.Nop()

		statusQuery := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

		handler := handlers.NewTwoFAHandler(nil, nil, nil, nil, nil, statusQuery, logger)

		req := httptest.NewRequest(http.MethodGet, "/auth/2fa/status", nil)
		userID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		secret, _ := identity.NewTOTPSecret([]byte("enc"), "test")
		secret.Enable()
		mockTOTPRepo.On("FindByUserID", mock.Anything, userID).Return(&secret, nil)

		mockBackupRepo.On("CountUnused", mock.Anything, userID).Return(5, nil)

		handler.Status(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.TwoFactorStatusDTO
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Enabled)
		assert.Equal(t, 5, resp.BackupCodesRemaining)
	})
}
