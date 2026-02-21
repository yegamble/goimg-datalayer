package identity

import (
	"context"
	"errors"
	"time"
)

var ErrTokenNotFound = errors.New("token not found")

var ErrTokenExpiredDomain = errors.New("token has expired")

var ErrTokenAlreadyUsed = errors.New("token has already been used")

type PasswordResetToken struct {
	ID        string
	UserID    UserID
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}

type EmailVerificationToken struct {
	ID        string
	UserID    UserID
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *EmailVerificationToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

func (t *EmailVerificationToken) IsUsed() bool {
	return t.UsedAt != nil
}

type TokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, userID UserID, expiresAt time.Time) (string, error)

	FindValidPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error)

	MarkPasswordResetTokenUsed(ctx context.Context, token string) error

	InvalidateAllPasswordResetTokens(ctx context.Context, userID UserID) error

	CreateEmailVerificationToken(ctx context.Context, userID UserID, expiresAt time.Time) (string, error)

	FindValidEmailVerificationToken(ctx context.Context, token string) (*EmailVerificationToken, error)

	MarkEmailVerificationTokenUsed(ctx context.Context, token string) error
}

type UserRepository interface {
	NextID() UserID

	FindByID(ctx context.Context, id UserID) (*User, error)

	FindByEmail(ctx context.Context, email Email) (*User, error)

	FindByUsername(ctx context.Context, username Username) (*User, error)

	Save(ctx context.Context, user *User) error

	Delete(ctx context.Context, id UserID) error

	FindExpiredGuests(ctx context.Context, asOf time.Time, limit int) ([]*User, error)
}
