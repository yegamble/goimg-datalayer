package notification

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationID(t *testing.T) {
	t.Run("NewNotificationID creates valid ID", func(t *testing.T) {
		id := NewNotificationID()
		assert.NotEqual(t, uuid.Nil.String(), id.String())
		assert.False(t, id.IsZero())
	})

	t.Run("ParseNotificationID", func(t *testing.T) {
		validStr := uuid.New().String()
		id, err := ParseNotificationID(validStr)
		require.NoError(t, err)
		assert.Equal(t, validStr, id.String())
		assert.False(t, id.IsZero())

		_, err = ParseNotificationID("invalid")
		require.Error(t, err)
	})

	t.Run("Equals", func(t *testing.T) {
		id1 := NewNotificationID()
		id2 := NewNotificationID()

		id1Copy, _ := ParseNotificationID(id1.String())

		assert.True(t, id1.Equals(id1Copy))
		assert.False(t, id1.Equals(id2))
	})
}
