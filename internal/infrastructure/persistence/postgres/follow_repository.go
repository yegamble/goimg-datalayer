package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for follow operations.
const (
	sqlInsertFollow = `
		INSERT INTO user_follows (follower_id, followed_id, created_at)
		VALUES ($1, $2, $3)
	`

	sqlDeleteFollow = `
		DELETE FROM user_follows
		WHERE follower_id = $1 AND followed_id = $2
	`

	sqlCheckFollowExists = `
		SELECT EXISTS(
			SELECT 1 FROM user_follows
			WHERE follower_id = $1 AND followed_id = $2
		)
	`

	sqlSelectFollowers = `
		SELECT follower_id, followed_id, created_at
		FROM user_follows
		WHERE followed_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountFollowers = `
		SELECT COUNT(*)
		FROM user_follows
		WHERE followed_id = $1
	`

	sqlSelectFollowing = `
		SELECT follower_id, followed_id, created_at
		FROM user_follows
		WHERE follower_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountFollowing = `
		SELECT COUNT(*)
		FROM user_follows
		WHERE follower_id = $1
	`
)

// followRow represents a follow relationship row in the database.
type followRow struct {
	FollowerID string    `db:"follower_id"`
	FollowedID string    `db:"followed_id"`
	CreatedAt  time.Time `db:"created_at"`
}

// toDomain converts a database row to a domain Follow entity.
func (r *followRow) toDomain() (*identity.Follow, error) {
	followerID, err := identity.ParseUserID(r.FollowerID)
	if err != nil {
		return nil, fmt.Errorf("parse follower id: %w", err)
	}

	followedID, err := identity.ParseUserID(r.FollowedID)
	if err != nil {
		return nil, fmt.Errorf("parse followed id: %w", err)
	}

	return identity.ReconstructFollow(followerID, followedID, r.CreatedAt), nil
}

// FollowRepository implements the identity.FollowRepository interface for PostgreSQL.
type FollowRepository struct {
	db *sqlx.DB
}

// NewFollowRepository creates a new FollowRepository with the given database connection.
func NewFollowRepository(db *sqlx.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

// Save persists a follow relationship to the repository.
// Returns ErrFollowAlreadyExists if the follow relationship already exists.
func (r *FollowRepository) Save(ctx context.Context, follow *identity.Follow) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertFollow,
		follow.FollowerID().String(),
		follow.FollowedID().String(),
		follow.CreatedAt(),
	)
	if err != nil {
		// Check for unique constraint violation
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // unique_violation
				return identity.ErrFollowAlreadyExists
			}
			// Check constraint violation (follower_id != followed_id)
			if pqErr.Code == "23514" { // check_violation
				return identity.ErrCannotFollowSelf
			}
		}
		return fmt.Errorf("failed to save follow: %w", err)
	}

	return nil
}

// Delete removes a follow relationship from the repository.
// Returns ErrFollowNotFound if the relationship does not exist.
func (r *FollowRepository) Delete(ctx context.Context, followerID, followedID identity.UserID) error {
	result, err := r.db.ExecContext(
		ctx,
		sqlDeleteFollow,
		followerID.String(),
		followedID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete follow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrFollowNotFound
	}

	return nil
}

// Exists checks whether a follow relationship exists between two users.
func (r *FollowRepository) Exists(ctx context.Context, followerID, followedID identity.UserID) (bool, error) {
	var exists bool
	err := r.db.GetContext(
		ctx,
		&exists,
		sqlCheckFollowExists,
		followerID.String(),
		followedID.String(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to check follow existence: %w", err)
	}

	return exists, nil
}

// FindFollowers retrieves all users following the specified user (pagination supported).
// Returns a slice of Follow entities and the total count.
func (r *FollowRepository) FindFollowers(
	ctx context.Context,
	userID identity.UserID,
	limit, offset int,
) ([]*identity.Follow, int, error) {
	// Get followers
	var rows []followRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectFollowers,
		userID.String(),
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find followers: %w", err)
	}

	// Convert to domain entities
	follows := make([]*identity.Follow, 0, len(rows))
	for _, row := range rows {
		follow, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert follow to domain: %w", err)
		}
		follows = append(follows, follow)
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountFollowers, userID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count followers: %w", err)
	}

	return follows, total, nil
}

// FindFollowing retrieves all users that the specified user is following (pagination supported).
// Returns a slice of Follow entities and the total count.
func (r *FollowRepository) FindFollowing(
	ctx context.Context,
	userID identity.UserID,
	limit, offset int,
) ([]*identity.Follow, int, error) {
	// Get following
	var rows []followRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectFollowing,
		userID.String(),
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find following: %w", err)
	}

	// Convert to domain entities
	follows := make([]*identity.Follow, 0, len(rows))
	for _, row := range rows {
		follow, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert follow to domain: %w", err)
		}
		follows = append(follows, follow)
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountFollowing, userID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count following: %w", err)
	}

	return follows, total, nil
}

// CountFollowers returns the number of followers for a user.
func (r *FollowRepository) CountFollowers(ctx context.Context, userID identity.UserID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountFollowers, userID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count followers: %w", err)
	}

	return count, nil
}

// CountFollowing returns the number of users that the specified user is following.
func (r *FollowRepository) CountFollowing(ctx context.Context, userID identity.UserID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountFollowing, userID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count following: %w", err)
	}

	return count, nil
}
