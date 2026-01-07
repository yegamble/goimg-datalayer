package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// Verify2FACommand represents the intent to verify and enable 2FA for a user.
type Verify2FACommand struct {
	UserID string
	Code   string // 6-digit TOTP code from authenticator app
}

// Verify2FAHandler processes 2FA verification commands.
// It validates the TOTP code against the pending secret and enables 2FA.
//
// Security considerations:
//   - Code validation uses constant-time comparison
//   - Allows 1 time-step skew (±30 seconds)
//   - Must have pending 2FA setup (from Setup2FA)
//   - Publishes UserTOTPEnabled event for audit logging
type Verify2FAHandler struct {
	users       identity.UserRepository
	totpRepo    TOTPRepository
	totpService *security.TOTPService
	logger      *zerolog.Logger
}

// NewVerify2FAHandler creates a new Verify2FAHandler with the given dependencies.
func NewVerify2FAHandler(
	users identity.UserRepository,
	totpRepo TOTPRepository,
	totpService *security.TOTPService,
	logger *zerolog.Logger,
) *Verify2FAHandler {
	return &Verify2FAHandler{
		users:       users,
		totpRepo:    totpRepo,
		totpService: totpService,
		logger:      logger,
	}
}

// Handle executes the 2FA verification use case.
//
// Process flow:
//  1. Validate user exists
//  2. Retrieve pending TOTP secret
//  3. Validate TOTP code against the secret
//  4. Enable the TOTP secret (mark as verified)
//  5. Save the updated secret
//
// Returns:
//   - nil on successful verification and enablement
//   - Err2FANotEnabled if no pending 2FA setup exists
//   - Err2FAAlreadyEnabled if 2FA is already active
//   - Err2FAInvalidCode if the code doesn't match
func (h *Verify2FAHandler) Handle(ctx context.Context, cmd Verify2FACommand) error {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user ID for 2FA verification")
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Verify user exists
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("user not found for 2FA verification")
		return fmt.Errorf("find user: %w", err)
	}

	// 3. Retrieve TOTP secret
	totpSecret, err := h.totpRepo.FindByUserID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("no TOTP secret found for verification")
		return appidentity.Err2FANotEnabled
	}

	// 4. Check if already enabled
	if totpSecret.IsEnabled() {
		h.logger.Debug().
			Str("user_id", cmd.UserID).
			Msg("2FA already enabled - verification not needed")
		return appidentity.Err2FAAlreadyEnabled
	}

	// 5. Validate TOTP code
	err = h.totpService.ValidateCode(totpSecret.EncryptedSecret(), cmd.Code)
	if err != nil {
		h.logger.Warn().
			Str("user_id", cmd.UserID).
			Str("email", user.Email().String()).
			Msg("invalid TOTP code during 2FA verification")
		return appidentity.Err2FAInvalidCode
	}

	// 6. Enable the TOTP secret
	totpSecret.Enable()

	// 7. Save updated secret
	if err := h.totpRepo.Save(ctx, userID, *totpSecret); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("failed to save enabled TOTP secret")
		return fmt.Errorf("save TOTP secret: %w", err)
	}

	h.logger.Info().
		Str("user_id", cmd.UserID).
		Str("email", user.Email().String()).
		Msg("2FA successfully enabled")

	return nil
}
