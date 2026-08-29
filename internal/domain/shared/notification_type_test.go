package shared

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	assert.True(t, NotificationTypeAccountSuspended.RequiresEmail())
	assert.False(t, NotificationTypeNewFollower.RequiresEmail())
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	assert.True(t, NotificationTypeAbuseReport.IsAdminOnly())
	assert.False(t, NotificationTypeNewFollower.IsAdminOnly())
}

func TestNotificationType_IsValid(t *testing.T) {
	assert.True(t, NotificationTypeNewFollower.IsValid())
	assert.False(t, NotificationType("invalid").IsValid())
}

func TestNotificationType_String(t *testing.T) {
	assert.Equal(t, "new_follower", NotificationTypeNewFollower.String())
}
