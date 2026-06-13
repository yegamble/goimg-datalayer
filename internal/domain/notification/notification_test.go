package notification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func TestNotification(t *testing.T) {
	recipientID := identity.NewUserID()

	n, err := notification.NewNotification(
		recipientID,
		notification.TypeNewFollower,
		"Test Title",
		"Test Body",
		map[string]string{"foo": "bar"},
	)

	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.False(t, n.IsRead())

	err = n.MarkRead()
	require.NoError(t, err)
	assert.True(t, n.IsRead())

	err = n.MarkRead()
	assert.NoError(t, err) // It's a no-op!

	now := time.Now()
	n2 := notification.ReconstructNotification(
		n.ID(),
		n.RecipientID(),
		n.Type(),
		n.Title(),
		n.Body(),
		nil,
		&now,
		n.CreatedAt(),
	)
	assert.True(t, n2.IsRead())
	assert.Equal(t, "bar", n.GetMetadata("foo"))
	assert.Empty(t, n.GetMetadata("invalid"))
	assert.NotNil(t, n.MetadataRaw())
	n.ClearEvents()
	assert.Empty(t, n.Events())
	assert.False(t, n.RecipientID().IsZero())
	assert.NotEmpty(t, n.ID().String())
	assert.Equal(t, "Test Body", n.Body())

	// Add testing for zero ID and empty title
	_, err = notification.NewNotification(identity.UserID{}, notification.TypeNewFollower, "Title", "Body", nil)
	assert.ErrorIs(t, err, notification.ErrRecipientRequired)

	_, err = notification.NewNotification(recipientID, notification.TypeNewFollower, "", "Body", nil)
	assert.ErrorIs(t, err, notification.ErrTitleRequired)
}
