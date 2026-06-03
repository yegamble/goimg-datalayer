package notification

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNotificationID(t *testing.T) {
	id := NewNotificationID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())

	id2, err := ParseNotificationID(id.String())
	assert.NoError(t, err)
	assert.True(t, id.Equals(id2))

	zeroID := NotificationID{}
	assert.True(t, zeroID.IsZero())
	assert.False(t, id.Equals(zeroID))

	_, err = ParseNotificationID("invalid")
	assert.Error(t, err)
}
