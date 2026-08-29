package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	t.Parallel()

	assert.True(t, shared.NotificationTypeAccountSuspended.RequiresEmail())
	assert.True(t, shared.NotificationTypeAccountBanned.RequiresEmail())
	assert.True(t, shared.NotificationTypeAccountReinstated.RequiresEmail())
	assert.True(t, shared.NotificationTypeMalwareDetected.RequiresEmail())

	assert.False(t, shared.NotificationTypeNewFollower.RequiresEmail())
	assert.False(t, shared.NotificationTypeAbuseReport.RequiresEmail())
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	t.Parallel()

	assert.True(t, shared.NotificationTypeAbuseReport.IsAdminOnly())
	assert.True(t, shared.NotificationTypeReportEscalated.IsAdminOnly())
	assert.True(t, shared.NotificationTypeModActionRequired.IsAdminOnly())

	assert.False(t, shared.NotificationTypeNewFollower.IsAdminOnly())
	assert.False(t, shared.NotificationTypeAccountBanned.IsAdminOnly())
}

func TestNotificationType_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, shared.NotificationTypeNewFollower.IsValid())
	assert.True(t, shared.NotificationTypeAbuseReport.IsValid())
	assert.False(t, shared.NotificationType("invalid_type").IsValid())
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
	assert.Equal(t, "account_banned", shared.NotificationTypeAccountBanned.String())
}
