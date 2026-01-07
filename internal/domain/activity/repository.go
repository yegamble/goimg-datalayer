package activity

import (
	"context"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ActivityRepository defines the interface for persisting and retrieving Activity entities.
// Implementations should be provided in the infrastructure layer.
type ActivityRepository interface {
	// Save persists an activity to the repository.
	Save(ctx context.Context, activity *Activity) error

	// FindByActor retrieves activities performed by a specific actor.
	// Returns a slice of Activity entities and supports pagination.
	FindByActor(ctx context.Context, actorID identity.UserID, pagination shared.Pagination) ([]*Activity, int, error)

	// FindFeedForUser retrieves activities from users that the specified user follows.
	// This is the main feed query that powers the user's activity feed.
	// Returns a slice of Activity entities and supports pagination.
	FindFeedForUser(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]*Activity, int, error)

	// DeleteOlderThan removes activities created before the specified time.
	// This is used for cleanup jobs to prevent the activities table from growing indefinitely.
	DeleteOlderThan(ctx context.Context, before time.Time) error
}
