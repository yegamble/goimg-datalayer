package notification_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID(t *testing.T) {
	t.Run("NewNotificationID", func(t *testing.T) {
		id := notification.NewNotificationID()
		assert.False(t, id.IsZero())
	})

	t.Run("ParseNotificationID", func(t *testing.T) {
		id := notification.NewNotificationID()
		str := id.String()

		parsed, err := notification.ParseNotificationID(str)
		require.NoError(t, err)
		assert.True(t, id.Equals(parsed))

		_, err = notification.ParseNotificationID("invalid-uuid")
		require.Error(t, err)
	})

	t.Run("Equals", func(t *testing.T) {
		id1 := notification.NewNotificationID()
		id2 := notification.NewNotificationID()

		assert.True(t, id1.Equals(id1))
		assert.False(t, id1.Equals(id2))
	})
}
