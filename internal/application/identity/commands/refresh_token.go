package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RefreshTokenCommand represents the intent to refresh an access token using a refresh token.
// This implements token rotation for enhanced security.
//
// Security considerations:
//   - IPAddress and UserAgent are compared with original token metadata for anomaly detection
//   - Token reuse (replay attack) triggers revocation of entire token family
//   - Each refresh generates a new token pair, invalidating the old one
type RefreshTokenCommand struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

// Implement Command interface from types.go
func (RefreshTokenCommand) isCommand() {}

// RefreshTokenHandler processes token refresh commands.
// It implements automatic token rotation with replay attack detection.
//
// Security features:
//   - Validates refresh token exists and is not expired
//   - Checks for replay attacks (token reuse)
//   - Detects anomalies (IP/UserAgent changes)
//   - Marks old token as used (one-time use only)
//   - Generates new token pair (rotation)
//   - Revokes entire token family if replay detected
type RefreshTokenHandler struct {
	users          identity.UserRepository
	jwtService     services.JWTService
	refreshService services.RefreshTokenService
	sessionStore   services.SessionStore
	logger         *zerolog.Logger
}

// NewRefreshTokenHandler creates a new RefreshTokenHandler with the given dependencies.
func NewRefreshTokenHandler(
	users identity.UserRepository,
	jwtService services.JWTService,
	refreshService services.RefreshTokenService,
	sessionStore services.SessionStore,
	logger *zerolog.Logger,
) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		users:          users,
		jwtService:     jwtService,
		refreshService: refreshService,
		sessionStore:   sessionStore,
		logger:         logger,
	}
}

