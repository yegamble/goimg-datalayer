package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group activity operations.
const (
	sqlInsertGroupActivity = `
		INSERT INTO group_activities (
			id, group_id, actor_id, activity_type, target_id, target_type, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	sqlSelectActivitiesByGroup = `
		SELECT id, group_id, actor_id, activity_type, target_id, target_type, metadata, created_at
		FROM group_activities
		WHERE group_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountActivitiesByGroup = `
		SELECT COUNT(*)
		FROM group_activities
		WHERE group_id = $1
	`

	sqlSelectActivitiesByGroupAndType = `
		SELECT id, group_id, actor_id, activity_type, target_id, target_type, metadata, created_at
		FROM group_activities
		WHERE group_id = $1 AND activity_type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	sqlCountActivitiesByGroupAndType = `
		SELECT COUNT(*) FROM group_activities
		WHERE group_id = $1 AND activity_type = $2
	`
)

// groupActivityRow represents a group activity row in the database.
type groupActivityRow struct {
	ID           string         `db:"id"`
	GroupID      string         `db:"group_id"`
	ActorID      string         `db:"actor_id"`
	ActivityType string         `db:"activity_type"`
	TargetID     sql.NullString `db:"target_id"`
	TargetType   sql.NullString `db:"target_type"`
	Metadata     []byte         `db:"metadata"` // Raw JSON bytes from JSONB column
	CreatedAt    time.Time      `db:"created_at"`
}

// GroupActivityRepository implements the community.GroupActivityRepository interface for PostgreSQL.
type GroupActivityRepository struct {
	db *sqlx.DB
}

// NewGroupActivityRepository creates a new GroupActivityRepository with the given database connection.
func NewGroupActivityRepository(db *sqlx.DB) *GroupActivityRepository {
	return &GroupActivityRepository{db: db}
}

// Save persists a group activity to storage.
// This is an append-only operation - activities are never updated.
func (r *GroupActivityRepository) Save(ctx context.Context, activity *community.GroupActivity) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(activity.Metadata())
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Handle optional target_id and target_type
	var targetID sql.NullString
	var targetTypeSQL sql.NullString

	if activity.TargetID() != nil {
		targetID = sql.NullString{String: *activity.TargetID(), Valid: true}
	}

	if activity.TargetType() != nil {
		targetTypeSQL = sql.NullString{String: activity.TargetType().String(), Valid: true}
	}

	_, err = r.db.ExecContext(
		ctx,
		sqlInsertGroupActivity,
		activity.ID().String(),
		activity.GroupID().String(),
		activity.ActorID().String(),
		activity.ActivityType().String(),
		targetID,
		targetTypeSQL,
		metadataJSON,
		activity.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save group activity: %w", err)
	}

	return nil
}

// FindByGroup retrieves activities for a specific group with pagination.
func (r *GroupActivityRepository) FindByGroup(
	ctx context.Context,
	groupID community.GroupID,
	pagination shared.Pagination,
) ([]*community.GroupActivity, int, error) {
	var rows []groupActivityRow
	if err := r.db.SelectContext(ctx, &rows, sqlSelectActivitiesByGroup, groupID.String(), pagination.Limit, pagination.Offset); err != nil {
		return nil, 0, fmt.Errorf("failed to find activities by group: %w", err)
	}

	activities := make([]*community.GroupActivity, 0, len(rows))
	for _, row := range rows {
		activity, err := rowToGroupActivity(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to activity: %w", err)
		}
		activities = append(activities, activity)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, sqlCountActivitiesByGroup, groupID.String()); err != nil {
		return nil, 0, fmt.Errorf("failed to count group activities: %w", err)
	}

	return activities, total, nil
}

// FindByGroupAndType retrieves activities for a specific group filtered by activity type.
func (r *GroupActivityRepository) FindByGroupAndType(
	ctx context.Context,
	groupID community.GroupID,
	activityType community.ActivityType,
	pagination shared.Pagination,
) ([]*community.GroupActivity, int, error) {
	var rows []groupActivityRow
	if err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectActivitiesByGroupAndType,
		groupID.String(),
		activityType.String(),
		pagination.Limit,
		pagination.Offset,
	); err != nil {
		return nil, 0, fmt.Errorf("failed to find activities by group and type: %w", err)
	}

	activities := make([]*community.GroupActivity, 0, len(rows))
	for _, row := range rows {
		activity, err := rowToGroupActivity(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to activity: %w", err)
		}
		activities = append(activities, activity)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, sqlCountActivitiesByGroupAndType, groupID.String(), activityType.String()); err != nil {
		return nil, 0, fmt.Errorf("failed to count group activities: %w", err)
	}

	return activities, total, nil
}

// rowToGroupActivity converts a database row to a GroupActivity domain entity.
func rowToGroupActivity(row groupActivityRow) (*community.GroupActivity, error) {
	activityID, err := community.ParseGroupActivityID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse activity id: %w", err)
	}

	groupID, err := community.ParseGroupID(row.GroupID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}

	actorID, err := identity.ParseUserID(row.ActorID)
	if err != nil {
		return nil, fmt.Errorf("parse actor id: %w", err)
	}

	activityType := community.ActivityType(row.ActivityType)
	if !activityType.IsValid() {
		return nil, fmt.Errorf("invalid activity type: %s", row.ActivityType)
	}

	var targetID *string
	if row.TargetID.Valid {
		targetID = &row.TargetID.String
	}

	var targetType *community.TargetType
	if row.TargetType.Valid {
		tt := community.TargetType(row.TargetType.String)
		if !tt.IsValid() {
			return nil, fmt.Errorf("invalid target type: %s", row.TargetType.String)
		}
		targetType = &tt
	}

	// Unmarshal JSONB metadata
	var metadata map[string]interface{}
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return community.ReconstructGroupActivity(
		activityID,
		groupID,
		actorID,
		activityType,
		targetID,
		targetType,
		metadata,
		row.CreatedAt,
	), nil
}
