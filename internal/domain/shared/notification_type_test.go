package shared_test

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"testing"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	if !shared.NotificationTypeAccountSuspended.RequiresEmail() {
		t.Errorf("AccountSuspended should require email")
	}
	if shared.NotificationTypeNewFollower.RequiresEmail() {
		t.Errorf("NewFollower should not require email")
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	if !shared.NotificationTypeAbuseReport.IsAdminOnly() {
		t.Errorf("AbuseReport should be admin only")
	}
	if shared.NotificationTypeNewFollower.IsAdminOnly() {
		t.Errorf("NewFollower should not be admin only")
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	if !shared.NotificationTypeNewFollower.IsValid() {
		t.Errorf("NewFollower should be valid")
	}
	if shared.NotificationType("invalid").IsValid() {
		t.Errorf("invalid should not be valid")
	}
}

func TestNotificationType_String(t *testing.T) {
	if shared.NotificationTypeNewFollower.String() != string(shared.NotificationTypeNewFollower) {
		t.Errorf("String method should return underlying string")
	}
}
