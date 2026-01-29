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

// SQL queries for TOTP operations.
// These reference the user_totp_secrets table - not hardcoded credentials.
const (
	sqlInsertTOTPSecret = `
		INSERT INTO user_totp_secrets (user_id, encrypted_secret, issuer, account_name, enabled, verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	` // #nosec G101

	sqlUpdateTOTPSecret = `
		UPDATE user_totp_secrets
		SET encrypted_secret = $2,
		    issuer = $3,
		    account_name = $4,
		    enabled = $5,
		    verified_at = $6,
		    updated_at = $7
		WHERE user_id = $1
	` // #nosec G101

	sqlSelectTOTPSecretByUserID = `
		SELECT user_id, encrypted_secret, issuer, account_name, enabled, verified_at, created_at, updated_at
		FROM user_totp_secrets
		WHERE user_id = $1
	` // #nosec G101

	sqlDeleteTOTPSecret = `
		DELETE FROM user_totp_secrets
		WHERE user_id = $1
	` // #nosec G101

	sqlCheckTOTPExists = `
		SELECT EXISTS(SELECT 1 FROM user_totp_secrets WHERE user_id = $1)
	` // #nosec G101
)

// totpSecretRow represents a TOTP secret row in the database.
type totpSecretRow struct {
	UserID          string       `db:"user_id"`
	EncryptedSecret []byte       `db:"encrypted_secret"`
	Issuer          string       `db:"issuer"`
	AccountName     string       `db:"account_name"`
	Enabled         bool         `db:"enabled"`
	VerifiedAt      sql.NullTime `db:"verified_at"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
}

// TOTPRepository handles persistence of TOTP secrets.
type TOTPRepository struct {
	db *sqlx.DB
}

// NewTOTPRepository creates a new TOTP repository.
func NewTOTPRepository(db *sqlx.DB) *TOTPRepository {
	return &TOTPRepository{db: db}
}

// Save persists a TOTP secret for a user.
// If a secret already exists, it is updated; otherwise, it is created.
func (r *TOTPRepository) Save(ctx context.Context, userID identity.UserID, secret identity.TOTPSecret) error {
	// Check if secret exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckTOTPExists, userID.String())
	if err != nil {
		return fmt.Errorf("failed to check TOTP existence: %w", err)
	}

	now := time.Now().UTC()

	var verifiedAt sql.NullTime
	if !secret.VerifiedAt().IsZero() {
		verifiedAt = sql.NullTime{Time: secret.VerifiedAt(), Valid: true}
	}

	if exists {
		_, err = r.db.ExecContext(
			ctx,
			sqlUpdateTOTPSecret,
			userID.String(),
			secret.EncryptedSecret(),
			secret.Issuer(),
			secret.AccountName(),
			secret.IsEnabled(),
			verifiedAt,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to update TOTP secret: %w", err)
		}
		return nil
	}

	_, err = r.db.ExecContext(
		ctx,
		sqlInsertTOTPSecret,
		userID.String(),
		secret.EncryptedSecret(),
		secret.Issuer(),
		secret.AccountName(),
		secret.IsEnabled(),
		verifiedAt,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert TOTP secret: %w", err)
	}

	return nil
}

// FindByUserID retrieves the TOTP secret for a user.
func (r *TOTPRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*identity.TOTPSecret, error) {
	var row totpSecretRow
	err := r.db.GetContext(ctx, &row, sqlSelectTOTPSecretByUserID, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrTOTPNotEnabled
		}
		return nil, fmt.Errorf("failed to find TOTP secret: %w", err)
	}

	var verifiedAt time.Time
	if row.VerifiedAt.Valid {
		verifiedAt = row.VerifiedAt.Time
	}

	secret := identity.ReconstructTOTPSecret(
		row.EncryptedSecret,
		row.Issuer,
		row.AccountName,
		row.Enabled,
		verifiedAt,
	)

	return &secret, nil
}

// Delete removes the TOTP secret for a user.
func (r *TOTPRepository) Delete(ctx context.Context, userID identity.UserID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteTOTPSecret, userID.String())
	if err != nil {
		return fmt.Errorf("failed to delete TOTP secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrTOTPNotEnabled
	}

	return nil
}

// Exists checks if a TOTP secret exists for a user.
func (r *TOTPRepository) Exists(ctx context.Context, userID identity.UserID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckTOTPExists, userID.String())
	if err != nil {
		return false, fmt.Errorf("failed to check TOTP existence: %w", err)
	}
	return exists, nil
}

// IsEnabled checks if 2FA is enabled for a user.
func (r *TOTPRepository) IsEnabled(ctx context.Context, userID identity.UserID) (bool, error) {
	var enabled bool
	err := r.db.GetContext(ctx, &enabled, `
		SELECT enabled
		FROM user_totp_secrets
		WHERE user_id = $1 AND enabled = true AND verified_at IS NOT NULL
	`, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check 2FA status: %w", err)
	}
	return enabled, nil
}
