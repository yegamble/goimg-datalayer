package notification_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotification_MetadataRaw(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := notification.NewNotification(recipientID, notification.TypeNewFollower, "Title", "Body", nil)

	raw := n.MetadataRaw()
	assert.Equal(t, []byte("{}"), raw)
}

func TestNotification_MetadataRaw_Populated(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	meta := map[string]string{"key": "value"}
	n, _ := notification.NewNotification(recipientID, notification.TypeNewFollower, "Title", "Body", meta)

	raw := n.MetadataRaw()

	var unmarshaled map[string]string
	err := json.Unmarshal(raw, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, meta, unmarshaled)
}

func TestNotification_GetMetadata_NilMap(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		recipientID,
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil,
		nil,
		now,
	)

	assert.Equal(t, "", n.GetMetadata("missing"))
}

func TestNotification_ClearEvents(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := notification.NewNotification(recipientID, notification.TypeNewFollower, "Title", "Body", nil)

	n.ClearEvents()
	assert.Empty(t, n.Events())
}

func TestNotification_ReconstructNotification_EventsEmpty(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		recipientID,
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil,
		nil,
		now,
	)

	assert.Empty(t, n.Events())
}

func TestNotification_Validate(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		recipientID,
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil,
		nil,
		now,
	)

	assert.NoError(t, n.Validate())
}

func TestNotification_Validate_EmptyRecipient(t *testing.T) {
	t.Parallel()

	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		identity.UserID{},
		notification.TypeNewFollower,
		"Title",
		"Body",
		nil,
		nil,
		now,
	)

	assert.ErrorIs(t, n.Validate(), notification.ErrRecipientRequired)
}

func TestNotification_Validate_EmptyTitle(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		recipientID,
		notification.TypeNewFollower,
		"",
		"Body",
		nil,
		nil,
		now,
	)

	assert.ErrorIs(t, n.Validate(), notification.ErrTitleRequired)
}

func TestNotification_Validate_InvalidType(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	now := time.Now()

	n := notification.ReconstructNotification(
		notification.NewNotificationID(),
		recipientID,
		notification.NotificationType("invalid_type"),
		"Title",
		"Body",
		nil,
		nil,
		now,
	)

	assert.ErrorIs(t, n.Validate(), notification.ErrInvalidNotificationType)
}

func TestNotification_MarkRead(t *testing.T) {
	t.Parallel()
	recipientID := identity.NewUserID()
	n, _ := notification.NewNotification(recipientID, notification.TypeNewFollower, "Title", "Body", nil)

	assert.False(t, n.IsRead())
	assert.Nil(t, n.ReadAt())

	err := n.MarkRead()
	assert.NoError(t, err)
	assert.True(t, n.IsRead())
	assert.NotNil(t, n.ReadAt())

	// second call
	err = n.MarkRead()
	assert.NoError(t, err)
}

func TestNotificationID(t *testing.T) {
	t.Parallel()
	id := notification.NewNotificationID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())

	id2 := id
	assert.True(t, id.Equals(id2))

	str := id.String()
	parsed, err := notification.ParseNotificationID(str)
	assert.NoError(t, err)
	assert.True(t, id.Equals(parsed))
}
