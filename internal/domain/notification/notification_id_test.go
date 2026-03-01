package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID(t *testing.T) {
	t.Run("NewNotificationID creates valid ID", func(t *testing.T) {
		id := notification.NewNotificationID()
		assert.False(t, id.IsZero())
		assert.NotEmpty(t, id.String())
	})

	t.Run("ParseNotificationID handles valid UUID", func(t *testing.T) {
		str := uuid.New().String()
		id, err := notification.ParseNotificationID(str)
		assert.NoError(t, err)
		assert.Equal(t, str, id.String())
	})

	t.Run("ParseNotificationID handles invalid UUID", func(t *testing.T) {
		_, err := notification.ParseNotificationID("invalid")
		assert.Error(t, err)
	})

	t.Run("Equals compares correctly", func(t *testing.T) {
		id1 := notification.NewNotificationID()
		id2 := notification.NewNotificationID()

		assert.True(t, id1.Equals(id1))
		assert.False(t, id1.Equals(id2))
	})
}
