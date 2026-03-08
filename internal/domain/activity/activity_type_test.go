package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestActivityType(t *testing.T) {
	assert.Equal(t, "user_followed", string(activity.ActivityTypeUserFollowed))
	assert.True(t, activity.ActivityTypeImageUploaded.Valid())
	assert.False(t, activity.ActivityType("invalid").Valid())
	assert.Equal(t, "image_uploaded", activity.ActivityTypeImageUploaded.String())

	typ, err := activity.ParseActivityType("image_liked")
	require.NoError(t, err)
	assert.Equal(t, activity.ActivityTypeImageLiked, typ)

	_, err = activity.ParseActivityType("invalid")
	assert.Error(t, err)
}
