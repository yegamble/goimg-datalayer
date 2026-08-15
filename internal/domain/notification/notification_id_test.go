package notification

import (
	"testing"
)

func TestNotificationID(t *testing.T) {
	id1 := NewNotificationID()
	id2 := NewNotificationID()

	if id1.Equals(id2) {
		t.Errorf("Expected IDs to be different")
	}

	if id1.IsZero() {
		t.Errorf("Expected ID not to be zero")
	}

	if id1.String() == "" {
		t.Errorf("Expected ID string not to be empty")
	}

	id3, err := ParseNotificationID(id1.String())
	if err != nil {
		t.Errorf("ParseNotificationID failed: %v", err)
	}

	if !id1.Equals(id3) {
		t.Errorf("Expected parsed ID to equal original ID")
	}

	_, err = ParseNotificationID("invalid")
	if err == nil {
		t.Errorf("Expected ParseNotificationID to fail for invalid string")
	}
}
