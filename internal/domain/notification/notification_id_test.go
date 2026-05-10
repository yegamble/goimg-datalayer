package notification

import (
	"testing"
)

func TestNotificationID(t *testing.T) {
	// Test NewNotificationID
	id := NewNotificationID()
	if id.IsZero() {
		t.Error("NewNotificationID() generated a zero ID")
	}

	// Test String
	str := id.String()
	if str == "" || str == "00000000-0000-0000-0000-000000000000" {
		t.Errorf("NotificationID.String() returned invalid string: %s", str)
	}

	// Test ParseNotificationID (valid)
	parsed, err := ParseNotificationID(str)
	if err != nil {
		t.Fatalf("ParseNotificationID() unexpected error: %v", err)
	}
	if parsed != id {
		t.Errorf("ParseNotificationID() = %v, want %v", parsed, id)
	}

	// Test ParseNotificationID (invalid)
	_, err = ParseNotificationID("invalid-uuid")
	if err == nil {
		t.Error("ParseNotificationID() expected error for invalid UUID, got nil")
	}

	// Test IsZero
	zeroID := NotificationID{}
	if !zeroID.IsZero() {
		t.Error("NotificationID{}.IsZero() = false, want true")
	}
	if id.IsZero() {
		t.Error("id.IsZero() = true, want false")
	}

	// Test Equals
	idCopy := id
	if !id.Equals(idCopy) {
		t.Error("NotificationID.Equals() should be true for identical IDs")
	}
	newID := NewNotificationID()
	if id.Equals(newID) {
		t.Error("NotificationID.Equals() should be false for different IDs")
	}
}
