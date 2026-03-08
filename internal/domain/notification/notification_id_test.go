package notification_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotificationID(t *testing.T) {
	id := notification.NewNotificationID()
	assert.NotEmpty(t, id.String())
	assert.False(t, id.IsZero())

	id2, err := notification.ParseNotificationID(id.String())
	assert.NoError(t, err)
	assert.True(t, id.Equals(id2))
}
