package shared_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType(t *testing.T) {
	t.Parallel()

	types := []shared.NotificationType{
		shared.NotificationTypeNewFollower,
		shared.NotificationTypeNewPhotos,
		shared.NotificationTypeAccountSuspended,
		shared.NotificationTypeAccountBanned,
		shared.NotificationTypeAccountReinstated,
		shared.NotificationTypeMalwareDetected,
		shared.NotificationTypeAbuseReport,
		shared.NotificationTypeReportEscalated,
		shared.NotificationTypeModActionRequired,
	}

	for _, nt := range types {
		if !nt.IsValid() {
			t.Errorf("expected type %v to be valid", nt)
		}
		if nt.String() == "" {
			t.Errorf("expected string representation to not be empty for type %v", nt)
		}

		// Test methods that shouldn't panic
		_ = nt.RequiresEmail()
		_ = nt.IsAdminOnly()
	}

	invalidType := shared.NotificationType("invalid_type")
	if invalidType.IsValid() {
		t.Errorf("expected invalid type to be invalid")
	}
}
