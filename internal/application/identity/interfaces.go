package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SessionStore interface for Redis session operations.
// Sessions track active user authentication sessions and enable multi-device logout.
type SessionStore interface {
	// Create stores a new session in Redis.
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by its ID.
	// Returns an error if the session does not exist or has expired.
	Get(ctx context.Context, sessionID uuid.UUID) (*Session, error)

	// GetUserSessions retrieves all active sessions for a user.
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)

	// Revoke invalidates a specific session.
	Revoke(ctx context.Context, sessionID uuid.UUID) error

	// RevokeAll invalidates all sessions for a user (e.g., password change).
	RevokeAll(ctx context.Context, userID uuid.UUID) error

	// Exists checks if a session exists without retrieving it.
	Exists(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

// JWTService interface for JWT token operations.
// Handles generation and validation of access and refresh JWTs.
type JWTService interface {
	// GenerateAccessToken creates a new access JWT with the given claims.
	GenerateAccessToken(claims *TokenClaims) (string, error)

	// GenerateRefreshToken creates a new refresh JWT with the given claims.
	GenerateRefreshToken(claims *TokenClaims) (string, error)

	// ValidateToken validates a JWT and returns the claims.
	// Returns an error if the token is invalid, expired, or malformed.
	ValidateToken(token string) (*TokenClaims, error)

	// ExtractTokenID extracts the JWT ID (jti) from a token without full validation.
	// This is used for blacklist checks before full validation.
	ExtractTokenID(token string) (string, error)

	// GetTokenExpiration extracts the expiration time from a token.
	GetTokenExpiration(token string) (time.Time, error)
}

// RefreshTokenService interface for refresh token rotation and family management.
// Implements refresh token rotation to detect and prevent token replay attacks.
type RefreshTokenService interface {
	// Generate creates a new cryptographically random refresh token.
	Generate() (string, error)

	// Store persists refresh token metadata in Redis.
	Store(ctx context.Context, tokenHash string, metadata *RefreshTokenMetadata) error

	// Validate checks if a refresh token is valid and not reused.
	// Returns metadata if valid, or an error if invalid/reused.
	Validate(ctx context.Context, token string) (*RefreshTokenMetadata, error)

	// MarkAsUsed marks a refresh token as consumed (for rotation).
	// If a used token is presented again, it indicates replay attack.
	MarkAsUsed(ctx context.Context, tokenHash string) error

	// RevokeFamily invalidates all tokens in a refresh token family.
	// Used when replay attack is detected.
	RevokeFamily(ctx context.Context, familyID string) error
}

// TokenBlacklist interface for revoking access tokens before expiration.
// Since JWTs are stateless, we need a blacklist to revoke them (e.g., logout).
type TokenBlacklist interface {
	// Add adds a token ID to the blacklist until its expiration time.
	Add(ctx context.Context, tokenID string, expiration time.Time) error

	// IsBlacklisted checks if a token ID is blacklisted.
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)

	// Remove removes a token ID from the blacklist (e.g., after expiration).
	Remove(ctx context.Context, tokenID string) error
}

// Session represents an active user authentication session.
// Sessions are stored in Redis with TTL matching the refresh token expiration.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// TokenClaims represents the claims embedded in a JWT.
// These claims are used for both access and refresh tokens.
type TokenClaims struct {
	UserID    uuid.UUID
	Email     string
	Role      string
	SessionID uuid.UUID
	TokenType string // "access" or "refresh"
}

// RefreshTokenMetadata contains metadata for a refresh token stored in Redis.
// This enables refresh token rotation and family-based revocation.
type RefreshTokenMetadata struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	FamilyID  string // All rotated tokens share the same family ID
	ParentID  string // ID of the previous token in the rotation chain
	IsUsed    bool   // Whether this token has been consumed
	CreatedAt time.Time
	ExpiresAt time.Time
}
