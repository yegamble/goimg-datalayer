package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID(t *testing.T) {
	t.Parallel()

	t.Run("NewNotificationID", func(t *testing.T) {
		t.Parallel()
		id := notification.NewNotificationID()
		assert.False(t, id.IsZero())
	})

	t.Run("ParseNotificationID", func(t *testing.T) {
		t.Parallel()
		idStr := uuid.New().String()
		id, err := notification.ParseNotificationID(idStr)
		assert.NoError(t, err)
		assert.Equal(t, idStr, id.String())
	})

	t.Run("ParseNotificationID Error", func(t *testing.T) {
		t.Parallel()
		_, err := notification.ParseNotificationID("invalid")
		assert.Error(t, err)
	})

	t.Run("Equals", func(t *testing.T) {
		t.Parallel()
		id1 := notification.NewNotificationID()
		id2 := id1
		id3 := notification.NewNotificationID()

		assert.True(t, id1.Equals(id2))
		assert.False(t, id1.Equals(id3))
	})
}
