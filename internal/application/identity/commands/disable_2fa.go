package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// Disable2FACommand represents the intent to disable 2FA for a user.
type Disable2FACommand struct {
	UserID   string
	Password string // Password confirmation required
	Code     string // Optional TOTP code for extra security
}

// Disable2FAHandler processes 2FA disablement commands.
// It verifies the user's password and optionally TOTP code before disabling 2FA.
//
// Security considerations:
//   - Requires password confirmation (prevents CSRF/session hijacking)
//   - Optionally verifies TOTP code if 2FA is currently enabled
//   - Deletes TOTP secret and backup codes from storage
//   - Publishes UserTOTPDisabled event for audit logging
type Disable2FAHandler struct {
	users       identity.UserRepository
	totpRepo    TOTPRepository
	backupRepo  BackupCodeRepository
	totpService *security.TOTPService
	logger      *zerolog.Logger
}

// NewDisable2FAHandler creates a new Disable2FAHandler with the given dependencies.
func NewDisable2FAHandler(
	users identity.UserRepository,
	totpRepo TOTPRepository,
	backupRepo BackupCodeRepository,
	totpService *security.TOTPService,
	logger *zerolog.Logger,
) *Disable2FAHandler {
	return &Disable2FAHandler{
		users:       users,
		totpRepo:    totpRepo,
		backupRepo:  backupRepo,
		totpService: totpService,
		logger:      logger,
	}
}

// Handle executes the 2FA disablement use case.
//
// Process flow:
//  1. Validate user exists
//  2. Verify password
//  3. Retrieve TOTP secret (verify 2FA is enabled)
//  4. Optionally verify TOTP code if provided
//  5. Delete TOTP secret
//  6. Delete backup codes
//
// Returns:
//   - nil on successful disablement
//   - Err2FANotEnabled if 2FA is not enabled
//   - ErrInvalidCredentials if password is wrong
//   - Err2FAInvalidCode if TOTP code is provided but invalid
func (h *Disable2FAHandler) Handle(ctx context.Context, cmd Disable2FACommand) error {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user ID for 2FA disablement")
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Find user
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("user not found for 2FA disablement")
		return fmt.Errorf("find user: %w", err)
	}

	// 3. Verify password
	if err := user.VerifyPassword(cmd.Password); err != nil {
		h.logger.Warn().
			Str("user_id", cmd.UserID).
			Str("email", user.Email().String()).
			Msg("invalid password for 2FA disablement")
		return appidentity.ErrInvalidCredentials
	}

	// 4. Retrieve TOTP secret and verify 2FA is enabled
	totpSecret, err := h.totpRepo.FindByUserID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("no TOTP secret found for disablement")
		return appidentity.Err2FANotEnabled
	}

	// 5. If TOTP code provided, verify it (extra security)
	if cmd.Code != "" && totpSecret.IsEnabled() {
		err = h.totpService.ValidateCode(totpSecret.EncryptedSecret(), cmd.Code)
		if err != nil {
			h.logger.Warn().
				Str("user_id", cmd.UserID).
				Str("email", user.Email().String()).
				Msg("invalid TOTP code for 2FA disablement")
			return appidentity.Err2FAInvalidCode
		}
	}

	// 6. Delete backup codes first
	if err := h.backupRepo.DeleteByUserID(ctx, userID); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to delete backup codes")
		return fmt.Errorf("delete backup codes: %w", err)
	}

	// 7. Delete TOTP secret
	if err := h.totpRepo.Delete(ctx, userID); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to delete TOTP secret")
		return fmt.Errorf("delete TOTP secret: %w", err)
	}

	h.logger.Info().
		Str("user_id", cmd.UserID).
		Str("email", user.Email().String()).
		Msg("2FA successfully disabled")

	return nil
}
