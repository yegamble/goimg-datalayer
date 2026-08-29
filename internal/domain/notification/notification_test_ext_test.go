package notification

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"testing"
)

type dummyEvent struct {
	shared.BaseEvent
}

func TestNotification_MetadataRaw_And_Events(t *testing.T) {
	recipientID := identity.NewUserID()

	meta := map[string]string{
		"key": "value",
	}

	n, err := NewNotification(
		recipientID,
		TypeNewFollower,
		"Test Title",
		"Test Body",
		meta,
	)
	require.NoError(t, err)

	// test MetadataRaw
	raw := n.MetadataRaw()
	assert.NotEmpty(t, raw)

	// Events doesn't add on creation (it's passive)
	events := n.Events()
	assert.Len(t, events, 0)

	// Trigger addEvent since it's not triggered from MarkRead
	n.addEvent(dummyEvent{shared.NewBaseEvent("dummy", n.ID().String())})

	events = n.Events()
	assert.Len(t, events, 1)
}
