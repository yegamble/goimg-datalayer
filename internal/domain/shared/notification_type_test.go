package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType(t *testing.T) {
	t.Run("RequiresEmail", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeAccountSuspended.RequiresEmail())
		assert.True(t, shared.NotificationTypeAccountBanned.RequiresEmail())
		assert.True(t, shared.NotificationTypeAccountReinstated.RequiresEmail())
		assert.True(t, shared.NotificationTypeMalwareDetected.RequiresEmail())
		assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
		assert.False(t, shared.NotificationTypeNewPhotos.RequiresEmail())
	})

	t.Run("IsAdminOnly", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
		assert.True(t, shared.NotificationTypeReportEscalated.IsAdminOnly())
		assert.True(t, shared.NotificationTypeModActionRequired.IsAdminOnly())
		assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
	})

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, shared.NotificationTypeNewFollower.IsValid())
		assert.True(t, shared.NotificationTypeAccountBanned.IsValid())
		assert.False(t, shared.NotificationType("invalid_type").IsValid())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
		assert.Equal(t, "account_banned", shared.NotificationTypeAccountBanned.String())
	})
}
