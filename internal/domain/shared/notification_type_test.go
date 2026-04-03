package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ntype    NotificationType
		expected bool
	}{
		{
			name:     "account_suspended_requires_email",
			ntype:    NotificationTypeAccountSuspended,
			expected: true,
		},
		{
			name:     "account_banned_requires_email",
			ntype:    NotificationTypeAccountBanned,
			expected: true,
		},
		{
			name:     "account_reinstated_requires_email",
			ntype:    NotificationTypeAccountReinstated,
			expected: true,
		},
		{
			name:     "malware_detected_requires_email",
			ntype:    NotificationTypeMalwareDetected,
			expected: true,
		},
		{
			name:     "new_follower_does_not_require_email",
			ntype:    NotificationTypeNewFollower,
			expected: false,
		},
		{
			name:     "new_photos_does_not_require_email",
			ntype:    NotificationTypeNewPhotos,
			expected: false,
		},
		{
			name:     "abuse_report_does_not_require_email",
			ntype:    NotificationTypeAbuseReport,
			expected: false,
		},
		{
			name:     "unknown_does_not_require_email",
			ntype:    NotificationType("unknown_type"),
			expected: false,
		},
	}

	for _, tt := range tests {
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
		ntype    NotificationType
		expected bool
	}{
		{
			name:     "abuse_report_is_admin_only",
			ntype:    NotificationTypeAbuseReport,
			expected: true,
		},
		{
			name:     "report_escalated_is_admin_only",
			ntype:    NotificationTypeReportEscalated,
			expected: true,
		},
		{
			name:     "mod_action_required_is_admin_only",
			ntype:    NotificationTypeModActionRequired,
			expected: true,
		},
		{
			name:     "new_follower_is_not_admin_only",
			ntype:    NotificationTypeNewFollower,
			expected: false,
		},
		{
			name:     "account_suspended_is_not_admin_only",
			ntype:    NotificationTypeAccountSuspended,
			expected: false,
		},
		{
			name:     "unknown_is_not_admin_only",
			ntype:    NotificationType("unknown_type"),
			expected: false,
		},
	}

	for _, tt := range tests {
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
		ntype    NotificationType
		expected bool
	}{
		{
			name:     "new_follower_is_valid",
			ntype:    NotificationTypeNewFollower,
			expected: true,
		},
		{
			name:     "new_photos_is_valid",
			ntype:    NotificationTypeNewPhotos,
			expected: true,
		},
		{
			name:     "account_suspended_is_valid",
			ntype:    NotificationTypeAccountSuspended,
			expected: true,
		},
		{
			name:     "account_banned_is_valid",
			ntype:    NotificationTypeAccountBanned,
			expected: true,
		},
		{
			name:     "account_reinstated_is_valid",
			ntype:    NotificationTypeAccountReinstated,
			expected: true,
		},
		{
			name:     "malware_detected_is_valid",
			ntype:    NotificationTypeMalwareDetected,
			expected: true,
		},
		{
			name:     "abuse_report_is_valid",
			ntype:    NotificationTypeAbuseReport,
			expected: true,
		},
		{
			name:     "report_escalated_is_valid",
			ntype:    NotificationTypeReportEscalated,
			expected: true,
		},
		{
			name:     "mod_action_required_is_valid",
			ntype:    NotificationTypeModActionRequired,
			expected: true,
		},
		{
			name:     "unknown_type_is_invalid",
			ntype:    NotificationType("unknown_type"),
			expected: false,
		},
		{
			name:     "empty_type_is_invalid",
			ntype:    NotificationType(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ntype.IsValid())
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	ntype := NotificationTypeNewFollower
	assert.Equal(t, "new_follower", ntype.String())
}
