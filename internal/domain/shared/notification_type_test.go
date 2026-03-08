package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType(t *testing.T) {
	assert.True(t, shared.NotificationTypeAccountBanned.RequiresEmail())
	assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
	assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
	assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
	assert.True(t, shared.NotificationTypeNewFollower.IsValid())
	assert.False(t, shared.NotificationType("invalid").IsValid())
	assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
}
