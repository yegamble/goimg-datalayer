package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// Setup2FACommand represents the intent to initiate 2FA setup for a user.
type Setup2FACommand struct {
	UserID string
}

// Setup2FAHandler processes 2FA setup commands.
// It generates a TOTP secret, backup codes, and returns the provisioning data.
//
// Security considerations:
//   - Only authenticated users can setup 2FA
//   - Users with existing 2FA must disable it first
//   - Secrets are encrypted before storage
//   - Backup codes are hashed like passwords
//   - Setup is not complete until verification (Verify2FA)
type Setup2FAHandler struct {
	users       identity.UserRepository
	totpRepo    TOTPRepository
	backupRepo  BackupCodeRepository
	totpService *security.TOTPService
	logger      *zerolog.Logger
}

// TOTPRepository defines the interface for TOTP secret persistence.
type TOTPRepository interface {
	Save(ctx context.Context, userID identity.UserID, secret identity.TOTPSecret) error
	FindByUserID(ctx context.Context, userID identity.UserID) (*identity.TOTPSecret, error)
	Delete(ctx context.Context, userID identity.UserID) error
	IsEnabled(ctx context.Context, userID identity.UserID) (bool, error)
}

// BackupCodeRepository defines the interface for backup code persistence.
type BackupCodeRepository interface {
	SaveAll(ctx context.Context, userID identity.UserID, codes []identity.BackupCode) error
	FindByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error)
	FindUnusedByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error)
	CountUnused(ctx context.Context, userID identity.UserID) (int, error)
	DeleteByUserID(ctx context.Context, userID identity.UserID) error
}

// NewSetup2FAHandler creates a new Setup2FAHandler with the given dependencies.
func NewSetup2FAHandler(
	users identity.UserRepository,
	totpRepo TOTPRepository,
	backupRepo BackupCodeRepository,
	totpService *security.TOTPService,
	logger *zerolog.Logger,
) *Setup2FAHandler {
	return &Setup2FAHandler{
		users:       users,
		totpRepo:    totpRepo,
		backupRepo:  backupRepo,
		totpService: totpService,
		logger:      logger,
	}
}

// Handle executes the 2FA setup use case.
//
// Process flow:
//  1. Validate user exists and is active
//  2. Check if 2FA is already enabled (must disable first)
//  3. Generate TOTP secret using TOTPService
//  4. Generate backup codes
//  5. Store encrypted secret and hashed backup codes
//  6. Return setup data (secret, QR URI, backup codes)
//
// Note: 2FA is not active until Verify2FA is called with a valid code.
//
// Returns:
//   - Setup2FAResponseDTO with secret, QR URI, and backup codes on success
//   - Err2FAAlreadyEnabled if user already has 2FA enabled
//   - ErrUserNotFound if user doesn't exist
func (h *Setup2FAHandler) Handle(ctx context.Context, cmd Setup2FACommand) (*dto.Setup2FAResponseDTO, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user ID for 2FA setup")
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Find user and verify they exist
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("user not found for 2FA setup")
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 3. Check if 2FA is already enabled
	enabled, err := h.totpRepo.IsEnabled(ctx, userID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to check 2FA status")
		return nil, fmt.Errorf("check 2FA status: %w", err)
	}

	if enabled {
		h.logger.Debug().
			Str("user_id", cmd.UserID).
			Msg("2FA already enabled")
		return nil, appidentity.Err2FAAlreadyEnabled
	}

	// 4. Generate TOTP secret
	accountName := user.Email().String()
	setupResult, err := h.totpService.GenerateSecret(accountName)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to generate TOTP secret")
		return nil, fmt.Errorf("generate TOTP secret: %w", err)
	}

	// 5. Create domain TOTP secret (setup pending, not yet verified)
	totpSecret, err := identity.NewTOTPSecret(setupResult.EncryptedSecret, accountName)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to create TOTP secret value object")
		return nil, fmt.Errorf("create TOTP secret: %w", err)
	}

	// 6. Generate backup codes
	plaintextCodes, hashedCodes, err := identity.GenerateBackupCodes()
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to generate backup codes")
		return nil, fmt.Errorf("generate backup codes: %w", err)
	}

	// 7. Store TOTP secret (not enabled yet, pending verification)
	if err := h.totpRepo.Save(ctx, userID, totpSecret); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to save TOTP secret")
		return nil, fmt.Errorf("save TOTP secret: %w", err)
	}

	// 8. Store backup codes
	if err := h.backupRepo.SaveAll(ctx, userID, hashedCodes); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to save backup codes")
		// Rollback TOTP secret
		_ = h.totpRepo.Delete(ctx, userID)
		return nil, fmt.Errorf("save backup codes: %w", err)
	}

	h.logger.Info().
		Str("user_id", cmd.UserID).
		Str("email", user.Email().String()).
		Msg("2FA setup initiated - awaiting verification")

	// 9. Return setup data
	return &dto.Setup2FAResponseDTO{
		Secret:          setupResult.Secret,
		ProvisioningURI: setupResult.ProvisioningURI,
		Issuer:          setupResult.Issuer,
		AccountName:     setupResult.AccountName,
		BackupCodes:     plaintextCodes,
	}, nil
}
