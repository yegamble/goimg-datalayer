package notification_test

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"testing"
)

func TestNotificationID_ParseNotificationID(t *testing.T) {
	idStr := notification.NewNotificationID().String()
	id, err := notification.ParseNotificationID(idStr)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if id.String() != idStr {
		t.Fatalf("expected %s, got %s", idStr, id.String())
	}

	if id.IsZero() {
		t.Fatalf("expected id to not be zero")
	}

	if !id.Equals(id) {
		t.Fatalf("expected id to be equal to itself")
	}
}
