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

const (
	// DefaultTokenExpiryMinutes is the default token expiration in minutes.
	DefaultTokenExpiryMinutes = 15
)

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
	// 1-2. Validate and check for replay
	metadata, err := h.validateAndCheckReplay(ctx, cmd)
	if err != nil {
		return nil, err
	}

	// 3. Verify session
	if err := h.verifySession(ctx, metadata); err != nil {
		return nil, err
	}

	// 4. Load and verify user
	user, err := h.loadAndVerifyUser(ctx, metadata)
	if err != nil {
		return nil, err
	}

	// 5. Detect anomalies (informational only)
	h.detectAnomalies(metadata, cmd)

	// 6-8. Rotate tokens
	newAccessToken, newRefreshToken, newMetadata, err := h.rotateTokens(ctx, user, metadata, cmd)
	if err != nil {
		return nil, err
	}

	// 9. Update session
	h.updateSession(ctx, user, metadata, newMetadata, cmd)

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("session_id", metadata.SessionID).
		Str("family_id", metadata.FamilyID).
		Str("ip_address", cmd.IPAddress).
		Msg("token refreshed successfully")

	// 10. Build response
	return h.buildTokenResponse(newAccessToken, newRefreshToken), nil
}

func (h *RefreshTokenHandler) validateAndCheckReplay(ctx context.Context, cmd RefreshTokenCommand) (*services.RefreshTokenMetadata, error) {
	metadata, err := h.refreshService.ValidateToken(ctx, cmd.RefreshToken)
	if err != nil {
		h.logger.Warn().Err(err).Str("ip_address", cmd.IPAddress).Msg("invalid refresh token")
		return nil, fmt.Errorf("%w: %w", appidentity.ErrInvalidToken, err)
	}

	if metadata.ExpiresAt.Before(time.Now().UTC()) {
		h.logger.Warn().Str("user_id", metadata.UserID).Str("session_id", metadata.SessionID).Msg("refresh token expired")
		return nil, appidentity.ErrTokenExpired
	}

	if metadata.Used {
		h.handleTokenReplay(ctx, metadata, cmd.IPAddress)
		return nil, appidentity.ErrTokenReplayDetected
	}

	return metadata, nil
}

func (h *RefreshTokenHandler) handleTokenReplay(ctx context.Context, metadata *services.RefreshTokenMetadata, ipAddress string) {
	h.logger.Error().
		Str("user_id", metadata.UserID).
		Str("session_id", metadata.SessionID).
		Str("family_id", metadata.FamilyID).
		Str("ip_address", ipAddress).
		Msg("SECURITY: refresh token replay detected - revoking entire family")

	_ = h.refreshService.RevokeFamily(ctx, metadata.FamilyID)
	_ = h.sessionStore.Revoke(ctx, metadata.SessionID)
}

func (h *RefreshTokenHandler) verifySession(ctx context.Context, metadata *services.RefreshTokenMetadata) error {
	sessionExists, err := h.sessionStore.Exists(ctx, metadata.SessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", metadata.SessionID).Msg("failed to check session existence")
		return fmt.Errorf("check session existence: %w", err)
	}

	if !sessionExists {
		h.logger.Warn().Str("user_id", metadata.UserID).Str("session_id", metadata.SessionID).Msg("refresh token used for non-existent session")
		return appidentity.ErrSessionNotFound
	}

	return nil
}

func (h *RefreshTokenHandler) loadAndVerifyUser(ctx context.Context, metadata *services.RefreshTokenMetadata) (*identity.User, error) {
	userID, err := identity.ParseUserID(metadata.UserID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", metadata.UserID).Msg("invalid user ID in refresh token metadata")
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		return h.handleUserLoadError(err, metadata)
	}

	if !user.CanLogin() {
		return nil, h.handleInactiveUser(ctx, user, metadata)
	}

	return user, nil
}

