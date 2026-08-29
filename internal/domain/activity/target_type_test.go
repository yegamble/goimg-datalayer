package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestTargetType_Valid(t *testing.T) {
	assert.True(t, activity.TargetTypeImage.Valid())
	assert.True(t, activity.TargetTypeUser.Valid())
	assert.True(t, activity.TargetTypeAlbum.Valid())
	assert.True(t, activity.TargetTypeComment.Valid())
	assert.False(t, activity.TargetType("invalid").Valid())
}

func TestTargetType_String(t *testing.T) {
	assert.Equal(t, "image", activity.TargetTypeImage.String())
}

func TestTargetType_ParseTargetType(t *testing.T) {
	tt, err := activity.ParseTargetType("user")
	assert.NoError(t, err)
	assert.Equal(t, activity.TargetTypeUser, tt)

	_, err = activity.ParseTargetType("invalid")
	assert.Error(t, err)
}
