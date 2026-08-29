package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID_ParseNotificationID(t *testing.T) {
	idStr := uuid.New().String()
	id, err := notification.ParseNotificationID(idStr)
	assert.NoError(t, err)
	assert.Equal(t, idStr, id.String())

	_, err = notification.ParseNotificationID("invalid")
	assert.Error(t, err)
}

func TestNotificationID_IsZero(t *testing.T) {
	var id notification.NotificationID
	assert.True(t, id.IsZero())

	id = notification.NewNotificationID()
	assert.False(t, id.IsZero())
}

func TestNotificationID_Equals(t *testing.T) {
	id1 := notification.NewNotificationID()
	id2 := notification.NewNotificationID()
	id3, _ := notification.ParseNotificationID(id1.String())

	assert.True(t, id1.Equals(id3))
	assert.False(t, id1.Equals(id2))
}
