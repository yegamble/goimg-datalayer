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
		{
			name:     "AccountSuspended requires email",
			ntype:    shared.NotificationTypeAccountSuspended,
			expected: true,
		},
		{
			name:     "AccountBanned requires email",
			ntype:    shared.NotificationTypeAccountBanned,
			expected: true,
		},
		{
			name:     "AccountReinstated requires email",
			ntype:    shared.NotificationTypeAccountReinstated,
			expected: true,
		},
		{
			name:     "MalwareDetected requires email",
			ntype:    shared.NotificationTypeMalwareDetected,
			expected: true,
		},
		{
			name:     "NewFollower does not require email",
			ntype:    shared.NotificationTypeNewFollower,
			expected: false,
		},
		{
			name:     "AbuseReport does not require email",
			ntype:    shared.NotificationTypeAbuseReport,
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.ntype.RequiresEmail())
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
		{
			name:     "AbuseReport is admin only",
			ntype:    shared.NotificationTypeAbuseReport,
			expected: true,
		},
		{
			name:     "ReportEscalated is admin only",
			ntype:    shared.NotificationTypeReportEscalated,
			expected: true,
		},
		{
			name:     "ModActionRequired is admin only",
			ntype:    shared.NotificationTypeModActionRequired,
			expected: true,
		},
		{
			name:     "NewFollower is not admin only",
			ntype:    shared.NotificationTypeNewFollower,
			expected: false,
		},
		{
			name:     "AccountBanned is not admin only",
			ntype:    shared.NotificationTypeAccountBanned,
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.ntype.IsAdminOnly())
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
		{
			name:     "NewFollower is valid",
			ntype:    shared.NotificationTypeNewFollower,
			expected: true,
		},
		{
			name:     "NewPhotos is valid",
			ntype:    shared.NotificationTypeNewPhotos,
			expected: true,
		},
		{
			name:     "AccountSuspended is valid",
			ntype:    shared.NotificationTypeAccountSuspended,
			expected: true,
		},
		{
			name:     "AccountBanned is valid",
			ntype:    shared.NotificationTypeAccountBanned,
			expected: true,
		},
		{
			name:     "AccountReinstated is valid",
			ntype:    shared.NotificationTypeAccountReinstated,
			expected: true,
		},
		{
			name:     "AbuseReport is valid",
			ntype:    shared.NotificationTypeAbuseReport,
			expected: true,
		},
		{
			name:     "ReportEscalated is valid",
			ntype:    shared.NotificationTypeReportEscalated,
			expected: true,
		},
		{
			name:     "ModActionRequired is valid",
			ntype:    shared.NotificationTypeModActionRequired,
			expected: true,
		},
		{
			name:     "MalwareDetected is valid",
			ntype:    shared.NotificationTypeMalwareDetected,
			expected: true,
		},
		{
			name:     "Invalid type",
			ntype:    shared.NotificationType("invalid_type"),
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.ntype.IsValid())
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	ntype := shared.NotificationTypeNewFollower
	assert.Equal(t, "new_follower", ntype.String())
}
