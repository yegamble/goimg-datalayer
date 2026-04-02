package notification_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID_Parse(t *testing.T) {
	t.Parallel()

	id := notification.NewNotificationID()

	parsed, err := notification.ParseNotificationID(id.String())
	if err != nil {
		t.Fatalf("unexpected error parsing ID: %v", err)
	}

	if !id.Equals(parsed) {
		t.Errorf("expected parsed ID to equal original ID")
	}
}

func TestNotificationID_IsZero(t *testing.T) {
	t.Parallel()

	var zeroID notification.NotificationID
	if !zeroID.IsZero() {
		t.Errorf("expected zero ID to be zero")
	}

	id := notification.NewNotificationID()
	if id.IsZero() {
		t.Errorf("expected new ID to not be zero")
	}
}
