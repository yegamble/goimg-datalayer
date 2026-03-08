package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nt       shared.NotificationType
		expected bool
	}{
		{"AccountSuspended requires email", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned requires email", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated requires email", shared.NotificationTypeAccountReinstated, true},
		{"MalwareDetected requires email", shared.NotificationTypeMalwareDetected, true},
		{"NewFollower does not require email", shared.NotificationTypeNewFollower, false},
		{"NewPhotos does not require email", shared.NotificationTypeNewPhotos, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.nt.RequiresEmail())
		})
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nt       shared.NotificationType
		expected bool
	}{
		{"AbuseReport is admin only", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated is admin only", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired is admin only", shared.NotificationTypeModActionRequired, true},
		{"NewFollower is not admin only", shared.NotificationTypeNewFollower, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.nt.IsAdminOnly())
		})
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nt       shared.NotificationType
		expected bool
	}{
		{"Valid user notification", shared.NotificationTypeNewFollower, true},
		{"Valid admin notification", shared.NotificationTypeAbuseReport, true},
		{"Invalid notification", shared.NotificationType("invalid_type"), false},
		{"Empty notification", shared.NotificationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.nt.IsValid())
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	nt := shared.NotificationTypeNewFollower
	assert.Equal(t, "new_follower", nt.String())
}
