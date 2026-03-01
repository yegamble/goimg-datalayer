package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType(t *testing.T) {
	t.Run("RequiresEmail returns true for critical types", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeAccountSuspended.RequiresEmail())
		assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
	})

	t.Run("IsAdminOnly returns true for admin types", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
		assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
	})

	t.Run("IsValid validates all known types", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeNewFollower.IsValid())
		assert.True(t, shared.NotificationTypeAbuseReport.IsValid())
		assert.False(t, shared.NotificationType("invalid").IsValid())
	})

	t.Run("String returns string representation", func(t *testing.T) {
		assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
	})
}
