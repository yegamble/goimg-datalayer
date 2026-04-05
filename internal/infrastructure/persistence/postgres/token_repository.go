package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	// #nosec G101 // False positive: SQL query, not a hardcoded credential

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlCreateEmailVerificationToken = `
		INSERT INTO email_verification_tokens (user_id, expires_at)
		VALUES ($1, $2)
		RETURNING token
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlFindValidEmailVerificationToken = `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlMarkEmailVerificationTokenUsed = `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE token = $1
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlCreatePasswordResetToken = `
		INSERT INTO password_reset_tokens (user_id, expires_at)
		VALUES ($1, $2)
		RETURNING token
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlFindValidPasswordResetToken = `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlMarkPasswordResetTokenUsed = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token = $1
	`

	// #nosec G101 // False positive: SQL query, not a hardcoded credential
	sqlInvalidateAllPasswordResetTokens = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE user_id = $1
		  AND used_at IS NULL
	`
)

type passwordResetTokenRow struct {
	ID        string       `db:"id"`
	UserID    string       `db:"user_id"`
	Token     string       `db:"token"`
	ExpiresAt time.Time    `db:"expires_at"`
	UsedAt    sql.NullTime `db:"used_at"`
	CreatedAt time.Time    `db:"created_at"`
}

type TokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) CreatePasswordResetToken(
	ctx context.Context,
	userID identity.UserID,
	expiresAt time.Time,
) (string, error) {
	var token string
	err := r.db.QueryRowContext(ctx, sqlCreatePasswordResetToken, userID.String(), expiresAt).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("create password reset token: %w", err)
	}
	return token, nil
}

func (r *TokenRepository) FindValidPasswordResetToken(
	ctx context.Context,
	token string,
) (*identity.PasswordResetToken, error) {
	var row passwordResetTokenRow
	err := r.db.GetContext(ctx, &row, sqlFindValidPasswordResetToken, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrTokenNotFound
		}
		return nil, fmt.Errorf("find valid password reset token: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	prt := &identity.PasswordResetToken{
		ID:        row.ID,
		UserID:    userID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
	if row.UsedAt.Valid {
		prt.UsedAt = &row.UsedAt.Time
	}

	return prt, nil
}

func (r *TokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, sqlMarkPasswordResetTokenUsed, token)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *TokenRepository) InvalidateAllPasswordResetTokens(
	ctx context.Context,
	userID identity.UserID,
) error {
	_, err := r.db.ExecContext(ctx, sqlInvalidateAllPasswordResetTokens, userID.String())
	if err != nil {
		return fmt.Errorf("invalidate password reset tokens: %w", err)
	}
	return nil
}

func (r *TokenRepository) CreateEmailVerificationToken(
	ctx context.Context,
	userID identity.UserID,
	expiresAt time.Time,
) (string, error) {
	var token string
	err := r.db.QueryRowContext(ctx, sqlCreateEmailVerificationToken, userID.String(), expiresAt).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("create email verification token: %w", err)
	}
	return token, nil
}

func (r *TokenRepository) FindValidEmailVerificationToken(
	ctx context.Context,
	token string,
) (*identity.EmailVerificationToken, error) {
	var row passwordResetTokenRow
	err := r.db.GetContext(ctx, &row, sqlFindValidEmailVerificationToken, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrTokenNotFound
		}
		return nil, fmt.Errorf("find valid email verification token: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	evt := &identity.EmailVerificationToken{
		ID:        row.ID,
		UserID:    userID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
	if row.UsedAt.Valid {
		evt.UsedAt = &row.UsedAt.Time
	}

	return evt, nil
}

func (r *TokenRepository) MarkEmailVerificationTokenUsed(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, sqlMarkEmailVerificationTokenUsed, token)
	if err != nil {
		return fmt.Errorf("mark email verification token used: %w", err)
	}
	return nil
}