// Handle executes the token refresh use case.
//
// Process flow:
//  1. Validate refresh token and retrieve metadata
//  2. Check if token has already been used (replay attack detection)
//  3. Verify session still exists
//  4. Load user and verify account status
//  5. Detect anomalies (IP/UserAgent changes)
//  6. Mark current token as used
//  7. Generate new access token
//  8. Generate new refresh token (rotation with same family)
//  9. Update session metadata in Redis
//
// 10. Return new TokenPairDTO
//
// Security notes:
//   - If token reuse is detected, entire token family is revoked
//   - Anomalies (IP/UA changes) are logged but not blocked (could be VPN/mobile network)
//   - One-time use enforced: each token can only be used once
//
// Returns:
//   - TokenPairDTO with new access and refresh tokens on success
//   - ErrInvalidToken if token is malformed or expired
//   - ErrTokenReplayDetected if token has already been used
//   - ErrSessionNotFound if session no longer exists
//   - ErrAccountSuspended if user account is suspended
func (h *RefreshTokenHandler) Handle(ctx context.Context, cmd RefreshTokenCommand) (*dto.TokenPairDTO, error) {
	// 1. Validate refresh token and retrieve metadata
	metadata, err := h.refreshService.ValidateToken(ctx, cmd.RefreshToken)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("ip_address", cmd.IPAddress).
			Msg("invalid refresh token")
		return nil, fmt.Errorf("%w: %v", appidentity.ErrInvalidToken, err)
	}

	// Check if token is expired (should be caught by ValidateToken, but double-check)
	if metadata.ExpiresAt.Before(time.Now().UTC()) {
		h.logger.Warn().
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Msg("refresh token expired")
		return nil, appidentity.ErrTokenExpired
	}

	// 2. Check for replay attack (token already used)
	if metadata.Used {
		h.logger.Error().
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Str("family_id", metadata.FamilyID).
			Str("ip_address", cmd.IPAddress).
			Msg("SECURITY: refresh token replay detected - revoking entire family")

		// Revoke entire token family as a security measure
		if err := h.refreshService.RevokeFamily(ctx, metadata.FamilyID); err != nil {
			h.logger.Error().
				Err(err).
				Str("family_id", metadata.FamilyID).
				Msg("failed to revoke token family after replay detection")
		}

		// Revoke session
		if err := h.sessionStore.Revoke(ctx, metadata.SessionID); err != nil {
			h.logger.Error().
				Err(err).
				Str("session_id", metadata.SessionID).
				Msg("failed to revoke session after replay detection")
		}

		return nil, appidentity.ErrTokenReplayDetected
	}

	// 3. Verify session still exists
	sessionExists, err := h.sessionStore.Exists(ctx, metadata.SessionID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("session_id", metadata.SessionID).
			Msg("failed to check session existence")
		return nil, fmt.Errorf("check session existence: %w", err)
	}
	if !sessionExists {
		h.logger.Warn().
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Msg("refresh token used for non-existent session")
		return nil, appidentity.ErrSessionNotFound
	}

	// 4. Load user and verify account status
	userID, err := identity.ParseUserID(metadata.UserID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", metadata.UserID).
			Msg("invalid user ID in refresh token metadata")
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			h.logger.Warn().
				Str("user_id", metadata.UserID).
				Msg("refresh token for non-existent user")
			return nil, appidentity.ErrInvalidToken
		}
		h.logger.Error().
			Err(err).
			Str("user_id", metadata.UserID).
			Msg("failed to load user during token refresh")
		return nil, fmt.Errorf("load user: %w", err)
	}

	// Verify user can still login
	if !user.CanLogin() {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Str("status", user.Status().String()).
			Msg("refresh token used for account that cannot login")

		// Revoke session
		_ = h.sessionStore.Revoke(ctx, metadata.SessionID)

		switch user.Status() {
		case identity.StatusSuspended:
			return nil, appidentity.ErrAccountSuspended
		case identity.StatusDeleted:
			return nil, appidentity.ErrAccountDeleted
		default:
			return nil, appidentity.ErrInvalidToken
		}
	}

	// 5. Detect anomalies (IP/UserAgent changes)
	// Note: We log anomalies but don't block the request as users may legitimately
	// change networks (VPN, mobile data, WiFi) or devices
	if h.refreshService.DetectAnomalies(metadata, cmd.IPAddress, cmd.UserAgent) {
		h.logger.Warn().
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Str("original_ip", metadata.IP).
			Str("current_ip", cmd.IPAddress).
			Str("original_ua", metadata.UserAgent).
			Str("current_ua", cmd.UserAgent).
			Msg("anomaly detected during token refresh (IP or UserAgent changed)")
		// Continue processing - this is informational only
	}

	// 6. Mark current token as used (one-time use enforcement)
	if err := h.refreshService.MarkAsUsed(ctx, cmd.RefreshToken); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Msg("failed to mark refresh token as used")
		return nil, fmt.Errorf("mark token as used: %w", err)
	}

	// 7. Generate new access token
	newAccessToken, err := h.jwtService.GenerateAccessToken(
		user.ID().String(),
		user.Email().String(),
		user.Role().String(),
		metadata.SessionID,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate new access token")
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 8. Generate new refresh token (rotation with same family)
	// Parent hash is the current token's hash for lineage tracking
	newRefreshToken, newMetadata, err := h.refreshService.GenerateToken(
		ctx,
		user.ID().String(),
		metadata.SessionID,
		metadata.FamilyID,  // Same family for rotation
		metadata.TokenHash, // Parent is current token
		cmd.IPAddress,
		cmd.UserAgent,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate new refresh token")
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 9. Update session metadata in Redis (update expiry and last access)
	session := services.Session{
		SessionID: metadata.SessionID,
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Role:      user.Role().String(),
		IP:        cmd.IPAddress,
		UserAgent: cmd.UserAgent,
		CreatedAt: metadata.IssuedAt, // Keep original creation time
		ExpiresAt: newMetadata.ExpiresAt,
	}
	if err := h.sessionStore.Create(ctx, session); err != nil {
		h.logger.Error().
			Err(err).
			Str("session_id", metadata.SessionID).
			Msg("failed to update session after token refresh")
		// Non-critical - continue
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("session_id", metadata.SessionID).
		Str("family_id", metadata.FamilyID).
		Str("ip_address", cmd.IPAddress).
		Msg("token refreshed successfully")

	// 10. Build response with new token pair
	expiresAt, err := h.jwtService.GetTokenExpiration(newAccessToken)
	if err != nil {
		// Non-critical - use default 15 min
		expiresAt = time.Now().UTC().Add(15 * time.Minute)
	}

	tokenPair := dto.NewTokenPairDTO(newAccessToken, newRefreshToken, expiresAt)
	return &tokenPair, nil
}
