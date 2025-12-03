package services

import (
	"context"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// AuthService orchestrates authentication and authorization operations.
// It coordinates between the domain layer (User aggregate), infrastructure
// services (JWT, sessions, token blacklist), and application layer (DTOs).
//
// This service is responsible for:
//   - User registration and login workflows
//   - Token generation and refresh
//   - Session management
//   - Logout (single device and all devices)
//   - Password changes and email verification
type AuthService interface {
	// Register creates a new user account and returns an authentication response.
	// This includes creating the user, generating tokens, and creating a session.
	//
	// Returns:
	//   - AuthResponseDTO with user data and token pair on success
	//   - Error if email/username already exists or validation fails
	Register(ctx context.Context, req dto.CreateUserDTO) (dto.AuthResponseDTO, error)

	// Login authenticates a user and returns an authentication response.
	// The identifier can be either email or username.
	//
	// This method:
	//   - Validates credentials
	//   - Checks user status (suspended users cannot login)
	//   - Generates new token pair
	//   - Creates a new session
	//
	// Returns:
	//   - AuthResponseDTO with user data and token pair on success
	//   - Error if credentials are invalid or user cannot login
	Login(ctx context.Context, req dto.LoginDTO) (dto.AuthResponseDTO, error)

	// RefreshToken validates a refresh token and returns a new token pair.
	//
	// This method:
	//   - Validates the refresh token
	//   - Checks for replay attacks (token reuse)
	//   - Detects anomalies (IP/User-Agent changes)
	//   - Marks old token as used
	//   - Generates new token pair with rotation
	//   - Revokes entire token family if replay detected
	//
	// Returns:
	//   - TokenPairDTO with new access and refresh tokens on success
	//   - Error if token is invalid, expired, or replay attack detected
	RefreshToken(ctx context.Context, req dto.RefreshTokenDTO) (dto.TokenPairDTO, error)

	// Logout revokes the current session and blacklists the access token.
	//
	// This method:
	//   - Revokes the session by session ID
	//   - Adds the access token to the blacklist
	//   - Revokes the refresh token
	//
	// Returns:
	//   - Error if logout fails (should be idempotent)
	Logout(ctx context.Context, sessionID, accessToken, refreshToken string) error

	// LogoutAll revokes all sessions for a user and blacklists all tokens.
	//
	// This method:
	//   - Retrieves all user sessions
	//   - Revokes all sessions
	//   - Adds all access tokens to blacklist
	//   - Revokes entire refresh token family
	//
	// Use cases:
	//   - User-initiated "logout from all devices"
	//   - Security incident (stolen credentials)
	//   - Password change
	//
	// Returns:
	//   - Error if operation fails
	LogoutAll(ctx context.Context, userID string) error

	// ChangePassword updates a user's password after validating the current password.
	//
	// This method:
	//   - Validates current password
	//   - Hashes new password
	//   - Updates user password via domain method
	//   - Optionally revokes all sessions (force re-login)
	//
	// Returns:
	//   - Error if current password is invalid or update fails
	ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordDTO, revokeAllSessions bool) error

	// GetActiveSessions retrieves all active sessions for a user.
	//
	// Returns:
	//   - List of SessionDTO with session metadata
	//   - Error if retrieval fails
	GetActiveSessions(ctx context.Context, userID string, currentSessionID string) ([]dto.SessionDTO, error)

	// RevokeSession revokes a specific session for a user.
	//
	// This is useful for "logout from another device" functionality.
	//
	// Returns:
	//   - Error if session does not exist or revocation fails
	RevokeSession(ctx context.Context, userID, sessionID string) error

	// ValidateToken validates an access token and returns its claims.
	//
	// This method:
	//   - Checks token blacklist
	//   - Validates token signature and expiry
	//   - Verifies session still exists
	//
	// Returns:
	//   - Token claims (user ID, email, role, session ID) on success
	//   - Error if token is invalid, expired, or blacklisted
	ValidateToken(ctx context.Context, accessToken string) (*TokenClaims, error)
}

