package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// LoginCommand represents the intent to authenticate a user.
// The identifier can be either an email address or username.
// IPAddress and UserAgent are captured for security auditing and anomaly detection.
type LoginCommand struct {
	Identifier string // Email or username
	Password   string
	IPAddress  string
	UserAgent  string
}

// LoginHandler processes login commands.
// It orchestrates the authentication workflow: credential validation,
// status checks, token generation, and session creation.
//
// Security considerations:
//   - Uses constant-time comparison for password verification
//   - Returns generic error to prevent user enumeration
//   - Enforces account status checks (suspended users cannot login)
//   - Creates session with IP and UserAgent for anomaly detection
type LoginHandler struct {
	users          identity.UserRepository
	jwtService     services.JWTService
	refreshService services.RefreshTokenService
	sessionStore   services.SessionStore
	logger         *zerolog.Logger
}

// NewLoginHandler creates a new LoginHandler with the given dependencies.
func NewLoginHandler(
	users identity.UserRepository,
	jwtService services.JWTService,
	refreshService services.RefreshTokenService,
	sessionStore services.SessionStore,
	logger *zerolog.Logger,
) *LoginHandler {
	return &LoginHandler{
		users:          users,
		jwtService:     jwtService,
		refreshService: refreshService,
		sessionStore:   sessionStore,
		logger:         logger,
	}
}

// Handle executes the login use case.
//
// Process flow:
//  1. Parse identifier (email or username) and find user
//  2. Verify password using constant-time comparison
//  3. Check user status (suspended/deleted users cannot login)
//  4. Generate session ID
//  5. Generate JWT access token (15 min TTL)
//  6. Generate refresh token with family ID (7 day TTL)
//  7. Create session in Redis
//  8. Return AuthResponseDTO with tokens and user data
//
// Security notes:
//   - Returns generic ErrInvalidCredentials to prevent user enumeration
//   - Never reveals whether email/username exists or password is wrong
//   - Uses constant-time password comparison to prevent timing attacks
//
// Returns:
//   - AuthResponseDTO with user data and token pair on success
//   - ErrInvalidCredentials if credentials are wrong or user not found
//   - ErrAccountSuspended if account is suspended
//   - ErrAccountDeleted if account is deleted
func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*dto.AuthResponseDTO, error) {
	// 1. Parse identifier and find user
	// Try email first, then username
	user, err := h.findUserByIdentifier(ctx, cmd.Identifier)
	if err != nil {
		// Don't reveal if user exists - return generic error
		h.logger.Debug().
			Err(err).
			Str("identifier", cmd.Identifier).
			Str("ip_address", cmd.IPAddress).
			Msg("login attempt with invalid identifier")
		return nil, appidentity.ErrInvalidCredentials
	}

	// 2. Verify password (constant-time comparison via domain method)
	if err := user.VerifyPassword(cmd.Password); err != nil {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Str("email", user.Email().String()).
			Str("ip_address", cmd.IPAddress).
			Msg("login attempt with invalid password")
		return nil, appidentity.ErrInvalidCredentials
	}

	// 3. Check user status
	if !user.CanLogin() {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Str("email", user.Email().String()).
			Str("status", user.Status().String()).
			Str("ip_address", cmd.IPAddress).
			Msg("login attempt for account that cannot login")

		// Return specific error based on status
		switch user.Status() {
		case identity.StatusSuspended:
			return nil, appidentity.ErrAccountSuspended
		case identity.StatusDeleted:
			return nil, appidentity.ErrAccountDeleted
		default:
			return nil, appidentity.ErrInvalidCredentials
		}
	}

	// 4. Generate session ID
	sessionID := uuid.New().String()

	// 5. Generate JWT access token (15 min TTL)
	accessToken, err := h.jwtService.GenerateAccessToken(
		user.ID().String(),
		user.Email().String(),
		user.Role().String(),
		sessionID,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate access token")
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 6. Generate refresh token with family ID (7 day TTL)
	familyID := uuid.New().String()
	refreshToken, metadata, err := h.refreshService.GenerateToken(
		ctx,
		user.ID().String(),
		sessionID,
		familyID,
		"", // No parent hash for first token in family
		cmd.IPAddress,
		cmd.UserAgent,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate refresh token")
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 7. Create session in Redis
	session := services.Session{
		SessionID: sessionID,
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Role:      user.Role().String(),
		IP:        cmd.IPAddress,
		UserAgent: cmd.UserAgent,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: metadata.ExpiresAt,
	}
	if err := h.sessionStore.Create(ctx, session); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("session_id", sessionID).
			Msg("failed to create session")
		return nil, fmt.Errorf("create session: %w", err)
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("email", user.Email().String()).
		Str("session_id", sessionID).
		Str("ip_address", cmd.IPAddress).
		Str("user_agent", cmd.UserAgent).
		Msg("user logged in successfully")

	// 8. Build response with tokens and user data
	expiresAt, err := h.jwtService.GetTokenExpiration(accessToken)
	if err != nil {
		// Non-critical - use default 15 min
		expiresAt = time.Now().UTC().Add(15 * time.Minute)
	}

	tokens := dto.NewTokenPairDTO(accessToken, refreshToken, expiresAt)
	authResponse := dto.NewAuthResponseDTO(user, tokens)

	return &authResponse, nil
}

// findUserByIdentifier attempts to find a user by email or username.
// Returns ErrInvalidCredentials if user not found (prevents enumeration).
func (h *LoginHandler) findUserByIdentifier(ctx context.Context, identifier string) (*identity.User, error) {
	// Try parsing as email first
	if email, err := identity.NewEmail(identifier); err == nil {
		user, err := h.users.FindByEmail(ctx, email)
		if err != nil {
			if errors.Is(err, identity.ErrUserNotFound) {
				return nil, appidentity.ErrInvalidCredentials
			}
			return nil, fmt.Errorf("find user by email: %w", err)
		}
		return user, nil
	}

	// Try parsing as username
	if username, err := identity.NewUsername(identifier); err == nil {
		user, err := h.users.FindByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, identity.ErrUserNotFound) {
				return nil, appidentity.ErrInvalidCredentials
			}
			return nil, fmt.Errorf("find user by username: %w", err)
		}
		return user, nil
	}

	// Neither email nor username format is valid
	return nil, appidentity.ErrInvalidCredentials
}
