package notification

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationID(t *testing.T) {
	t.Parallel()

	t.Run("NewNotificationID", func(t *testing.T) {
		t.Parallel()
		id := NewNotificationID()
		assert.False(t, id.IsZero())
		assert.NotEqual(t, uuid.Nil.String(), id.String())
	})

	t.Run("ParseNotificationID valid", func(t *testing.T) {
		t.Parallel()
		idStr := uuid.New().String()
		id, err := ParseNotificationID(idStr)
		require.NoError(t, err)
		assert.Equal(t, idStr, id.String())
	})

	t.Run("ParseNotificationID invalid", func(t *testing.T) {
		t.Parallel()
		id, err := ParseNotificationID("invalid-uuid")
		require.Error(t, err)
		assert.True(t, id.IsZero())
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		idStr := uuid.New().String()
		id, _ := ParseNotificationID(idStr)
		assert.Equal(t, idStr, id.String())
	})

	t.Run("IsZero", func(t *testing.T) {
		t.Parallel()
		id1 := NotificationID{}
		assert.True(t, id1.IsZero())
		id2 := NewNotificationID()
		assert.False(t, id2.IsZero())
	})

	t.Run("Equals", func(t *testing.T) {
		t.Parallel()
		id1 := NewNotificationID()
		id1Copy, _ := ParseNotificationID(id1.String())
		id2 := NewNotificationID()
		assert.True(t, id1.Equals(id1Copy))
		assert.False(t, id1.Equals(id2))
	})
}
