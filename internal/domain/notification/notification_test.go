package notification_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotification(t *testing.T) {
	recipientID := identity.NewUserID()
	typ := shared.NotificationTypeNewFollower
	title := "New Follower"
	body := "Someone followed you"

	n, err := notification.NewNotification(recipientID, typ, title, body, nil)
	require.NoError(t, err)

	err = n.MarkRead()
	require.NoError(t, err)

	assert.NotNil(t, n.MetadataRaw())
	n.ClearEvents()
	assert.Empty(t, n.Events())
}
