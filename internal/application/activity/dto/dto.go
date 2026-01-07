// Package dto provides data transfer objects for activity operations.
package dto

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

// ActivityDTO represents a single activity in API responses.
type ActivityDTO struct {
	ID           string            `json:"id"`
	ActorID      string            `json:"actor_id"`
	ActivityType string            `json:"activity_type"`
	TargetType   string            `json:"target_type"`
	TargetID     string            `json:"target_id"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
}

// FromDomain converts a domain Activity aggregate to an ActivityDTO.
func FromDomain(act *activity.Activity) ActivityDTO {
	return ActivityDTO{
		ID:           act.ID().String(),
		ActorID:      act.ActorID().String(),
		ActivityType: act.ActivityType().String(),
		TargetType:   act.TargetType().String(),
		TargetID:     act.TargetID().String(),
		Metadata:     act.Metadata(),
		CreatedAt:    act.CreatedAt(),
	}
}

// ActivityFeedDTO represents a paginated list of activities for a user's feed.
type ActivityFeedDTO struct {
	Activities []ActivityDTO `json:"activities"`
	TotalCount int           `json:"total_count"`
	Offset     int           `json:"offset"`
	Limit      int           `json:"limit"`
}
