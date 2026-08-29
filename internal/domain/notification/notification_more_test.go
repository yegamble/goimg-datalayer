package notification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestReconstructNotification(t *testing.T) {
	id := notification.NewNotificationID()
	recipientID := identity.NewUserID()
	createdAt := time.Now()

	n := notification.ReconstructNotification(
		id,
		recipientID,
		shared.NotificationTypeNewFollower,
		"title",
		"body",
		[]byte(`{"key":"value"}`),
		nil,
		createdAt,
	)

	assert.Equal(t, id, n.ID())
	assert.Equal(t, recipientID, n.RecipientID())
	assert.Equal(t, shared.NotificationTypeNewFollower, n.Type())
	assert.Equal(t, "title", n.Title())
	assert.Equal(t, "body", n.Body())
	assert.Nil(t, n.ReadAt())
	assert.Equal(t, createdAt, n.CreatedAt())
	assert.False(t, n.IsRead())

    meta := n.Metadata()
    assert.Equal(t, "value", meta["key"])

    val := n.GetMetadata("key")
    assert.Equal(t, "value", val)

    valMissing := n.GetMetadata("missing")
    assert.Empty(t, valMissing)

    err := n.MarkRead()
    assert.NoError(t, err)
    err = n.MarkRead()
    assert.NoError(t, err)

    n2, err := notification.NewNotification(recipientID, shared.NotificationTypeNewFollower, "title", "body", map[string]string{"key": "value"})
    assert.NoError(t, err)
    err = n2.Validate()
    assert.NoError(t, err)

    _, err = notification.NewNotification(identity.UserID{}, shared.NotificationTypeNewFollower, "title", "body", nil)
    assert.ErrorIs(t, err, notification.ErrRecipientRequired)

    _, err = notification.NewNotification(recipientID, shared.NotificationType("invalid"), "title", "body", nil)
    assert.ErrorIs(t, err, notification.ErrInvalidNotificationType)

    _, err = notification.NewNotification(recipientID, shared.NotificationTypeNewFollower, "", "body", nil)
    assert.ErrorIs(t, err, notification.ErrTitleRequired)

    err = n2.Validate()
    assert.NoError(t, err)

	nid := notification.NewNotificationID()
	nid2, _ := notification.ParseNotificationID(nid.String())
	assert.True(t, nid.Equals(nid2))
	assert.False(t, nid.IsZero())
	assert.NotEmpty(t, nid.String())

	_, err = notification.ParseNotificationID("invalid")
	assert.Error(t, err)
}
