package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotificationType(t *testing.T) {
	t.Run("RequiresEmail", func(t *testing.T) {
		assert.True(t, NotificationTypeAccountSuspended.RequiresEmail())
		assert.True(t, NotificationTypeAccountBanned.RequiresEmail())
		assert.False(t, NotificationTypeNewFollower.RequiresEmail())
	})

	t.Run("IsAdminOnly", func(t *testing.T) {
		assert.True(t, NotificationTypeAbuseReport.IsAdminOnly())
		assert.True(t, NotificationTypeModActionRequired.IsAdminOnly())
		assert.False(t, NotificationTypeNewFollower.IsAdminOnly())
	})

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, NotificationTypeNewFollower.IsValid())
		assert.True(t, NotificationTypeAccountBanned.IsValid())
		assert.False(t, NotificationType("invalid").IsValid())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "new_follower", NotificationTypeNewFollower.String())
	})
}
