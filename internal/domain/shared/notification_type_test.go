package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	assert.True(t, shared.NotificationTypeAccountBanned.RequiresEmail())
	assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
	assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
}

func TestNotificationType_IsValid(t *testing.T) {
	assert.True(t, shared.NotificationTypeNewFollower.IsValid())
	assert.False(t, shared.NotificationType("invalid").IsValid())
}

func TestNotificationType_String(t *testing.T) {
	assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
}
