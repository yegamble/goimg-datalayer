package notification_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID_New(t *testing.T) {
	t.Parallel()
	id := notification.NewNotificationID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestNotificationID_Parse(t *testing.T) {
	t.Parallel()

	t.Run("valid uuid", func(t *testing.T) {
		t.Parallel()
		uuidStr := uuid.New().String()
		id, err := notification.ParseNotificationID(uuidStr)
		require.NoError(t, err)
		assert.Equal(t, uuidStr, id.String())
		assert.False(t, id.IsZero())
	})

	t.Run("invalid uuid", func(t *testing.T) {
		t.Parallel()
		id, err := notification.ParseNotificationID("invalid-uuid")
		require.Error(t, err)
		assert.True(t, id.IsZero())
	})
}

func TestNotificationID_Equals(t *testing.T) {
	t.Parallel()
	id1 := notification.NewNotificationID()
	id2 := notification.NewNotificationID()
	id3, _ := notification.ParseNotificationID(id1.String())

	assert.True(t, id1.Equals(id3))
	assert.False(t, id1.Equals(id2))
}
