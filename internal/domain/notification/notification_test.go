package notification_test

import (
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID(t *testing.T) {
	id := notification.NewNotificationID()

	if id.IsZero() {
		t.Errorf("NewNotificationID should not be zero")
	}

	str := id.String()
	if str == "" {
		t.Errorf("NotificationID.String() should not be empty")
	}

	parsedID, err := notification.ParseNotificationID(str)
	if err != nil {
		t.Errorf("ParseNotificationID failed: %v", err)
	}

	if !id.Equals(parsedID) {
		t.Errorf("Parsed ID should equal original ID")
	}

	parsedIDCopy := parsedID
	if !id.Equals(parsedIDCopy) {
		t.Errorf("ID should equal itself")
	}

	_, err = notification.ParseNotificationID("invalid-uuid")
	if err == nil {
		t.Errorf("ParseNotificationID should fail on invalid uuid")
	}
}

func TestNotification_Coverage(t *testing.T) {
	recipientID := identity.NewUserID()

	notif, err := notification.NewNotification(
		recipientID,
		notification.TypeNewFollower,
		"Test Title",
		"Test Body",
		map[string]string{"key": "value"},
	)

	if err != nil {
		t.Fatalf("Failed to create notification: %v", err)
	}

	// Validate accessors
	if notif.ID().IsZero() {
		t.Errorf("ID should not be zero")
	}
	if notif.RecipientID().IsZero() {
		t.Errorf("RecipientID should not be zero")
	}
	if notif.Type() != notification.TypeNewFollower {
		t.Errorf("Type mismatch")
	}
	if notif.Title() != "Test Title" {
		t.Errorf("Title mismatch")
	}
	if notif.Body() != "Test Body" {
		t.Errorf("Body mismatch")
	}
	if notif.ReadAt() != nil {
		t.Errorf("ReadAt should be nil initially")
	}
	if notif.CreatedAt().IsZero() {
		t.Errorf("CreatedAt should not be zero")
	}
	if notif.IsRead() {
		t.Errorf("IsRead should be false initially")
	}
	err = notif.MarkRead()
	if err != nil {
		t.Errorf("MarkRead should not return error")
	}
	if !notif.IsRead() {
		t.Errorf("IsRead should be true after MarkRead")
	}
	err = notif.MarkRead() // Call again for coverage of already read
	if err != nil {
		t.Errorf("MarkRead again should not return error")
	}

	raw := notif.MetadataRaw()
	if len(raw) == 0 {
		t.Errorf("MetadataRaw should return bytes")
	}

	if len(notif.Events()) != 0 {
		t.Errorf("New notification should have no events")
	}

	notif.ClearEvents()

	val := notif.GetMetadata("key")
	if val != "value" {
		t.Errorf("GetMetadata should return 'value'")
	}

	val2 := notif.GetMetadata("missing")
	if val2 != "" {
		t.Errorf("GetMetadata should return empty string for missing key")
	}

	err = notif.Validate()
	if err != nil {
		t.Errorf("Validate should return nil")
	}

	// Reconstruct with nil bytes to ensure GetMetadata handles nil meta
	id := notification.NewNotificationID()
	now := time.Now()

	reconstructed := notification.ReconstructNotification(
		id,
		recipientID,
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil, // passing nil for raw bytes to test fallback
		&now,
		now,
	)

	val3 := reconstructed.GetMetadata("key")
	if val3 != "" {
		t.Errorf("GetMetadata on nil metadata should return empty string")
	}

	// Test MetadataRaw fallbacks
	raw2 := reconstructed.MetadataRaw()
	if string(raw2) != "{}" {
		t.Errorf("MetadataRaw should return '{}' for nil map and bytes")
	}

	// Test MetadataRaw with pre-populated raw string
	rawNotif := notification.ReconstructNotification(
		id, recipientID, notification.TypeNewFollower, "T", "B", []byte(`{"k":"v"}`), nil, now,
	)
	if string(rawNotif.MetadataRaw()) != `{"k":"v"}` {
		t.Errorf("MetadataRaw should return underlying raw bytes")
	}

	// For coverage of MetadataRaw lazy parsing error fallback
	badJsonNotif := notification.ReconstructNotification(
		id, recipientID, notification.TypeNewFollower, "T", "B",
		[]byte(`{bad json`), nil, now,
	)
	_ = badJsonNotif.Metadata()

	// Test Validate errors
	badNotif1 := notification.ReconstructNotification(
		id, identity.UserID{}, notification.TypeNewFollower, "T", "B", nil, nil, now,
	)
	if badNotif1.Validate() == nil {
		t.Errorf("Validate should fail on missing recipient")
	}

	badNotif2 := notification.ReconstructNotification(
		id, recipientID, notification.TypeNewFollower, "", "B", nil, nil, now,
	)
	if badNotif2.Validate() == nil {
		t.Errorf("Validate should fail on missing title")
	}

	badNotif3 := notification.ReconstructNotification(
		id, recipientID, "", "T", "B", nil, nil, now,
	)
	if badNotif3.Validate() == nil {
		t.Errorf("Validate should fail on invalid type")
	}
}

func TestNotification_Validation(t *testing.T) {
	recipientID := identity.NewUserID()

	_, err := notification.NewNotification(
		identity.UserID{}, // Invalid recipient
		notification.TypeNewFollower,
		"Test Title",
		"Test Body",
		nil,
	)
	if err == nil {
		t.Errorf("Expected error for missing recipient")
	}

	_, err = notification.NewNotification(
		recipientID,
		notification.TypeNewFollower,
		"", // Invalid title
		"Test Body",
		nil,
	)
	if err == nil {
		t.Errorf("Expected error for missing title")
	}

	_, err = notification.NewNotification(
		recipientID,
		"", // Invalid type
		"Test Title",
		"Test Body",
		nil,
	)
	if err == nil {
		t.Errorf("Expected error for missing/invalid type")
	}

}
