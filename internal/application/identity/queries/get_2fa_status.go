package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// Get2FAStatusQuery represents the intent to retrieve a user's 2FA status.
type Get2FAStatusQuery struct {
	UserID string
}

// TOTPRepository defines the interface for TOTP secret queries.
type TOTPRepository interface {
	FindByUserID(ctx context.Context, userID identity.UserID) (*identity.TOTPSecret, error)
	IsEnabled(ctx context.Context, userID identity.UserID) (bool, error)
}

// BackupCodeRepository defines the interface for backup code queries.
type BackupCodeRepository interface {
	CountUnused(ctx context.Context, userID identity.UserID) (int, error)
}

// Get2FAStatusHandler processes 2FA status queries.
type Get2FAStatusHandler struct {
	totpRepo   TOTPRepository
	backupRepo BackupCodeRepository
	logger     *zerolog.Logger
}

// NewGet2FAStatusHandler creates a new Get2FAStatusHandler.
func NewGet2FAStatusHandler(
	totpRepo TOTPRepository,
	backupRepo BackupCodeRepository,
	logger *zerolog.Logger,
) *Get2FAStatusHandler {
	return &Get2FAStatusHandler{
		totpRepo:   totpRepo,
		backupRepo: backupRepo,
		logger:     logger,
	}
}

// Handle executes the 2FA status query.
//
// Returns:
//   - TwoFactorStatusDTO with current 2FA status
//   - Error if user ID is invalid or database query fails
func (h *Get2FAStatusHandler) Handle(ctx context.Context, query Get2FAStatusQuery) (*dto.TwoFactorStatusDTO, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(query.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", query.UserID).
			Msg("invalid user ID for 2FA status query")
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Try to find TOTP secret
	totpSecret, err := h.totpRepo.FindByUserID(ctx, userID)
	if err != nil {
		// No TOTP secret means 2FA is not set up
		return &dto.TwoFactorStatusDTO{
			Enabled:              false,
			SetupPending:         false,
			BackupCodesRemaining: 0,
			EnabledAt:            nil,
		}, nil
	}

	// 3. Check if enabled
	enabled := totpSecret.IsEnabled()
	setupPending := totpSecret.IsSetupPending()

	// 4. Get backup code count if 2FA is enabled
	var backupCodesRemaining int
	if enabled {
		backupCodesRemaining, err = h.backupRepo.CountUnused(ctx, userID)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", query.UserID).
				Msg("failed to count unused backup codes")
			// Don't fail the whole query, just report 0
			backupCodesRemaining = 0
		}
	}

	// 5. Get enabled timestamp
	var enabledAt *interface{}
	if enabled {
		t := totpSecret.VerifiedAt()
		result := &dto.TwoFactorStatusDTO{
			Enabled:              enabled,
			SetupPending:         setupPending,
			BackupCodesRemaining: backupCodesRemaining,
			EnabledAt:            &t,
		}
		return result, nil
	}

	return &dto.TwoFactorStatusDTO{
		Enabled:              enabled,
		SetupPending:         setupPending,
		BackupCodesRemaining: backupCodesRemaining,
		EnabledAt:            nil,
	}, nil
}
