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
	sqlInsertUser = `
		INSERT INTO users (id, email, username, password_hash, role, status, display_name, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	sqlUpdateUser = `
		UPDATE users
		SET email = $2,
		    username = $3,
		    password_hash = $4,
		    role = $5,
		    status = $6,
		    display_name = $7,
		    bio = $8,
		    updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectUserByID = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectUserByEmail = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	sqlSelectUserByUsername = `
		SELECT id, email, username, password_hash, role, status, display_name, bio, created_at, updated_at
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
)

// userRow represents a user row in the database.
type userRow struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	Role         string    `db:"role"`
	Status       string    `db:"status"`
	DisplayName  string    `db:"display_name"`
	Bio          string    `db:"bio"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
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
func (r *UserRepository) Save(ctx context.Context, user *identity.User) error {
	// Check if user exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", user.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		return r.update(ctx, user)
	}
	return r.insert(ctx, user)
}

// insert creates a new user in the database.
func (r *UserRepository) insert(ctx context.Context, user *identity.User) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertUser,
		user.ID().String(),
		user.Email().String(),
		user.Username().String(),
		user.PasswordHash().String(),
		user.Role().String(),
		user.Status().String(),
		user.DisplayName(),
		user.Bio(),
		user.CreatedAt(),
		user.UpdatedAt(),
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
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// update updates an existing user in the database.
func (r *UserRepository) update(ctx context.Context, user *identity.User) error {
	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateUser,
		user.ID().String(),
		user.Email().String(),
		user.Username().String(),
		user.PasswordHash().String(),
		user.Role().String(),
		user.Status().String(),
		user.DisplayName(),
		user.Bio(),
		user.UpdatedAt(),
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
		return fmt.Errorf("failed to update user: %w", err)
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
		row.CreatedAt,
		row.UpdatedAt,
	)

	return user, nil
}
