package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// Verify2FALoginCommand represents the intent to verify 2FA during login.
// This is used after initial password authentication when a user has 2FA enabled.
// The command takes the non-elevated JWT token and TOTP code, verifies the code,
// and returns an elevated token with TwoFAVerified flag set to true.
type Verify2FALoginCommand struct {
	UserID        string // From authenticated JWT context
	SessionID     string // From authenticated JWT context
	Code          string // 6-digit TOTP code or 8-character backup code
	UseBackupCode bool   // Whether the code is a backup code
	IPAddress     string // For session tracking
	UserAgent     string // For session tracking
}

// isCommand implements the Command interface marker.
func (Verify2FALoginCommand) isCommand() {}

// Verify2FALoginHandler processes 2FA login verification commands.
// It validates the TOTP/backup code and generates an elevated access token.
//
// Security considerations:
//   - Requires valid non-elevated JWT in context (set by middleware)
//   - Validates TOTP code or backup code via domain methods
//   - Issues elevated token with TwoFAVerified=true
//   - Applies timing defense to prevent enumeration
type Verify2FALoginHandler struct {
	users       identity.UserRepository
	totpRepo    TOTPRepository
	jwtService  services.JWTService
	totpService *security.TOTPService
	logger      *zerolog.Logger
}

// NewVerify2FALoginHandler creates a new handler with the given dependencies.
func NewVerify2FALoginHandler(
	users identity.UserRepository,
	totpRepo TOTPRepository,
	jwtService services.JWTService,
	totpService *security.TOTPService,
	logger *zerolog.Logger,
) *Verify2FALoginHandler {
	return &Verify2FALoginHandler{
		users:       users,
		totpRepo:    totpRepo,
		jwtService:  jwtService,
		totpService: totpService,
		logger:      logger,
	}
}

// Handle executes the 2FA login verification use case.
//
// Process flow:
//  1. Parse user ID from command (extracted from JWT by handler)
//  2. Load user aggregate
//  3. Verify that user has 2FA enabled
//  4. Validate TOTP code or backup code
//  5. Generate elevated access token with TwoFAVerified=true
//  6. Return token pair (elevated access token + existing refresh token)
//
// Returns:
//   - TokenPairDTO with elevated access token on success
//   - ErrInvalidCredentials if code is wrong
//   - Err2FANotEnabled if user doesn't have 2FA
func (h *Verify2FALoginHandler) Handle(ctx context.Context, cmd Verify2FALoginCommand) (*dto.TokenPairDTO, error) {
	targetDelay := appidentity.CalculateRandomAuthDelay()
	startTime := time.Now()

	defer func() {
		actualDuration := time.Since(startTime)
		appidentity.ApplyAuthDelay(targetDelay, actualDuration)
		h.logger.Debug().Msg("2FA login verification timing defense applied")
	}()

	// 1. Parse user ID from command
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("invalid user ID in 2FA login verification")
		return nil, appidentity.ErrInvalidCredentials
	}

	// 2. Load user aggregate
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Msg("failed to load user for 2FA verification")
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 3. Verify user has 2FA enabled
	if !user.IsTOTPEnabled() {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Msg("2FA login verification attempted for user without 2FA enabled")
		return nil, appidentity.Err2FANotEnabled
	}

	// 4. Validate TOTP code or backup code
	var verifyErr error
	if cmd.UseBackupCode {
		// UseBackupCode validates and marks the backup code as used
		verifyErr = user.UseBackupCode(cmd.Code)
		if verifyErr == nil {
			// Save user to persist backup code usage
			if saveErr := h.users.Save(ctx, user); saveErr != nil {
				h.logger.Error().
					Err(saveErr).
					Str("user_id", user.ID().String()).
					Msg("failed to save user after backup code usage")
				return nil, fmt.Errorf("save user after backup code: %w", saveErr)
			}
		}
	} else {
		// Load TOTP secret from repository
		totpSecret, loadErr := h.totpRepo.FindByUserID(ctx, userID)
		if loadErr != nil {
			h.logger.Error().
				Err(loadErr).
				Str("user_id", userID.String()).
				Msg("failed to load TOTP secret for verification")
			return nil, fmt.Errorf("load TOTP secret: %w", loadErr)
		}
		// Validate TOTP code against encrypted secret
		verifyErr = h.totpService.ValidateCode(totpSecret.EncryptedSecret(), cmd.Code)
	}

	if verifyErr != nil {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Bool("used_backup_code", cmd.UseBackupCode).
			Msg("invalid 2FA code during login verification")
		return nil, appidentity.Err2FAInvalidCode
	}

	// 5. Generate elevated access token with TwoFAVerified=true
	elevatedAccessToken, err := h.jwtService.GenerateElevatedAccessToken(
		user.ID().String(),
		user.Email().String(),
		user.Role().String(),
		cmd.SessionID,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate elevated access token")
		return nil, fmt.Errorf("generate elevated access token: %w", err)
	}

	// 6. Get token expiration for response
	expiresAt, err := h.jwtService.GetTokenExpiration(elevatedAccessToken)
	if err != nil {
		// Non-critical - use default
		expiresAt = time.Now().UTC().Add(defaultTokenExpirationMinutes * time.Minute)
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("email", user.Email().String()).
		Str("session_id", cmd.SessionID).
		Str("ip_address", cmd.IPAddress).
		Bool("used_backup_code", cmd.UseBackupCode).
		Msg("2FA verification successful, session elevated")

	// 7. Build response with elevated token
	// Note: We return the elevated access token but keep the same refresh token
	// The refresh token is already associated with the session
	tokens := dto.TokenPairDTO{
		AccessToken:  elevatedAccessToken,
		RefreshToken: "", // Refresh token not changed, client keeps existing one
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
		ExpiresAt:    expiresAt,
	}

	return &tokens, nil
}
