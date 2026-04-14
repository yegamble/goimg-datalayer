package notification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotification(t *testing.T) {
	userID := identity.NewUserID()
	notifType := notification.TypeNewFollower
	title := "Test Notification"
	body := "This is a test notification"
	metadata := map[string]string{"key": "value"}

	t.Run("NewNotification creates valid notification", func(t *testing.T) {
		notif, err := notification.NewNotification(
			userID,
			notifType,
			title,
			body,
			metadata,
		)
		require.NoError(t, err)

		assert.False(t, notif.ID().IsZero())
		assert.Equal(t, userID, notif.RecipientID())
		assert.Equal(t, notifType, notif.Type())
		assert.Equal(t, title, notif.Title())
		assert.Equal(t, body, notif.Body())
		assert.Equal(t, metadata, notif.Metadata())
		assert.False(t, notif.IsRead())
		assert.False(t, notif.CreatedAt().IsZero())
	})

	t.Run("MarkRead marks notification as read", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			userID,
			notifType,
			title,
			body,
			metadata,
		)

		err := notif.MarkRead()
		assert.NoError(t, err)
		assert.True(t, notif.IsRead())

		err = notif.MarkRead()
		assert.NoError(t, err) // Idempotent
	})

	t.Run("Reconstruct returns correctly assembled entity", func(t *testing.T) {
		id := notification.NewNotificationID()
		now := time.Now()
		readAt := time.Now()
		notif := notification.ReconstructNotification(
			id,
			userID,
			notifType,
			title,
			body,
			[]byte(`{"key":"value"}`),
			&readAt,
			now,
		)

		assert.Equal(t, id, notif.ID())
		assert.Equal(t, userID, notif.RecipientID())
		assert.Equal(t, notifType, notif.Type())
		assert.Equal(t, title, notif.Title())
		assert.Equal(t, body, notif.Body())
		assert.Equal(t, metadata, notif.Metadata())
		assert.True(t, notif.IsRead())
		assert.Equal(t, now, notif.CreatedAt())
	})


	t.Run("NewNotification handles invalid inputs", func(t *testing.T) {
		_, err := notification.NewNotification(
			identity.UserID{},
			notifType,
			title,
			body,
			metadata,
		)
		assert.ErrorIs(t, err, notification.ErrRecipientRequired)

		_, err = notification.NewNotification(
			userID,
			notifType,
			"",
			body,
			metadata,
		)
		assert.ErrorIs(t, err, notification.ErrTitleRequired)

		_, err = notification.NewNotification(
			userID,
			"invalid",
			title,
			body,
			metadata,
		)
		assert.ErrorIs(t, err, notification.ErrInvalidNotificationType)
	})

	t.Run("NewNotification handles nil metadata", func(t *testing.T) {
		notif, err := notification.NewNotification(
			userID,
			notifType,
			title,
			body,
			nil,
		)
		assert.NoError(t, err)
		assert.NotNil(t, notif.Metadata())
	})

	t.Run("Notification metadata methods", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			userID,
			notifType,
			title,
			body,
			metadata,
		)
		assert.Equal(t, "value", notif.GetMetadata("key"))
		assert.Equal(t, "", notif.GetMetadata("non-existent"))
		assert.NotNil(t, notif.MetadataRaw())
	})

	t.Run("Reconstruct handles nil/empty raw metadata safely", func(t *testing.T) {
		id := notification.NewNotificationID()
		now := time.Now()

		notif1 := notification.ReconstructNotification(
			id, userID, notifType, title, body, nil, nil, now,
		)
		assert.NotNil(t, notif1.Metadata())

		notif2 := notification.ReconstructNotification(
			id, userID, notifType, title, body, []byte{}, nil, now,
		)
		assert.NotNil(t, notif2.Metadata())
	})

	t.Run("Events return empty array and can be cleared", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			userID, notifType, title, body, metadata,
		)
		assert.Empty(t, notif.Events())
		notif.ClearEvents()
		assert.Empty(t, notif.Events())
	})


	t.Run("MetadataRaw returns correct bytes", func(t *testing.T) {
		id := notification.NewNotificationID()
		now := time.Now()

		// Test when metadata map is populated
		notif1 := notification.ReconstructNotification(
			id, userID, notifType, title, body, nil, nil, now,
		)
		bytes1 := notif1.MetadataRaw()
		assert.NotNil(t, bytes1)

		// Test when metadataRaw is available but map is nil
		notif2 := notification.ReconstructNotification(
			id, userID, notifType, title, body, []byte(`{"raw":"data"}`), nil, now,
		)
		// Access MetadataRaw directly before accessing Metadata()
		bytes2 := notif2.MetadataRaw()
		assert.Equal(t, []byte(`{"raw":"data"}`), bytes2)

		// Now access Metadata() to trigger unmarshal error path with bad JSON
		notif3 := notification.ReconstructNotification(
			id, userID, notifType, title, body, []byte(`{bad json`), nil, now,
		)
		meta3 := notif3.Metadata()
		assert.Empty(t, meta3)
	})

	t.Run("Validation handles missing fields", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			userID, notifType, title, body, metadata,
		)
		assert.NoError(t, notif.Validate())

		id := notification.NewNotificationID()
		now := time.Now()

		badNotif := notification.ReconstructNotification(
			id, identity.UserID{}, notifType, title, body, nil, nil, now,
		)
		assert.ErrorIs(t, badNotif.Validate(), notification.ErrRecipientRequired)

		badNotif = notification.ReconstructNotification(
			id, userID, notifType, "", body, nil, nil, now,
		)
		assert.ErrorIs(t, badNotif.Validate(), notification.ErrTitleRequired)

		badNotif = notification.ReconstructNotification(
			id, userID, "invalid", title, body, nil, nil, now,
		)
		assert.ErrorIs(t, badNotif.Validate(), notification.ErrInvalidNotificationType)
	})

}
