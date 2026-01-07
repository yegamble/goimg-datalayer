package activity

import (
	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ActivityCreated is emitted when a new activity is created.
type ActivityCreated struct {
	shared.BaseEvent
	ActivityID   ActivityID
	ActorID      identity.UserID
	ActivityType ActivityType
	TargetID     uuid.UUID
	TargetType   TargetType
}

// NewActivityCreated creates a new ActivityCreated event.
func NewActivityCreated(
	activityID ActivityID,
	actorID identity.UserID,
	activityType ActivityType,
	targetID uuid.UUID,
	targetType TargetType,
) ActivityCreated {
	return ActivityCreated{
		BaseEvent:    shared.NewBaseEvent("activity.created", activityID.String()),
		ActivityID:   activityID,
		ActorID:      actorID,
		ActivityType: activityType,
		TargetID:     targetID,
		TargetType:   targetType,
	}
}
