package notification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNewNotification(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	notifType := shared.NotificationTypeNewFollower
	title := "Test Notification"
	body := "This is a test notification"
	metadata := map[string]string{"key": "value"}

	t.Run("Success", func(t *testing.T) {
		n, err := notification.NewNotification(recipientID, notification.NotificationType(notifType), title, body, metadata)
		require.NoError(t, err)
		assert.NotNil(t, n)
		assert.NotEmpty(t, n.ID())
		assert.Equal(t, recipientID, n.RecipientID())
		assert.Equal(t, notification.NotificationType(notifType), n.Type())
		assert.Equal(t, title, n.Title())
		assert.Equal(t, body, n.Body())
		assert.Equal(t, metadata, n.Metadata())
		assert.False(t, n.IsRead())
		assert.Nil(t, n.ReadAt())
		assert.WithinDuration(t, time.Now(), n.CreatedAt(), time.Second)
		assert.Empty(t, n.Events())
	})

	t.Run("Success_NilMetadata", func(t *testing.T) {
		n, err := notification.NewNotification(recipientID, notification.NotificationType(notifType), title, body, nil)
		require.NoError(t, err)
		assert.NotNil(t, n)
		assert.NotNil(t, n.Metadata()) // Should be initialized to empty map
		assert.Empty(t, n.Metadata())
	})

	t.Run("Error_MissingRecipient", func(t *testing.T) {
		_, err := notification.NewNotification(identity.UserID{}, notification.NotificationType(notifType), title, body, metadata)
		assert.ErrorIs(t, err, notification.ErrRecipientRequired)
	})

	t.Run("Error_MissingTitle", func(t *testing.T) {
		_, err := notification.NewNotification(recipientID, notification.NotificationType(notifType), "", body, metadata)
		assert.ErrorIs(t, err, notification.ErrTitleRequired)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		_, err := notification.NewNotification(recipientID, "invalid_type", title, body, metadata)
		assert.ErrorIs(t, err, notification.ErrInvalidNotificationType)
	})
}

func TestReconstructNotification(t *testing.T) {
	t.Parallel()

	id := notification.NewNotificationID()
	recipientID := identity.NewUserID()
	notifType := notification.NotificationType(shared.NotificationTypeAccountSuspended)
	title := "Reconstructed"
	body := "Body"
	metadata := map[string]string{"foo": "bar"}
	now := time.Now()
	readAt := &now

	n := notification.ReconstructNotification(id, recipientID, notifType, title, body, metadata, readAt, now)

	assert.Equal(t, id, n.ID())
	assert.Equal(t, recipientID, n.RecipientID())
	assert.Equal(t, notifType, n.Type())
	assert.Equal(t, title, n.Title())
	assert.Equal(t, body, n.Body())
	assert.Equal(t, metadata, n.Metadata())
	assert.Equal(t, readAt, n.ReadAt())
	assert.True(t, n.IsRead())
	assert.Equal(t, now, n.CreatedAt())
	assert.Empty(t, n.Events())

	// Test nil metadata initialization
	n2 := notification.ReconstructNotification(id, recipientID, notifType, title, body, nil, nil, now)
	assert.NotNil(t, n2.Metadata())
	assert.Empty(t, n2.Metadata())
}

func TestNotification_MarkRead(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, err := notification.NewNotification(recipientID, notification.NotificationType(shared.NotificationTypeNewFollower), "Title", "Body", nil)
	require.NoError(t, err)

	// Mark as read
	err = n.MarkRead()
	require.NoError(t, err)
	assert.True(t, n.IsRead())
	assert.NotNil(t, n.ReadAt())
	assert.WithinDuration(t, time.Now(), *n.ReadAt(), time.Second)

	// Mark read again (idempotent)
	firstReadAt := *n.ReadAt()
	time.Sleep(10 * time.Millisecond) // Ensure time passes
	err = n.MarkRead()
	require.NoError(t, err)
	assert.True(t, n.IsRead())
	assert.Equal(t, firstReadAt, *n.ReadAt()) // Timestamp should not change
}

func TestNotification_GetMetadata(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	metadata := map[string]string{"key1": "value1"}
	n, err := notification.NewNotification(recipientID, notification.NotificationType(shared.NotificationTypeNewFollower), "Title", "Body", metadata)
	require.NoError(t, err)

	assert.Equal(t, "value1", n.GetMetadata("key1"))
	assert.Equal(t, "", n.GetMetadata("nonexistent"))

	// Test with nil metadata (should not panic)
	// Although NewNotification initializes it, we check resilience.
	n2 := notification.ReconstructNotification(notification.NewNotificationID(), recipientID, notification.NotificationType(shared.NotificationTypeNewFollower), "T", "B", nil, nil, time.Now())
	assert.Equal(t, "", n2.GetMetadata("key"))
}

func TestNotification_Validate(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	notifType := notification.NotificationType(shared.NotificationTypeNewFollower)

	tests := []struct {
		name      string
		n         *notification.Notification
		wantErr   error
		errTarget error
	}{
		{
			name: "Valid",
			n: func() *notification.Notification {
				n, _ := notification.NewNotification(recipientID, notifType, "Title", "Body", nil)
				return n
			}(),
			wantErr: nil,
		},
		{
			name: "Missing Recipient",
			n: func() *notification.Notification {
				// Reconstruct allow bypassing NewNotification validation
				return notification.ReconstructNotification(notification.NewNotificationID(), identity.UserID{}, notifType, "Title", "Body", nil, nil, time.Now())
			}(),
			wantErr:   notification.ErrRecipientRequired,
			errTarget: notification.ErrRecipientRequired,
		},
		{
			name: "Missing Title",
			n: func() *notification.Notification {
				return notification.ReconstructNotification(notification.NewNotificationID(), recipientID, notifType, "", "Body", nil, nil, time.Now())
			}(),
			wantErr:   notification.ErrTitleRequired,
			errTarget: notification.ErrTitleRequired,
		},
		{
			name: "Invalid Type",
			n: func() *notification.Notification {
				return notification.ReconstructNotification(notification.NewNotificationID(), recipientID, "invalid", "Title", "Body", nil, nil, time.Now())
			}(),
			wantErr:   notification.ErrInvalidNotificationType,
			errTarget: notification.ErrInvalidNotificationType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.n.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.errTarget)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotification_Events(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := notification.NewNotification(recipientID, notification.NotificationType(shared.NotificationTypeNewFollower), "Title", "Body", nil)

	assert.Empty(t, n.Events())

	n.ClearEvents()
	assert.Empty(t, n.Events())
}