// TokenClaims represents the claims extracted from a validated JWT token.
// This is returned by ValidateToken and used by HTTP middleware.
type TokenClaims struct {
	UserID    string
	Email     string
	Role      string
	SessionID string
	ExpiresAt time.Time
}

// SessionStore defines the interface for managing user sessions in Redis.
// This interface abstracts the Redis implementation and is injected into AuthService.
type SessionStore interface {
	// Create stores a new session with the given TTL.
	Create(ctx context.Context, session Session) error

	// Get retrieves a session by its ID.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Exists checks if a session exists.
	Exists(ctx context.Context, sessionID string) (bool, error)

	// Revoke removes a single session.
	Revoke(ctx context.Context, sessionID string) error

	// RevokeAll removes all sessions for a user.
	RevokeAll(ctx context.Context, userID string) error

	// GetUserSessions retrieves all sessions for a user.
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
}

// Session represents session metadata stored in Redis.
// This mirrors the infrastructure layer's Session type but defined in application layer.
type Session struct {
	SessionID string
	UserID    string
	Email     string
	Role      string
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// JWTService defines the interface for JWT token operations.
// This abstracts the JWT implementation from the application layer.
type JWTService interface {
	// GenerateAccessToken creates a short-lived access token.
	GenerateAccessToken(userID, email, role, sessionID string) (string, error)

	// GenerateRefreshToken creates a long-lived refresh token.
	GenerateRefreshToken(userID, email, role, sessionID string) (string, error)

	// ValidateToken validates a token and returns its claims.
	ValidateToken(tokenString string) (*JWTClaims, error)

	// ExtractTokenID extracts the JWT ID (jti) without full validation.
	ExtractTokenID(tokenString string) (string, error)

	// GetTokenExpiration extracts expiration time without full validation.
	GetTokenExpiration(tokenString string) (time.Time, error)
}

// JWTClaims represents JWT claims returned by the JWT service.
// This mirrors the infrastructure JWT claims structure.
type JWTClaims struct {
	UserID    string
	Email     string
	Role      string
	SessionID string
	TokenType string
	JTI       string // JWT ID for blacklisting
	ExpiresAt time.Time
}

// RefreshTokenService defines the interface for refresh token operations.
// This handles cryptographically secure refresh tokens with rotation.
type RefreshTokenService interface {
	// GenerateToken creates a new refresh token with metadata.
	GenerateToken(ctx context.Context, userID, sessionID, familyID, parentHash, ip, userAgent string) (string, *RefreshTokenMetadata, error)

	// ValidateToken validates a refresh token and returns its metadata.
	ValidateToken(ctx context.Context, token string) (*RefreshTokenMetadata, error)

	// MarkAsUsed marks a token as used for replay detection.
	MarkAsUsed(ctx context.Context, token string) error

	// RevokeToken revokes a single refresh token.
	RevokeToken(ctx context.Context, token string) error

	// RevokeFamily revokes all tokens in a family (replay attack response).
	RevokeFamily(ctx context.Context, familyID string) error

	// DetectAnomalies checks for suspicious behavior (IP/UA changes).
	DetectAnomalies(metadata *RefreshTokenMetadata, currentIP, currentUserAgent string) bool
}

// RefreshTokenMetadata represents refresh token metadata.
// This mirrors the infrastructure layer's metadata structure.
type RefreshTokenMetadata struct {
	TokenHash  string
	UserID     string
	SessionID  string
	FamilyID   string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	IP         string
	UserAgent  string
	ParentHash string
	Used       bool
}

// TokenBlacklist defines the interface for blacklisting revoked tokens.
// This is used for immediate logout and token revocation.
type TokenBlacklist interface {
	// Add adds a token to the blacklist with TTL until expiry.
	Add(ctx context.Context, tokenID string, expiresAt time.Time) error

	// IsBlacklisted checks if a token is blacklisted.
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)

	// Remove removes a token from the blacklist (rare).
	Remove(ctx context.Context, tokenID string) error
}

// UserRepository defines the repository interface for User aggregates.
// This is defined in the domain layer but referenced here for clarity.
type UserRepository interface {
	identity.UserRepository
}
