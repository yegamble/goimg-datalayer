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
		ntype    shared.NotificationType
		expected bool
	}{
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated", shared.NotificationTypeAccountReinstated, true},
		{"MalwareDetected", shared.NotificationTypeMalwareDetected, true},
		{"NewFollower", shared.NotificationTypeNewFollower, false},
		{"Invalid", shared.NotificationType("invalid"), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ntype.RequiresEmail())
		})
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ntype    shared.NotificationType
		expected bool
	}{
		{"AbuseReport", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired", shared.NotificationTypeModActionRequired, true},
		{"NewFollower", shared.NotificationTypeNewFollower, false},
		{"Invalid", shared.NotificationType("invalid"), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ntype.IsAdminOnly())
		})
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ntype    shared.NotificationType
		expected bool
	}{
		{"NewFollower", shared.NotificationTypeNewFollower, true},
		{"NewPhotos", shared.NotificationTypeNewPhotos, true},
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated", shared.NotificationTypeAccountReinstated, true},
		{"AbuseReport", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired", shared.NotificationTypeModActionRequired, true},
		{"MalwareDetected", shared.NotificationTypeMalwareDetected, true},
		{"Invalid", shared.NotificationType("invalid"), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ntype.IsValid())
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	ntype := shared.NotificationTypeNewFollower
	assert.Equal(t, "new_follower", ntype.String())
}
