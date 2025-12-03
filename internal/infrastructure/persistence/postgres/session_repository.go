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

// Session cleanup constants.
const (
	revokedSessionRetentionDays = 30
)

// SQL queries for session operations.
const (
	sqlInsertSession = `
		INSERT INTO sessions (id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	sqlSelectSessionByID = `
		SELECT id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions
		WHERE id = $1
	`

	sqlSelectSessionsByUserID = `
		SELECT id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	sqlRevokeSession = `
		UPDATE sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	sqlDeleteExpiredSessions = `
		DELETE FROM sessions
		WHERE expires_at < $1 OR revoked_at < $2
	`
)

// Session represents a user authentication session.
type Session struct {
	ID               uuid.UUID
	UserID           identity.UserID
	RefreshTokenHash string
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	RevokedAt        *time.Time
}

// sessionRow represents a session row in the database.
type sessionRow struct {
	ID               string         `db:"id"`
	UserID           string         `db:"user_id"`
	RefreshTokenHash string         `db:"refresh_token_hash"`
	IPAddress        sql.NullString `db:"ip_address"`
	UserAgent        sql.NullString `db:"user_agent"`
	ExpiresAt        time.Time      `db:"expires_at"`
	CreatedAt        time.Time      `db:"created_at"`
	RevokedAt        sql.NullTime   `db:"revoked_at"`
}

// SessionRepository manages user authentication sessions in PostgreSQL.
type SessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository creates a new SessionRepository with the given database connection.
func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session in the database.
func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertSession,
		session.ID.String(),
		session.UserID.String(),
		session.RefreshTokenHash,
		nullString(session.IPAddress),
		nullString(session.UserAgent),
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetByID retrieves a session by its ID.
func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	var row sessionRow
	if err := r.db.GetContext(ctx, &row, sqlSelectSessionByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session by id: %w", err)
	}

	session, err := rowToSession(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to session: %w", err)
	}

	return session, nil
}

// GetByUserID retrieves all active sessions for a user.
func (r *SessionRepository) GetByUserID(ctx context.Context, userID identity.UserID) ([]*Session, error) {
	var rows []sessionRow
	if err := r.db.SelectContext(ctx, &rows, sqlSelectSessionsByUserID, userID.String()); err != nil {
		return nil, fmt.Errorf("failed to get sessions by user id: %w", err)
	}

	sessions := make([]*Session, 0, len(rows))
	for _, row := range rows {
		session, err := rowToSession(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Revoke revokes a session by setting the revoked_at timestamp.
func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, sqlRevokeSession, id.String(), now)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found or already revoked")
	}

	return nil
}

// DeleteExpired deletes expired and old revoked sessions.
// This should be called periodically to clean up stale sessions.
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	// Delete sessions that expired or were revoked more than revokedSessionRetentionDays ago
	revokedCutoff := now.Add(-revokedSessionRetentionDays * 24 * time.Hour)

	result, err := r.db.ExecContext(ctx, sqlDeleteExpiredSessions, now, revokedCutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// rowToSession converts a database row to a Session struct.
func rowToSession(row sessionRow) (*Session, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid session id: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	session := &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: row.RefreshTokenHash,
		IPAddress:        nullStringValue(row.IPAddress),
		UserAgent:        nullStringValue(row.UserAgent),
		ExpiresAt:        row.ExpiresAt,
		CreatedAt:        row.CreatedAt,
	}

	if row.RevokedAt.Valid {
		session.RevokedAt = &row.RevokedAt.Time
	}

	return session, nil
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullStringValue extracts the string value from sql.NullString.
func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
