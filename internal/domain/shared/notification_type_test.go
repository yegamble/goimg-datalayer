package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType(t *testing.T) {
	tests := []struct {
		name          string
		ntype         shared.NotificationType
		requiresEmail bool
		isAdminOnly   bool
		isValid       bool
		str           string
	}{
		{"NewFollower", shared.NotificationTypeNewFollower, false, false, true, "new_follower"},
		{"NewPhotos", shared.NotificationTypeNewPhotos, false, false, true, "new_photos"},
		{"AccountSuspended", shared.NotificationTypeAccountSuspended, true, false, true, "account_suspended"},
		{"AccountBanned", shared.NotificationTypeAccountBanned, true, false, true, "account_banned"},
		{"AccountReinstated", shared.NotificationTypeAccountReinstated, true, false, true, "account_reinstated"},
		{"MalwareDetected", shared.NotificationTypeMalwareDetected, true, false, true, "malware_detected"},
		{"AbuseReport", shared.NotificationTypeAbuseReport, false, true, true, "abuse_report"},
		{"ReportEscalated", shared.NotificationTypeReportEscalated, false, true, true, "report_escalated"},
		{"ModActionRequired", shared.NotificationTypeModActionRequired, false, true, true, "mod_action_required"},
		{"Invalid", shared.NotificationType("invalid_type"), false, false, false, "invalid_type"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.requiresEmail, tc.ntype.RequiresEmail())
			assert.Equal(t, tc.isAdminOnly, tc.ntype.IsAdminOnly())
			assert.Equal(t, tc.isValid, tc.ntype.IsValid())
			assert.Equal(t, tc.str, tc.ntype.String())
		})
	}
}
