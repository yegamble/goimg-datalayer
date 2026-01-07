package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for backup code operations.
const (
	sqlInsertBackupCode = `
		INSERT INTO user_backup_codes (id, user_id, code_hash, used, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	sqlSelectBackupCodesByUserID = `
		SELECT id, user_id, code_hash, used, used_at, created_at
		FROM user_backup_codes
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	sqlSelectUnusedBackupCodesByUserID = `
		SELECT id, user_id, code_hash, used, used_at, created_at
		FROM user_backup_codes
		WHERE user_id = $1 AND used = false
		ORDER BY created_at ASC
	`

	sqlMarkBackupCodeUsed = `
		UPDATE user_backup_codes
		SET used = true,
		    used_at = $2
		WHERE id = $1
	`

	sqlDeleteBackupCodesByUserID = `
		DELETE FROM user_backup_codes
		WHERE user_id = $1
	`

	sqlCountUnusedBackupCodes = `
		SELECT COUNT(*)
		FROM user_backup_codes
		WHERE user_id = $1 AND used = false
	`
)

// backupCodeRow represents a backup code row in the database.
type backupCodeRow struct {
	ID        string       `db:"id"`
	UserID    string       `db:"user_id"`
	CodeHash  string       `db:"code_hash"`
	Used      bool         `db:"used"`
	UsedAt    sql.NullTime `db:"used_at"`
	CreatedAt time.Time    `db:"created_at"`
}

// BackupCodeRepository handles persistence of backup codes.
type BackupCodeRepository struct {
	db *sqlx.DB
}

// NewBackupCodeRepository creates a new backup code repository.
func NewBackupCodeRepository(db *sqlx.DB) *BackupCodeRepository {
	return &BackupCodeRepository{db: db}
}

// SaveAll persists a set of backup codes for a user.
// This replaces any existing backup codes for the user.
func (r *BackupCodeRepository) SaveAll(ctx context.Context, userID identity.UserID, codes []identity.BackupCode) error {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete existing backup codes
	_, err = tx.ExecContext(ctx, sqlDeleteBackupCodesByUserID, userID.String())
	if err != nil {
		return fmt.Errorf("failed to delete existing backup codes: %w", err)
	}

	// Insert new backup codes
	now := time.Now().UTC()
	for _, code := range codes {
		id := uuid.New().String()
		var usedAt sql.NullTime
		if code.IsUsed() && !code.UsedAt().IsZero() {
			usedAt = sql.NullTime{Time: code.UsedAt(), Valid: true}
		}

		_, err = tx.ExecContext(
			ctx,
			sqlInsertBackupCode,
			id,
			userID.String(),
			code.HashedCode(),
			code.IsUsed(),
			usedAt,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert backup code: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByUserID retrieves all backup codes for a user.
func (r *BackupCodeRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error) {
	var rows []backupCodeRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectBackupCodesByUserID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find backup codes: %w", err)
	}

	codes := make([]identity.BackupCode, len(rows))
	for i, row := range rows {
		var usedAt time.Time
		if row.UsedAt.Valid {
			usedAt = row.UsedAt.Time
		}
		codes[i] = identity.ReconstructBackupCode(row.CodeHash, row.Used, usedAt)
	}

	return codes, nil
}

// FindUnusedByUserID retrieves all unused backup codes for a user.
func (r *BackupCodeRepository) FindUnusedByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error) {
	var rows []backupCodeRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectUnusedBackupCodesByUserID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find unused backup codes: %w", err)
	}

	codes := make([]identity.BackupCode, len(rows))
	for i, row := range rows {
		codes[i] = identity.ReconstructBackupCode(row.CodeHash, false, time.Time{})
	}

	return codes, nil
}

// MarkUsed marks a specific backup code as used.
// The codeID is the database ID, not the plaintext code.
func (r *BackupCodeRepository) MarkUsed(ctx context.Context, codeID string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, sqlMarkBackupCodeUsed, codeID, now)
	if err != nil {
		return fmt.Errorf("failed to mark backup code as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrBackupCodeInvalid
	}

	return nil
}

// MarkUsedByHash finds a backup code by its hash and marks it as used.
// Returns error if the code is not found or already used.
func (r *BackupCodeRepository) MarkUsedByHash(ctx context.Context, userID identity.UserID, codeHash string) error {
	var row backupCodeRow
	err := r.db.GetContext(ctx, &row, `
		SELECT id, user_id, code_hash, used, used_at, created_at
		FROM user_backup_codes
		WHERE user_id = $1 AND code_hash = $2 AND used = false
	`, userID.String(), codeHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return identity.ErrBackupCodeInvalid
		}
		return fmt.Errorf("failed to find backup code: %w", err)
	}

	return r.MarkUsed(ctx, row.ID)
}

// DeleteByUserID removes all backup codes for a user.
func (r *BackupCodeRepository) DeleteByUserID(ctx context.Context, userID identity.UserID) error {
	_, err := r.db.ExecContext(ctx, sqlDeleteBackupCodesByUserID, userID.String())
	if err != nil {
		return fmt.Errorf("failed to delete backup codes: %w", err)
	}
	return nil
}

// CountUnused returns the number of unused backup codes for a user.
func (r *BackupCodeRepository) CountUnused(ctx context.Context, userID identity.UserID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountUnusedBackupCodes, userID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count unused backup codes: %w", err)
	}
	return count, nil
}
