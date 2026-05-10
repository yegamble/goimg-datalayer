package shared

import (
	"testing"
)

func TestNotificationType_RequiresEmail(t *testing.T) {
	tests := []struct {
		name string
		nt   NotificationType
		want bool
	}{
		{"Account Suspended", NotificationTypeAccountSuspended, true},
		{"Account Banned", NotificationTypeAccountBanned, true},
		{"Account Reinstated", NotificationTypeAccountReinstated, true},
		{"Malware Detected", NotificationTypeMalwareDetected, true},
		{"New Follower", NotificationTypeNewFollower, false},
		{"New Photos", NotificationTypeNewPhotos, false},
		{"Abuse Report", NotificationTypeAbuseReport, false},
		{"Unknown", NotificationType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nt.RequiresEmail(); got != tt.want {
				t.Errorf("NotificationType.RequiresEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationType_IsAdminOnly(t *testing.T) {
	tests := []struct {
		name string
		nt   NotificationType
		want bool
	}{
		{"Abuse Report", NotificationTypeAbuseReport, true},
		{"Report Escalated", NotificationTypeReportEscalated, true},
		{"Mod Action Required", NotificationTypeModActionRequired, true},
		{"New Follower", NotificationTypeNewFollower, false},
		{"New Photos", NotificationTypeNewPhotos, false},
		{"Account Suspended", NotificationTypeAccountSuspended, false},
		{"Unknown", NotificationType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nt.IsAdminOnly(); got != tt.want {
				t.Errorf("NotificationType.IsAdminOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		nt   NotificationType
		want bool
	}{
		{"New Follower", NotificationTypeNewFollower, true},
		{"New Photos", NotificationTypeNewPhotos, true},
		{"Account Suspended", NotificationTypeAccountSuspended, true},
		{"Account Banned", NotificationTypeAccountBanned, true},
		{"Account Reinstated", NotificationTypeAccountReinstated, true},
		{"Abuse Report", NotificationTypeAbuseReport, true},
		{"Report Escalated", NotificationTypeReportEscalated, true},
		{"Mod Action Required", NotificationTypeModActionRequired, true},
		{"Malware Detected", NotificationTypeMalwareDetected, true},
		{"Unknown", NotificationType("unknown"), false},
		{"Empty", NotificationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nt.IsValid(); got != tt.want {
				t.Errorf("NotificationType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationType_String(t *testing.T) {
	nt := NotificationTypeNewFollower
	if got := nt.String(); got != "new_follower" {
		t.Errorf("NotificationType.String() = %v, want %v", got, "new_follower")
	}
}
