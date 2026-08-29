package activity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestActivity_NewActivity_Errors(t *testing.T) {
	validUserID := identity.NewUserID()
	validTargetID := uuid.New()

	_, err := activity.NewActivity(
		identity.UserID{},
		activity.ActivityTypeImageUploaded,
		validTargetID,
		activity.TargetTypeImage,
		nil,
	)
	assert.ErrorIs(t, err, activity.ErrActorRequired)

	_, err = activity.NewActivity(
		validUserID,
		activity.ActivityType("invalid"),
		validTargetID,
		activity.TargetTypeImage,
		nil,
	)
	assert.ErrorIs(t, err, activity.ErrInvalidActivityType)

	_, err = activity.NewActivity(
		validUserID,
		activity.ActivityTypeImageUploaded,
		uuid.Nil,
		activity.TargetTypeImage,
		nil,
	)
	assert.ErrorIs(t, err, activity.ErrTargetRequired)

	_, err = activity.NewActivity(
		validUserID,
		activity.ActivityTypeImageUploaded,
		validTargetID,
		activity.TargetType("invalid"),
		nil,
	)
	assert.ErrorIs(t, err, activity.ErrInvalidTargetType)
}

func TestActivity_ReconstructActivity(t *testing.T) {
	id := activity.NewActivityID()
	actorID := identity.NewUserID()
	targetID := uuid.New()
	createdAt := time.Now().UTC()

	a := activity.ReconstructActivity(
		id,
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		nil,
		createdAt,
	)

	assert.Equal(t, id, a.ID())
	assert.Equal(t, actorID, a.ActorID())
	assert.Equal(t, activity.ActivityTypeImageUploaded, a.ActivityType())
	assert.Equal(t, targetID, a.TargetID())
	assert.Equal(t, activity.TargetTypeImage, a.TargetType())
	assert.NotNil(t, a.Metadata())
	assert.Equal(t, createdAt, a.CreatedAt())
	assert.Empty(t, a.Events())
}
