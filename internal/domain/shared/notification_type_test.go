package shared_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"testing"
)

func TestNotificationType(t *testing.T) {
	t.Parallel()

	t.Run("RequiresEmail", func(t *testing.T) {
		t.Parallel()
		assert.True(t, shared.NotificationTypeAccountBanned.RequiresEmail())
		assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
	})

	t.Run("IsAdminOnly", func(t *testing.T) {
		t.Parallel()
		assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
		assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
	})

	t.Run("IsValid", func(t *testing.T) {
		t.Parallel()
		assert.True(t, shared.NotificationTypeNewFollower.IsValid())
		assert.False(t, shared.NotificationType("invalid").IsValid())
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
	})
}
