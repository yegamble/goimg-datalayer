package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	tests := []struct {
		name string
		nt   shared.NotificationType
		want bool
	}{
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated", shared.NotificationTypeAccountReinstated, true},
		{"MalwareDetected", shared.NotificationTypeMalwareDetected, true},
		{"NewFollower", shared.NotificationTypeNewFollower, false},
		{"NewPhotos", shared.NotificationTypeNewPhotos, false},
		{"AbuseReport", shared.NotificationTypeAbuseReport, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.nt.RequiresEmail())
		})
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	tests := []struct {
		name string
		nt   shared.NotificationType
		want bool
	}{
		{"AbuseReport", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired", shared.NotificationTypeModActionRequired, true},
		{"NewFollower", shared.NotificationTypeNewFollower, false},
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.nt.IsAdminOnly())
		})
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		nt   shared.NotificationType
		want bool
	}{
		{"NewFollower", shared.NotificationTypeNewFollower, true},
		{"NewPhotos", shared.NotificationTypeNewPhotos, true},
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, true},
		{"AbuseReport", shared.NotificationTypeAbuseReport, true},
		{"Invalid", shared.NotificationType("invalid_type"), false},
		{"Empty", shared.NotificationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.nt.IsValid())
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	nt := shared.NotificationTypeNewFollower
	assert.Equal(t, "new_follower", nt.String())
}
