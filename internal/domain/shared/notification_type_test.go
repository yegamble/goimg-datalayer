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
		{"AccountSuspended requires email", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned requires email", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated requires email", shared.NotificationTypeAccountReinstated, true},
		{"MalwareDetected requires email", shared.NotificationTypeMalwareDetected, true},
		{"NewFollower does not require email", shared.NotificationTypeNewFollower, false},
		{"NewPhotos does not require email", shared.NotificationTypeNewPhotos, false},
		{"AbuseReport does not require email", shared.NotificationTypeAbuseReport, false},
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
		{"AbuseReport is admin only", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated is admin only", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired is admin only", shared.NotificationTypeModActionRequired, true},
		{"NewFollower is not admin only", shared.NotificationTypeNewFollower, false},
		{"AccountSuspended is not admin only", shared.NotificationTypeAccountSuspended, false},
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
		{"NewFollower is valid", shared.NotificationTypeNewFollower, true},
		{"NewPhotos is valid", shared.NotificationTypeNewPhotos, true},
		{"AccountSuspended is valid", shared.NotificationTypeAccountSuspended, true},
		{"AccountBanned is valid", shared.NotificationTypeAccountBanned, true},
		{"AccountReinstated is valid", shared.NotificationTypeAccountReinstated, true},
		{"AbuseReport is valid", shared.NotificationTypeAbuseReport, true},
		{"ReportEscalated is valid", shared.NotificationTypeReportEscalated, true},
		{"ModActionRequired is valid", shared.NotificationTypeModActionRequired, true},
		{"MalwareDetected is valid", shared.NotificationTypeMalwareDetected, true},
		{"Unknown string is invalid", shared.NotificationType("unknown_type"), false},
		{"Empty string is invalid", shared.NotificationType(""), false},
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
