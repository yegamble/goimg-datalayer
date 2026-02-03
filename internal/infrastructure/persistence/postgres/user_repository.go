package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for user operations.
const (
	sqlUpsertUser = `
		INSERT INTO users (id, email, username, password_hash, role, status, display_name, bio, infected_file_count, created_at, updated_at, user_type, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE
		SET email = EXCLUDED.email,
		    username = EXCLUDED.username,
		    password_hash = EXCLUDED.password_hash,
		    role = EXCLUDED.role,
		    status = EXCLUDED.status,
		    display_name = EXCLUDED.display_name,
		    bio = EXCLUDED.bio,
		    infected_file_count = EXCLUDED.infected_file_count,
		    updated_at = EXCLUDED.updated_at,
		    user_type = EXCLUDED.user_type,
		    ip_address = EXCLUDED.ip_address,
		    expires_at = EXCLUDED.expires_at
		WHERE users.deleted_at IS NULL
	`

	sqlSelectUserByID = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, infected_file_count, created_at, updated_at, user_type, ip_address, expires_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectUserByEmail = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, infected_file_count, created_at, updated_at, user_type, ip_address, expires_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	sqlSelectUserByUsername = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, infected_file_count, created_at, updated_at, user_type, ip_address, expires_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
	`

	sqlSoftDeleteUser = `
		UPDATE users
		SET deleted_at = $2,
		    status = 'deleted',
		    updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlFindExpiredGuests = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, infected_file_count, created_at, updated_at, user_type, ip_address, expires_at
		FROM users
		WHERE user_type = 'guest'
		  AND expires_at <= $1
		  AND deleted_at IS NULL
		ORDER BY expires_at ASC
		LIMIT $2
	`
)

// userRow represents a user row in the database.
type userRow struct {
	ID                string         `db:"id"`
	Email             string         `db:"email"`
	Username          string         `db:"username"`
	PasswordHash      string         `db:"password_hash"`
	Role              string         `db:"role"`
	Status            string         `db:"status"`
	DisplayName       string         `db:"display_name"`
	Bio               string         `db:"bio"`
	InfectedFileCount int            `db:"infected_file_count"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
	UserType          string         `db:"user_type"`
	IPAddress         sql.NullString `db:"ip_address"`
	ExpiresAt         sql.NullTime   `db:"expires_at"`
}

// UserRepository implements the identity.UserRepository interface for PostgreSQL.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository with the given database connection.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// NextID generates the next available UserID.
func (r *UserRepository) NextID() identity.UserID {
	return identity.NewUserID()
}

// FindByID retrieves a user by their unique ID.
func (r *UserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	var row userRow
	if err := r.db.GetContext(ctx, &row, sqlSelectUserByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	user, err := rowToUser(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to user: %w", err)
	}

	return user, nil
}

// FindByEmail retrieves a user by their email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	var row userRow
	if err := r.db.GetContext(ctx, &row, sqlSelectUserByEmail, email.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	user, err := rowToUser(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to user: %w", err)
	}

	return user, nil
}

// FindByUsername retrieves a user by their username.
func (r *UserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	var row userRow
	if err := r.db.GetContext(ctx, &row, sqlSelectUserByUsername, username.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	user, err := rowToUser(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to user: %w", err)
	}

	return user, nil
}

// Save persists a user to the repository.
// If the user already exists, it is updated; otherwise, it is created.
// Uses UPSERT (INSERT ... ON CONFLICT) for efficiency and concurrency safety.
func (r *UserRepository) Save(ctx context.Context, user *identity.User) error {
	// Convert nullable fields to sql.Null types
	var ipAddress sql.NullString
	if user.IPAddress() != nil {
		ipAddress = sql.NullString{String: *user.IPAddress(), Valid: true}
	}

	var expiresAt sql.NullTime
	if user.ExpiresAt() != nil {
		expiresAt = sql.NullTime{Time: *user.ExpiresAt(), Valid: true}
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpsertUser,
		user.ID().String(),
		user.Email().String(),
		user.Username().String(),
		user.PasswordHash().String(),
		user.Role().String(),
		user.Status().String(),
		user.DisplayName(),
		user.Bio(),
		user.InfectedFileCount(),
		user.CreatedAt(),
		user.UpdatedAt(),
		user.UserType().String(),
		ipAddress,
		expiresAt,
	)
	if err != nil {
		// Handle unique constraint violations
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Constraint {
			case "users_email_key":
				return identity.ErrEmailExists
			case "users_username_key":
				return identity.ErrUsernameExists
			}
		}
		return fmt.Errorf("failed to save user: %w", err)
	}

	// If no rows were affected, it means the user exists but is soft-deleted
	// (because of WHERE users.deleted_at IS NULL in the ON CONFLICT clause)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrUserNotFound
	}

	return nil
}

// Delete removes a user from the repository (soft delete).
func (r *UserRepository) Delete(ctx context.Context, id identity.UserID) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, sqlSoftDeleteUser, id.String(), now)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrUserNotFound
	}

	return nil
}

// rowToUser converts a database row to a domain User entity.
func rowToUser(row userRow) (*identity.User, error) {
	// Parse UUID
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	userID, err := identity.ParseUserID(id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id: %w", err)
	}

	// Parse email
	email, err := identity.NewEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	// Parse username
	username, err := identity.NewUsername(row.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to parse username: %w", err)
	}

	// Parse password hash
	passwordHash, err := identity.ParsePasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse password hash: %w", err)
	}

	// Parse role
	role, err := identity.ParseRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to parse role: %w", err)
	}

	// Parse status
	status, err := identity.ParseUserStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	// Parse user type
	userType, err := identity.ParseUserType(row.UserType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user type: %w", err)
	}

	// Parse nullable IP address
	var ipAddress *string
	if row.IPAddress.Valid {
		ipAddress = &row.IPAddress.String
	}

	// Parse nullable expires_at
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		expiresAt = &row.ExpiresAt.Time
	}

	// Reconstitute user without validation or events
	user := identity.ReconstructUser(
		userID,
		email,
		username,
		passwordHash,
		role,
		status,
		row.DisplayName,
		row.Bio,
		row.InfectedFileCount,
		row.CreatedAt,
		row.UpdatedAt,
		userType,
		ipAddress,
		expiresAt,
	)

	return user, nil
}

// FindExpiredGuests retrieves all guest users whose expiration date has passed.
// This method is used by cleanup jobs to remove expired guest accounts.
// The asOf parameter specifies the cutoff time (typically time.Now().UTC()).
// The limit parameter prevents loading too many records at once.
func (r *UserRepository) FindExpiredGuests(
	ctx context.Context,
	asOf time.Time,
	limit int,
) ([]*identity.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, status, display_name, bio,
		       infected_file_count, created_at, updated_at, user_type, ip_address, expires_at
		FROM users
		WHERE user_type = 'guest'
		  AND expires_at IS NOT NULL
		  AND expires_at <= $1
		  AND deleted_at IS NULL
		ORDER BY expires_at ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, asOf, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired guests: %w", err)
	}
	defer rows.Close()

	var users []*identity.User
	for rows.Next() {
		var row userRow
		if err := rows.Scan(
			&row.ID,
			&row.Email,
			&row.Username,
			&row.PasswordHash,
			&row.Role,
			&row.Status,
			&row.DisplayName,
			&row.Bio,
			&row.InfectedFileCount,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.UserType,
			&row.IPAddress,
			&row.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expired guest row: %w", err)
		}

		user, err := rowToUser(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired guest rows: %w", err)
	}

	return users, nil
}
