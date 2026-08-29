package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestTargetType(t *testing.T) {
	assert.True(t, activity.TargetTypeImage.Valid())
	assert.False(t, activity.TargetType("invalid").Valid())
	assert.Equal(t, "image", activity.TargetTypeImage.String())

	typ, err := activity.ParseTargetType("user")
	require.NoError(t, err)
	assert.Equal(t, activity.TargetTypeUser, typ)

	_, err = activity.ParseTargetType("invalid")
	assert.Error(t, err)
}
