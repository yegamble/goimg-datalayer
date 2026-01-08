package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// SQL queries for ban operations.
const (
	sqlInsertBan = `
		INSERT INTO user_bans (id, user_id, banned_by, reason, expires_at, created_at, revoked_at, revoked_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	sqlUpdateBan = `
		UPDATE user_bans
		SET revoked_at = $2,
		    revoked_by = $3
		WHERE id = $1
	`

	sqlSelectBanByID = `
		SELECT id, user_id, banned_by, reason, expires_at, created_at, revoked_at, revoked_by
		FROM user_bans
		WHERE id = $1
	`

	sqlSelectBanByUserID = `
		SELECT id, user_id, banned_by, reason, expires_at, created_at, revoked_at, revoked_by
		FROM user_bans
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	sqlSelectActiveBans = `
		SELECT id, user_id, banned_by, reason, expires_at, created_at, revoked_at, revoked_by
		FROM user_bans
		WHERE revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
	`

	sqlSelectExpiredBans = `
		SELECT id, user_id, banned_by, reason, expires_at, created_at, revoked_at, revoked_by
		FROM user_bans
		WHERE revoked_at IS NULL
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()
		ORDER BY expires_at DESC
	`

	sqlCheckUserBanned = `
		SELECT EXISTS(
			SELECT 1
			FROM user_bans
			WHERE user_id = $1
			  AND revoked_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
		)
	`
)

// banRow represents a ban row in the database.
type banRow struct {
	ID        string         `db:"id"`
	UserID    string         `db:"user_id"`
	BannedBy  string         `db:"banned_by"`
	Reason    string         `db:"reason"`
	ExpiresAt sql.NullTime   `db:"expires_at"`
	CreatedAt time.Time      `db:"created_at"`
	RevokedAt sql.NullTime   `db:"revoked_at"`
	RevokedBy sql.NullString `db:"revoked_by"`
}

// BanRepository implements the moderation.BanRepository interface for PostgreSQL.
type BanRepository struct {
	db *sqlx.DB
}

// NewBanRepository creates a new BanRepository with the given database connection.
func NewBanRepository(db *sqlx.DB) *BanRepository {
	return &BanRepository{db: db}
}

// NextID generates the next available BanID.
func (r *BanRepository) NextID() moderation.BanID {
	return moderation.NewBanID()
}

// FindByID retrieves a ban by its unique ID.
func (r *BanRepository) FindByID(ctx context.Context, id moderation.BanID) (*moderation.Ban, error) {
	var row banRow
	if err := r.db.GetContext(ctx, &row, sqlSelectBanByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, moderation.ErrBanNotFound
		}
		return nil, fmt.Errorf("failed to find ban by id: %w", err)
	}

	ban, err := rowToBan(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to ban: %w", err)
	}

	return ban, nil
}

// FindByUserID retrieves the most recent ban for a specific user.
func (r *BanRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*moderation.Ban, error) {
	var row banRow
	if err := r.db.GetContext(ctx, &row, sqlSelectBanByUserID, userID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, moderation.ErrBanNotFound
		}
		return nil, fmt.Errorf("failed to find ban by user id: %w", err)
	}

	ban, err := rowToBan(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to ban: %w", err)
	}

	return ban, nil
}

// FindActiveBans retrieves all currently active bans.
func (r *BanRepository) FindActiveBans(ctx context.Context) ([]*moderation.Ban, error) {
	var rows []banRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectActiveBans)
	if err != nil {
		return nil, fmt.Errorf("failed to find active bans: %w", err)
	}

	// Convert rows to domain entities
	bans := make([]*moderation.Ban, 0, len(rows))
	for _, row := range rows {
		ban, err := rowToBan(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to ban: %w", err)
		}
		bans = append(bans, ban)
	}

	return bans, nil
}

// FindExpiredBans retrieves all bans that have naturally expired.
func (r *BanRepository) FindExpiredBans(ctx context.Context) ([]*moderation.Ban, error) {
	var rows []banRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectExpiredBans)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired bans: %w", err)
	}

	// Convert rows to domain entities
	bans := make([]*moderation.Ban, 0, len(rows))
	for _, row := range rows {
		ban, err := rowToBan(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to ban: %w", err)
		}
		bans = append(bans, ban)
	}

	return bans, nil
}

// IsUserBanned checks if a user currently has an active ban.
func (r *BanRepository) IsUserBanned(ctx context.Context, userID identity.UserID) (bool, error) {
	var banned bool
	err := r.db.GetContext(ctx, &banned, sqlCheckUserBanned, userID.String())
	if err != nil {
		return false, fmt.Errorf("failed to check if user is banned: %w", err)
	}

	return banned, nil
}

// Save persists a ban to the repository.
// If the ban already exists, it is updated; otherwise, it is created.
func (r *BanRepository) Save(ctx context.Context, ban *moderation.Ban) error {
	// Check if ban exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM user_bans WHERE id = $1)", ban.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check ban existence: %w", err)
	}

	if exists {
		return r.update(ctx, ban)
	}
	return r.insert(ctx, ban)
}

// insert creates a new ban in the database.
func (r *BanRepository) insert(ctx context.Context, ban *moderation.Ban) error {
	var expiresAt *time.Time
	if ban.ExpiresAt() != nil {
		expiresAt = ban.ExpiresAt()
	}

	var revokedAt *time.Time
	if ban.RevokedAt() != nil {
		revokedAt = ban.RevokedAt()
	}

	var revokedBy *string
	if ban.RevokedBy() != nil {
		val := ban.RevokedBy().String()
		revokedBy = &val
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertBan,
		ban.ID().String(),
		ban.UserID().String(),
		ban.BannedBy().String(),
		ban.Reason(),
		expiresAt,
		ban.CreatedAt(),
		revokedAt,
		revokedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert ban: %w", err)
	}

	return nil
}

// update updates an existing ban in the database.
func (r *BanRepository) update(ctx context.Context, ban *moderation.Ban) error {
	var revokedAt *time.Time
	if ban.RevokedAt() != nil {
		revokedAt = ban.RevokedAt()
	}

	var revokedBy *string
	if ban.RevokedBy() != nil {
		val := ban.RevokedBy().String()
		revokedBy = &val
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateBan,
		ban.ID().String(),
		revokedAt,
		revokedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to update ban: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return moderation.ErrBanNotFound
	}

	return nil
}

// rowToBan converts a database row to a domain Ban entity.
func rowToBan(row banRow) (*moderation.Ban, error) {
	// Parse IDs
	banID, err := moderation.ParseBanID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid ban id: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	bannedBy, err := identity.ParseUserID(row.BannedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid banned by id: %w", err)
	}

	// Parse nullable fields
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		expiresAt = &row.ExpiresAt.Time
	}

	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}

	var revokedBy *identity.UserID
	if row.RevokedBy.Valid {
		id, err := identity.ParseUserID(row.RevokedBy.String)
		if err != nil {
			return nil, fmt.Errorf("invalid revoked by id: %w", err)
		}
		revokedBy = &id
	}

	// Reconstitute ban without validation or events
	ban := moderation.ReconstructBan(
		banID,
		userID,
		bannedBy,
		row.Reason,
		expiresAt,
		row.CreatedAt,
		revokedAt,
		revokedBy,
	)

	return ban, nil
}
