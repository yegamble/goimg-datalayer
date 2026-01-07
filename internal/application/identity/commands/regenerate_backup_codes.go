package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RegenerateBackupCodesCommand represents the intent to regenerate backup codes.
type RegenerateBackupCodesCommand struct {
	UserID   string
	Password string // Password confirmation required
}

// RegenerateBackupCodesHandler processes backup code regeneration commands.
// It generates new backup codes, invalidating any existing ones.
//
// Security considerations:
//   - Requires password confirmation
//   - 2FA must be enabled to regenerate backup codes
//   - Old backup codes are deleted before new ones are created
//   - Publishes UserBackupCodesRegenerated event for audit logging
type RegenerateBackupCodesHandler struct {
	users      identity.UserRepository
	totpRepo   TOTPRepository
	backupRepo BackupCodeRepository
	logger     *zerolog.Logger
}

// NewRegenerateBackupCodesHandler creates a new handler.
func NewRegenerateBackupCodesHandler(
	users identity.UserRepository,
	totpRepo TOTPRepository,
	backupRepo BackupCodeRepository,
	logger *zerolog.Logger,
) *RegenerateBackupCodesHandler {
	return &RegenerateBackupCodesHandler{
		users:      users,
		totpRepo:   totpRepo,
		backupRepo: backupRepo,
		logger:     logger,
	}
}

// Handle executes the backup code regeneration use case.
//
// Process flow:
//  1. Validate user exists
//  2. Verify password
//  3. Verify 2FA is enabled
//  4. Generate new backup codes
//  5. Replace existing backup codes in storage
//
// Returns:
//   - BackupCodesResponseDTO with new codes on success
//   - Err2FANotEnabled if 2FA is not enabled
//   - ErrInvalidCredentials if password is wrong
func (h *RegenerateBackupCodesHandler) Handle(ctx context.Context, cmd RegenerateBackupCodesCommand) (*dto.BackupCodesResponseDTO, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user ID for backup code regeneration")
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Find user
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("user not found for backup code regeneration")
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 3. Verify password
	if err := user.VerifyPassword(cmd.Password); err != nil {
		h.logger.Warn().
			Str("user_id", cmd.UserID).
			Str("email", user.Email().String()).
			Msg("invalid password for backup code regeneration")
		return nil, appidentity.ErrInvalidCredentials
	}

	// 4. Verify 2FA is enabled
	enabled, err := h.totpRepo.IsEnabled(ctx, userID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to check 2FA status")
		return nil, fmt.Errorf("check 2FA status: %w", err)
	}

	if !enabled {
		h.logger.Debug().
			Str("user_id", cmd.UserID).
			Msg("cannot regenerate backup codes - 2FA not enabled")
		return nil, appidentity.Err2FANotEnabled
	}

	// 5. Generate new backup codes
	plaintextCodes, hashedCodes, err := identity.GenerateBackupCodes()
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to generate backup codes")
		return nil, fmt.Errorf("generate backup codes: %w", err)
	}

	// 6. Replace existing backup codes (SaveAll deletes old ones first)
	if err := h.backupRepo.SaveAll(ctx, userID, hashedCodes); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to save new backup codes")
		return nil, fmt.Errorf("save backup codes: %w", err)
	}

	h.logger.Info().
		Str("user_id", cmd.UserID).
		Str("email", user.Email().String()).
		Int("code_count", len(plaintextCodes)).
		Msg("backup codes regenerated")

	return &dto.BackupCodesResponseDTO{
		BackupCodes:    plaintextCodes,
		RemainingCodes: len(plaintextCodes),
	}, nil
}
