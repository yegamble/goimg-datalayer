package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ntype    shared.NotificationType
		expected bool
	}{
		{shared.NotificationTypeAccountSuspended, true},
		{shared.NotificationTypeAccountBanned, true},
		{shared.NotificationTypeAccountReinstated, true},
		{shared.NotificationTypeMalwareDetected, true},
		{shared.NotificationTypeNewFollower, false},
		{shared.NotificationTypeNewPhotos, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.ntype), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ntype.RequiresEmail())
		})
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ntype    shared.NotificationType
		expected bool
	}{
		{shared.NotificationTypeAbuseReport, true},
		{shared.NotificationTypeReportEscalated, true},
		{shared.NotificationTypeModActionRequired, true},
		{shared.NotificationTypeNewFollower, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.ntype), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ntype.IsAdminOnly())
		})
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, shared.NotificationTypeNewFollower.IsValid())
	assert.True(t, shared.NotificationTypeModActionRequired.IsValid())
	assert.False(t, shared.NotificationType("invalid").IsValid())
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "new_follower", shared.NotificationTypeNewFollower.String())
	assert.Equal(t, "invalid", shared.NotificationType("invalid").String())
}
