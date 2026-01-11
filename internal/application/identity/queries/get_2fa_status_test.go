package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestGet2FAStatusHandler_Handle_Enabled(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()
	now := time.Now().UTC()
	verifiedAt := now.Add(-24 * time.Hour)

	// Create enabled TOTP secret
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		true,       // enabled
		verifiedAt, // verified timestamp
	)

	// Mock expectations
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)
	mockBackupRepo.On("CountUnused", mock.Anything, userID).
		Return(5, nil)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.False(t, result.SetupPending)
	assert.Equal(t, 5, result.BackupCodesRemaining)
	assert.NotNil(t, result.EnabledAt)
	assert.Equal(t, verifiedAt, *result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertExpectations(t)
}

func TestGet2FAStatusHandler_Handle_SetupPending(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()

	// Create TOTP secret that's not enabled yet (setup pending)
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		false,       // not enabled
		time.Time{}, // not verified
	)

	// Mock expectations
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Enabled)
	assert.True(t, result.SetupPending)
	assert.Equal(t, 0, result.BackupCodesRemaining)
	assert.Nil(t, result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertNotCalled(t, "CountUnused", mock.Anything, mock.Anything)
}

func TestGet2FAStatusHandler_Handle_NotSetup(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()

	// Mock expectations - no TOTP secret found
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(nil, assert.AnError)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Enabled)
	assert.False(t, result.SetupPending)
	assert.Equal(t, 0, result.BackupCodesRemaining)
	assert.Nil(t, result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertNotCalled(t, "CountUnused", mock.Anything, mock.Anything)
}

func TestGet2FAStatusHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	query := queries.Get2FAStatusQuery{
		UserID: "invalid-uuid",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")

	mockTOTPRepo.AssertNotCalled(t, "FindByUserID", mock.Anything, mock.Anything)
	mockBackupRepo.AssertNotCalled(t, "CountUnused", mock.Anything, mock.Anything)
}

func TestGet2FAStatusHandler_Handle_BackupCodeCountError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()
	now := time.Now().UTC()
	verifiedAt := now.Add(-24 * time.Hour)

	// Create enabled TOTP secret
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		true,
		verifiedAt,
	)

	// Mock expectations - backup code count fails, but query should still succeed
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)
	mockBackupRepo.On("CountUnused", mock.Anything, userID).
		Return(0, assert.AnError)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert - should succeed but with 0 backup codes
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.False(t, result.SetupPending)
	assert.Equal(t, 0, result.BackupCodesRemaining) // Defaults to 0 on error
	assert.NotNil(t, result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertExpectations(t)
}

func TestGet2FAStatusHandler_Handle_ZeroBackupCodes(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()
	now := time.Now().UTC()
	verifiedAt := now.Add(-24 * time.Hour)

	// Create enabled TOTP secret
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		true,
		verifiedAt,
	)

	// Mock expectations - user has used all backup codes
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)
	mockBackupRepo.On("CountUnused", mock.Anything, userID).
		Return(0, nil)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.False(t, result.SetupPending)
	assert.Equal(t, 0, result.BackupCodesRemaining)
	assert.NotNil(t, result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertExpectations(t)
}

func TestGet2FAStatusHandler_Handle_EmptyUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	query := queries.Get2FAStatusQuery{
		UserID: "",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")

	mockTOTPRepo.AssertNotCalled(t, "FindByUserID", mock.Anything, mock.Anything)
	mockBackupRepo.AssertNotCalled(t, "CountUnused", mock.Anything, mock.Anything)
}

func TestGet2FAStatusHandler_Handle_EnabledWithoutVerification(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()

	// Create TOTP secret that's enabled but not verified (edge case)
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		true,        // enabled flag set
		time.Time{}, // but not verified
	)

	// Mock expectations
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert - enabled requires both flag and verification timestamp
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Enabled)      // IsEnabled() checks both conditions
	assert.False(t, result.SetupPending) // Has secret but enabled flag is set
	assert.Equal(t, 0, result.BackupCodesRemaining)
	assert.Nil(t, result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertNotCalled(t, "CountUnused", mock.Anything, mock.Anything)
}

func TestGet2FAStatusHandler_Handle_MultipleBackupCodes(t *testing.T) {
	t.Parallel()

	// Arrange
	mockTOTPRepo := new(testhelpers.MockTOTPRepository)
	mockBackupRepo := new(testhelpers.MockBackupCodeRepository)
	logger := zerolog.Nop()
	handler := queries.NewGet2FAStatusHandler(mockTOTPRepo, mockBackupRepo, &logger)

	userID := identity.NewUserID()
	now := time.Now().UTC()
	verifiedAt := now.Add(-7 * 24 * time.Hour)

	// Create enabled TOTP secret
	totpSecret := identity.ReconstructTOTPSecret(
		[]byte("encrypted-secret"),
		"goimg",
		"user@example.com",
		true,
		verifiedAt,
	)

	// Mock expectations - user has 10 backup codes
	mockTOTPRepo.On("FindByUserID", mock.Anything, userID).
		Return(&totpSecret, nil)
	mockBackupRepo.On("CountUnused", mock.Anything, userID).
		Return(10, nil)

	query := queries.Get2FAStatusQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.False(t, result.SetupPending)
	assert.Equal(t, 10, result.BackupCodesRemaining)
	assert.NotNil(t, result.EnabledAt)
	assert.Equal(t, verifiedAt, *result.EnabledAt)

	mockTOTPRepo.AssertExpectations(t)
	mockBackupRepo.AssertExpectations(t)
}
