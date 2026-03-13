package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID_NewNotificationID(t *testing.T) {
	t.Parallel()

	id := notification.NewNotificationID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestNotificationID_ParseNotificationID(t *testing.T) {
	t.Parallel()

	t.Run("Valid UUID", func(t *testing.T) {
		t.Parallel()

		str := uuid.New().String()
		id, err := notification.ParseNotificationID(str)
		require.NoError(t, err)
		assert.Equal(t, str, id.String())
		assert.False(t, id.IsZero())
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		t.Parallel()

		_, err := notification.ParseNotificationID("invalid-uuid")
		require.Error(t, err)
	})
}

func TestNotificationID_String(t *testing.T) {
	t.Parallel()

	str := uuid.New().String()
	id, err := notification.ParseNotificationID(str)
	require.NoError(t, err)
	assert.Equal(t, str, id.String())
}

func TestNotificationID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("Zero ID", func(t *testing.T) {
		t.Parallel()
		var id notification.NotificationID
		assert.True(t, id.IsZero())
	})

	t.Run("Non-zero ID", func(t *testing.T) {
		t.Parallel()
		id := notification.NewNotificationID()
		assert.False(t, id.IsZero())
	})
}

func TestNotificationID_Equals(t *testing.T) {
	t.Parallel()

	id1 := notification.NewNotificationID()
	id2, err := notification.ParseNotificationID(id1.String())
	require.NoError(t, err)
	id3 := notification.NewNotificationID()

	assert.True(t, id1.Equals(id2))
	assert.True(t, id2.Equals(id1))
	assert.False(t, id1.Equals(id3))
}
