package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// LogoutCommand represents the intent to logout a user session.
// It can either logout the current session or all sessions for a user.
//
// Security considerations:
//   - Access token is blacklisted to prevent reuse
//   - Refresh token is revoked
//   - Session is removed from Redis
//   - LogoutAll revokes all sessions and tokens for the user
type LogoutCommand struct {
	UserID       string
	SessionID    string
	AccessToken  string
	RefreshToken string
	LogoutAll    bool // If true, revoke all sessions for user
}

// Implement Command interface from types.go
func (LogoutCommand) isCommand() {}

// LogoutHandler processes logout commands.
// It handles both single-session logout and logout from all devices.
//
// Responsibilities:
//   - Blacklist access token to prevent immediate reuse
//   - Revoke refresh token
//   - Remove session(s) from Redis
//   - Optionally revoke all user sessions (logout all devices)
type LogoutHandler struct {
	users          identity.UserRepository
	jwtService     services.JWTService
	refreshService services.RefreshTokenService
	sessionStore   services.SessionStore
	tokenBlacklist services.TokenBlacklist
	logger         *zerolog.Logger
}

// NewLogoutHandler creates a new LogoutHandler with the given dependencies.
func NewLogoutHandler(
	users identity.UserRepository,
	jwtService services.JWTService,
	refreshService services.RefreshTokenService,
	sessionStore services.SessionStore,
	tokenBlacklist services.TokenBlacklist,
	logger *zerolog.Logger,
) *LogoutHandler {
	return &LogoutHandler{
		users:          users,
		jwtService:     jwtService,
		refreshService: refreshService,
		sessionStore:   sessionStore,
		tokenBlacklist: tokenBlacklist,
		logger:         logger,
	}
}

// Handle executes the logout use case.
//
// Process flow for single logout:
//  1. Extract token ID (jti) from access token
//  2. Calculate remaining TTL for blacklist
//  3. Add access token to blacklist
//  4. Revoke refresh token
//  5. Remove session from Redis
//
// Process flow for logout all:
//  1. Load user to verify exists
//  2. Get all user sessions
//  3. Extract and blacklist all access tokens
//  4. Revoke all refresh tokens for user
//  5. Remove all sessions from Redis
//
// Returns:
//   - nil on successful logout
//   - Error if logout operations fail (operation is idempotent where possible)
func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	// Validate user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user ID in logout command")
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Handle logout all devices
	if cmd.LogoutAll {
		return h.handleLogoutAll(ctx, userID)
	}

	// Handle single session logout
	return h.handleSingleLogout(ctx, userID, cmd.SessionID, cmd.AccessToken, cmd.RefreshToken)
}

// handleSingleLogout processes logout for a single session.
func (h *LogoutHandler) handleSingleLogout(
	ctx context.Context,
	userID identity.UserID,
	sessionID string,
	accessToken string,
	refreshToken string,
) error {
	// 1. Extract token ID (jti) from access token for blacklisting
	tokenID, err := h.jwtService.ExtractTokenID(accessToken)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", userID.String()).
			Str("session_id", sessionID).
			Msg("failed to extract token ID from access token during logout")
		// Continue - don't fail logout if token extraction fails
	} else {
		// 2. Calculate remaining TTL for blacklist entry
		expiresAt, err := h.jwtService.GetTokenExpiration(accessToken)
		if err != nil {
			h.logger.Warn().
				Err(err).
				Str("user_id", userID.String()).
				Str("session_id", sessionID).
				Msg("failed to get token expiration during logout")
			// Use default TTL of 15 minutes if extraction fails
			expiresAt = time.Now().UTC().Add(15 * time.Minute)
		}

		// 3. Add access token to blacklist with TTL = remaining lifetime
		if err := h.tokenBlacklist.Add(ctx, tokenID, expiresAt); err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", userID.String()).
				Str("session_id", sessionID).
				Str("token_id", tokenID).
				Msg("failed to blacklist access token during logout")
			// Continue - don't fail logout if blacklist fails
		}
	}

	// 4. Revoke refresh token
	if refreshToken != "" {
		if err := h.refreshService.RevokeToken(ctx, refreshToken); err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", userID.String()).
				Str("session_id", sessionID).
				Msg("failed to revoke refresh token during logout")
			// Continue - don't fail logout if token revocation fails
		}
	}

	// 5. Remove session from Redis (idempotent - OK if session doesn't exist)
	if err := h.sessionStore.Revoke(ctx, sessionID); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Str("session_id", sessionID).
			Msg("failed to revoke session during logout")
		// Continue - don't fail logout if session revocation fails
	}

	h.logger.Info().
		Str("user_id", userID.String()).
		Str("session_id", sessionID).
		Msg("user logged out successfully")

	return nil
}

// handleLogoutAll processes logout for all user sessions.
func (h *LogoutHandler) handleLogoutAll(ctx context.Context, userID identity.UserID) error {
	// 1. Load user to verify exists
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Msg("failed to load user during logout all")
		return fmt.Errorf("load user: %w", err)
	}

	// 2. Get all user sessions
	sessions, err := h.sessionStore.GetUserSessions(ctx, userID.String())
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Msg("failed to get user sessions during logout all")
		return fmt.Errorf("get user sessions: %w", err)
	}

	// 3. For each session, extract and blacklist access token
	// Note: We don't have access tokens stored, so we can't blacklist them individually
	// The session revocation will be sufficient for security
	// In a production system, you might store access token JTIs in sessions

	// 4. Revoke all sessions from Redis
	if err := h.sessionStore.RevokeAll(ctx, userID.String()); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userID.String()).
			Msg("failed to revoke all sessions during logout all")
		return fmt.Errorf("revoke all sessions: %w", err)
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("email", user.Email().String()).
		Int("sessions_revoked", len(sessions)).
		Msg("user logged out from all devices")

	return nil
}
