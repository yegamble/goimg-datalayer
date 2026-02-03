package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for activity operations.
const (
	sqlInsertActivity = `
		INSERT INTO activities (id, actor_id, activity_type, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	sqlSelectActivitiesByActor = `
		SELECT id, actor_id, activity_type, target_type, target_id, metadata, created_at
		FROM activities
		WHERE actor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountActivitiesByActor = `
		SELECT COUNT(*)
		FROM activities
		WHERE actor_id = $1
	`

	// Optimized feed query with capped inner LIMIT in code
	sqlSelectFeedForUser = `
		SELECT a.id, a.actor_id, a.activity_type, a.target_type, a.target_id, a.metadata, a.created_at
		FROM user_follows uf
		CROSS JOIN LATERAL (
			SELECT *
			FROM activities
			WHERE actor_id = uf.followed_id
			ORDER BY created_at DESC
			LIMIT $4
		) a
		WHERE uf.follower_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountFeedForUser = `
		SELECT COUNT(*)
		FROM activities a
		INNER JOIN user_follows uf ON a.actor_id = uf.followed_id
		WHERE uf.follower_id = $1
	`

	sqlDeleteActivitiesOlderThan = `
		DELETE FROM activities
		WHERE created_at < $1
	`
)

// activityRow represents an activity row in the database.
type activityRow struct {
	ID           string    `db:"id"`
	ActorID      string    `db:"actor_id"`
	ActivityType string    `db:"activity_type"`
	TargetType   string    `db:"target_type"`
	TargetID     string    `db:"target_id"`
	Metadata     []byte    `db:"metadata"`
	CreatedAt    time.Time `db:"created_at"`
}

// toDomain converts a database row to a domain Activity entity.
func (r *activityRow) toDomain() (*activity.Activity, error) {
	activityID, err := activity.ParseActivityID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse activity id: %w", err)
	}

	actorID, err := identity.ParseUserID(r.ActorID)
	if err != nil {
		return nil, fmt.Errorf("parse actor id: %w", err)
	}

	activityType, err := activity.ParseActivityType(r.ActivityType)
	if err != nil {
		return nil, fmt.Errorf("parse activity type: %w", err)
	}

	targetType, err := activity.ParseTargetType(r.TargetType)
	if err != nil {
		return nil, fmt.Errorf("parse target type: %w", err)
	}

	targetID, err := uuid.Parse(r.TargetID)
	if err != nil {
		return nil, fmt.Errorf("parse target id: %w", err)
	}

	var metadata map[string]string
	if len(r.Metadata) > 0 {
		if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return activity.ReconstructActivity(
		activityID,
		actorID,
		activityType,
		targetID,
		targetType,
		metadata,
		r.CreatedAt,
	), nil
}

// ActivityRepository implements the activity.ActivityRepository interface for PostgreSQL.
type ActivityRepository struct {
	db *sqlx.DB
}

// NewActivityRepository creates a new ActivityRepository with the given database connection.
func NewActivityRepository(db *sqlx.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Save persists an activity to the repository.
func (r *ActivityRepository) Save(ctx context.Context, act *activity.Activity) error {
	metadata, err := json.Marshal(act.Metadata())
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(
		ctx,
		sqlInsertActivity,
		act.ID().String(),
		act.ActorID().String(),
		act.ActivityType().String(),
		act.TargetType().String(),
		act.TargetID().String(),
		metadata,
		act.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save activity: %w", err)
	}

	return nil
}

// FindByActor retrieves activities performed by a specific actor.
func (r *ActivityRepository) FindByActor(
	ctx context.Context,
	actorID identity.UserID,
	pagination shared.Pagination,
) ([]*activity.Activity, int, error) {
	// Get activities
	var rows []activityRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectActivitiesByActor,
		actorID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find activities by actor: %w", err)
	}

	// Convert to domain entities
	activities := make([]*activity.Activity, 0, len(rows))
	for _, row := range rows {
		act, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert activity to domain: %w", err)
		}
		activities = append(activities, act)
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountActivitiesByActor, actorID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activities by actor: %w", err)
	}

	return activities, total, nil
}

// FindFeedForUser retrieves activities from users that the specified user follows.
// This is the main feed query that powers the user's activity feed.
func (r *ActivityRepository) FindFeedForUser(
	ctx context.Context,
	userID identity.UserID,
	pagination shared.Pagination,
) ([]*activity.Activity, int, error) {

	// Calculate inner limit for LATERAL JOIN optimization
	// Cap the inner limit to prevent unbounded scans on large offsets
	innerLimit := pagination.Limit() + pagination.Offset()
	if innerLimit > 1000 {
		innerLimit = 1000
	}

	// Get feed activities
	var rows []activityRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectFeedForUser,
		userID.String(),
		pagination.Limit(),
		pagination.Offset(),
		innerLimit, // $4: Capped inner limit
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find feed for user: %w", err)
	}

	// Convert to domain entities
	activities := make([]*activity.Activity, 0, len(rows))
	for _, row := range rows {
		act, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert activity to domain: %w", err)
		}
		activities = append(activities, act)
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, sqlCountFeedForUser, userID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count feed for user: %w", err)
	}

	return activities, total, nil
}

// DeleteOlderThan removes activities created before the specified time.
// This is used for cleanup jobs to prevent the activities table from growing indefinitely.
func (r *ActivityRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteActivitiesOlderThan, before)
	if err != nil {
		return fmt.Errorf("failed to delete old activities: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of deleted activities (optional, can be used by caller)
	_ = rowsAffected

	return nil
}
