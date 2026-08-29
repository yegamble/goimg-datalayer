package notification_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID_ParseNotificationID(t *testing.T) {
	id := notification.NewNotificationID()

	parsedID, err := notification.ParseNotificationID(id.String())
	require.NoError(t, err)
	assert.Equal(t, id, parsedID)

	_, err = notification.ParseNotificationID("invalid-uuid")
	require.Error(t, err)
}

func TestNotificationID_IsZero(t *testing.T) {
	id := notification.NewNotificationID()
	assert.False(t, id.IsZero())

	var zeroID notification.NotificationID
	assert.True(t, zeroID.IsZero())
}

func TestNotificationID_Equals(t *testing.T) {
	id1 := notification.NewNotificationID()
	id2 := notification.NewNotificationID()

	assert.True(t, id1.Equals(id1))
	assert.False(t, id1.Equals(id2))
}

func TestNotificationID_String(t *testing.T) {
	id := notification.NewNotificationID()
	assert.NotEmpty(t, id.String())
}