func (h *RefreshTokenHandler) handleUserLoadError(err error, metadata *services.RefreshTokenMetadata) (*identity.User, error) {
	if errors.Is(err, identity.ErrUserNotFound) {
		h.logger.Warn().Str("user_id", metadata.UserID).Msg("refresh token for non-existent user")
		return nil, appidentity.ErrInvalidToken
	}

	h.logger.Error().Err(err).Str("user_id", metadata.UserID).Msg("failed to load user during token refresh")
	return nil, fmt.Errorf("load user: %w", err)
}

func (h *RefreshTokenHandler) handleInactiveUser(ctx context.Context, user *identity.User, metadata *services.RefreshTokenMetadata) error {
	h.logger.Warn().
		Str("user_id", user.ID().String()).
		Str("status", user.Status().String()).
		Msg("refresh token used for account that cannot login")

	_ = h.sessionStore.Revoke(ctx, metadata.SessionID)

	switch user.Status() {
	case identity.StatusSuspended:
		return appidentity.ErrAccountSuspended
	case identity.StatusDeleted:
		return appidentity.ErrAccountDeleted
	case identity.StatusActive, identity.StatusPending:
		// CanLogin() should have handled these, but included for exhaustiveness
		return appidentity.ErrInvalidToken
	default:
		return appidentity.ErrInvalidToken
	}
}

func (h *RefreshTokenHandler) detectAnomalies(metadata *services.RefreshTokenMetadata, cmd RefreshTokenCommand) {
	if h.refreshService.DetectAnomalies(metadata, cmd.IPAddress, cmd.UserAgent) {
		h.logger.Warn().
			Str("user_id", metadata.UserID).
			Str("session_id", metadata.SessionID).
			Str("original_ip", metadata.IP).
			Str("current_ip", cmd.IPAddress).
			Str("original_ua", metadata.UserAgent).
			Str("current_ua", cmd.UserAgent).
			Msg("anomaly detected during token refresh (IP or UserAgent changed)")
	}
}

func (h *RefreshTokenHandler) rotateTokens(ctx context.Context, user *identity.User, metadata *services.RefreshTokenMetadata, cmd RefreshTokenCommand) (string, string, *services.RefreshTokenMetadata, error) {
	if err := h.refreshService.MarkAsUsed(ctx, cmd.RefreshToken); err != nil {
		h.logger.Error().Err(err).Str("user_id", metadata.UserID).Str("session_id", metadata.SessionID).Msg("failed to mark refresh token as used")
		return "", "", nil, fmt.Errorf("mark token as used: %w", err)
	}

	newAccessToken, err := h.jwtService.GenerateAccessToken(user.ID().String(), user.Email().String(), user.Role().String(), metadata.SessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID().String()).Msg("failed to generate new access token")
		return "", "", nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, newMetadata, err := h.refreshService.GenerateToken(ctx, user.ID().String(), metadata.SessionID, metadata.FamilyID, metadata.TokenHash, cmd.IPAddress, cmd.UserAgent)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID().String()).Msg("failed to generate new refresh token")
		return "", "", nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return newAccessToken, newRefreshToken, newMetadata, nil
}

func (h *RefreshTokenHandler) updateSession(ctx context.Context, user *identity.User, metadata *services.RefreshTokenMetadata, newMetadata *services.RefreshTokenMetadata, cmd RefreshTokenCommand) {
	session := services.Session{
		SessionID: metadata.SessionID,
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Role:      user.Role().String(),
		IP:        cmd.IPAddress,
		UserAgent: cmd.UserAgent,
		CreatedAt: metadata.IssuedAt,
		ExpiresAt: newMetadata.ExpiresAt,
	}

	if err := h.sessionStore.Create(ctx, session); err != nil {
		h.logger.Error().Err(err).Str("session_id", metadata.SessionID).Msg("failed to update session after token refresh")
	}
}

func (h *RefreshTokenHandler) buildTokenResponse(accessToken, refreshToken string) *dto.TokenPairDTO {
	expiresAt, err := h.jwtService.GetTokenExpiration(accessToken)
	if err != nil {
		expiresAt = time.Now().UTC().Add(DefaultTokenExpiryMinutes * time.Minute)
	}

	tokenPair := dto.NewTokenPairDTO(accessToken, refreshToken, expiresAt)
	return &tokenPair
}
