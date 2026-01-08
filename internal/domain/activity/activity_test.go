package activity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewActivity_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()
	metadata := map[string]string{
		"image_title": "Sunset",
		"image_url":   "https://example.com/image.jpg",
	}

	// Act
	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		metadata,
	)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, act)
	assert.False(t, act.ID().IsZero())
	assert.Equal(t, actorID, act.ActorID())
	assert.Equal(t, activity.ActivityTypeImageUploaded, act.ActivityType())
	assert.Equal(t, targetID, act.TargetID())
	assert.Equal(t, activity.TargetTypeImage, act.TargetType())
	assert.Equal(t, metadata, act.Metadata())
	assert.False(t, act.CreatedAt().IsZero())
	assert.Len(t, act.Events(), 1)
}

func TestNewActivity_WithNilMetadata(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()

	// Act
	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeUserFollowed,
		targetID,
		activity.TargetTypeUser,
		nil,
	)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, act)
	assert.NotNil(t, act.Metadata())
	assert.Empty(t, act.Metadata())
}

func TestNewActivity_ActorRequired(t *testing.T) {
	t.Parallel()

	// Arrange
	targetID := uuid.New()

	// Act
	act, err := activity.NewActivity(
		identity.UserID{}, // Zero value
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		nil,
	)

	// Assert
	assert.Nil(t, act)
	require.ErrorIs(t, err, activity.ErrActorRequired)
}

func TestNewActivity_TargetRequired(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()

	// Act
	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeImageUploaded,
		uuid.Nil, // Zero value
		activity.TargetTypeImage,
		nil,
	)

	// Assert
	assert.Nil(t, act)
	require.ErrorIs(t, err, activity.ErrTargetRequired)
}

func TestNewActivity_InvalidActivityType(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()

	// Act
	act, err := activity.NewActivity(
		actorID,
		activity.ActivityType("invalid_type"),
		targetID,
		activity.TargetTypeImage,
		nil,
	)

	// Assert
	assert.Nil(t, act)
	require.ErrorIs(t, err, activity.ErrInvalidActivityType)
}

func TestNewActivity_InvalidTargetType(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()

	// Act
	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetType("invalid_target"),
		nil,
	)

	// Assert
	assert.Nil(t, act)
	require.ErrorIs(t, err, activity.ErrInvalidTargetType)
}

func TestActivity_Metadata_Immutability(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()
	metadata := map[string]string{
		"key": "value",
	}

	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		metadata,
	)
	require.NoError(t, err)

	// Act - modify returned metadata
	returnedMetadata := act.Metadata()
	returnedMetadata["new_key"] = "new_value"

	// Assert - original metadata unchanged
	actualMetadata := act.Metadata()
	assert.NotContains(t, actualMetadata, "new_key")
	assert.Equal(t, "value", actualMetadata["key"])
}

func TestReconstructActivity(t *testing.T) {
	t.Parallel()

	// Arrange
	activityID := activity.NewActivityID()
	actorID := identity.NewUserID()
	targetID := uuid.New()
	metadata := map[string]string{"key": "value"}
	refActivity, err := activity.NewActivity(actorID, activity.ActivityTypeImageUploaded, targetID, activity.TargetTypeImage, nil)
	require.NoError(t, err)
	require.NotNil(t, refActivity)

	// Act
	act := activity.ReconstructActivity(
		activityID,
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		metadata,
		refActivity.CreatedAt(),
	)

	// Assert
	assert.NotNil(t, act)
	assert.Equal(t, activityID, act.ID())
	assert.Equal(t, actorID, act.ActorID())
	assert.Equal(t, activity.ActivityTypeImageUploaded, act.ActivityType())
	assert.Equal(t, targetID, act.TargetID())
	assert.Equal(t, activity.TargetTypeImage, act.TargetType())
	assert.Equal(t, metadata, act.Metadata())
	assert.Equal(t, refActivity.CreatedAt(), act.CreatedAt())
	assert.Empty(t, act.Events()) // No events on reconstruction
}

func TestActivity_ClearEvents(t *testing.T) {
	t.Parallel()

	// Arrange
	actorID := identity.NewUserID()
	targetID := uuid.New()

	act, err := activity.NewActivity(
		actorID,
		activity.ActivityTypeImageUploaded,
		targetID,
		activity.TargetTypeImage,
		nil,
	)
	require.NoError(t, err)
	require.Len(t, act.Events(), 1)

	// Act
	act.ClearEvents()

	// Assert
	assert.Empty(t, act.Events())
}
