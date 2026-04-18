package shared

import (
	"testing"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	if !NotificationTypeAccountSuspended.RequiresEmail() {
		t.Errorf("Expected true")
	}
	if NotificationTypeNewFollower.RequiresEmail() {
		t.Errorf("Expected false")
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	if !NotificationTypeAbuseReport.IsAdminOnly() {
		t.Errorf("Expected true")
	}
	if NotificationTypeNewFollower.IsAdminOnly() {
		t.Errorf("Expected false")
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	if !NotificationTypeNewFollower.IsValid() {
		t.Errorf("Expected true")
	}
	if NotificationType("invalid").IsValid() {
		t.Errorf("Expected false")
	}
}

func TestNotificationType_String(t *testing.T) {
	if NotificationTypeNewFollower.String() != "new_follower" {
		t.Errorf("Expected 'new_follower'")
	}
}
