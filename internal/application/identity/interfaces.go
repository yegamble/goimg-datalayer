package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SessionStore interface {
	Create(ctx context.Context, session *Session) error

	Get(ctx context.Context, sessionID uuid.UUID) (*Session, error)

	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)

	Revoke(ctx context.Context, sessionID uuid.UUID) error

	RevokeAll(ctx context.Context, userID uuid.UUID) error

	Exists(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

type JWTService interface {
	GenerateAccessToken(claims *TokenClaims) (string, error)

	GenerateRefreshToken(claims *TokenClaims) (string, error)

	ValidateToken(token string) (*TokenClaims, error)

	ExtractTokenID(token string) (string, error)

	GetTokenExpiration(token string) (time.Time, error)
}

type RefreshTokenService interface {
	Generate() (string, error)

	Store(ctx context.Context, tokenHash string, metadata *RefreshTokenMetadata) error

	Validate(ctx context.Context, token string) (*RefreshTokenMetadata, error)

	MarkAsUsed(ctx context.Context, tokenHash string) error

	RevokeFamily(ctx context.Context, familyID string) error
}

type TokenBlacklist interface {
	Add(ctx context.Context, tokenID string, expiration time.Time) error

	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)

	Remove(ctx context.Context, tokenID string) error
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type TokenClaims struct {
	UserID        uuid.UUID
	Email         string
	Role          string
	SessionID     uuid.UUID
	TokenType     string
	EmailVerified bool
}

type RefreshTokenMetadata struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	FamilyID  string
	ParentID  string
	IsUsed    bool
	CreatedAt time.Time
	ExpiresAt time.Time
}
