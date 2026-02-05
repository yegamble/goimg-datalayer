package notification

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNewNotification(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	notifType := TypeNewFollower
	title := "Welcome"
	body := "Welcome to the app!"
	metadata := map[string]string{"key": "value"}

	t.Run("Success", func(t *testing.T) {
		n, err := NewNotification(recipientID, notifType, title, body, metadata)
		require.NoError(t, err)
		assert.NotNil(t, n)
		assert.NotEmpty(t, n.ID())
		assert.Equal(t, recipientID, n.RecipientID())
		assert.Equal(t, notifType, n.Type())
		assert.Equal(t, title, n.Title())
		assert.Equal(t, body, n.Body())
		assert.Equal(t, metadata, n.Metadata())
		assert.False(t, n.IsRead())
		assert.Nil(t, n.ReadAt())
		assert.WithinDuration(t, time.Now(), n.CreatedAt(), 2*time.Second)
		assert.Empty(t, n.Events())
	})

	t.Run("Success_NilMetadata", func(t *testing.T) {
		n, err := NewNotification(recipientID, notifType, title, body, nil)
		require.NoError(t, err)
		assert.NotNil(t, n.Metadata())
		assert.Empty(t, n.Metadata())
	})

	t.Run("Error_ZeroRecipient", func(t *testing.T) {
		n, err := NewNotification(identity.UserID{}, notifType, title, body, metadata)
		assert.ErrorIs(t, err, ErrRecipientRequired)
		assert.Nil(t, n)
	})

	t.Run("Error_EmptyTitle", func(t *testing.T) {
		n, err := NewNotification(recipientID, notifType, "", body, metadata)
		assert.ErrorIs(t, err, ErrTitleRequired)
		assert.Nil(t, n)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		n, err := NewNotification(recipientID, NotificationType("invalid"), title, body, metadata)
		assert.ErrorIs(t, err, ErrInvalidNotificationType)
		assert.Nil(t, n)
	})
}

func TestReconstructNotification(t *testing.T) {
	t.Parallel()

	id := NewNotificationID()
	recipientID := identity.NewUserID()
	notifType := TypeNewFollower
	title := "Reconstructed"
	body := "Body"
	metadata := map[string]string{"foo": "bar"}
	metadataBytes, _ := json.Marshal(metadata)
	now := time.Now().UTC()
	readAt := &now

	n := ReconstructNotification(id, recipientID, notifType, title, body, metadataBytes, readAt, now)

	assert.Equal(t, id, n.ID())
	assert.Equal(t, recipientID, n.RecipientID())
	assert.Equal(t, notifType, n.Type())
	assert.Equal(t, title, n.Title())
	assert.Equal(t, body, n.Body())
	assert.Equal(t, metadata, n.Metadata())
	assert.Equal(t, readAt, n.ReadAt())
	assert.True(t, n.IsRead())
	assert.Equal(t, now, n.CreatedAt())
	assert.Empty(t, n.Events())

	// Test with nil metadata
	n2 := ReconstructNotification(id, recipientID, notifType, title, body, nil, nil, now)
	assert.NotNil(t, n2.Metadata())
}

func TestNotification_MarkRead(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)

	// First mark read
	err := n.MarkRead()
	require.NoError(t, err)
	assert.True(t, n.IsRead())
	assert.NotNil(t, n.ReadAt())
	firstReadAt := *n.ReadAt()

	// Second mark read (idempotent)
	time.Sleep(10 * time.Millisecond) // Ensure time passes
	err = n.MarkRead()
	require.NoError(t, err)
	assert.Equal(t, firstReadAt, *n.ReadAt())
}

func TestNotification_GetMetadata(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	metadata := map[string]string{"key": "value"}
	n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", metadata)

	assert.Equal(t, "value", n.GetMetadata("key"))
	assert.Equal(t, "", n.GetMetadata("missing"))
}

func TestNotification_Validate(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)

	// Valid
	assert.NoError(t, n.Validate())

	// Invalid recipient
	n2, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)
	n2.recipientID = identity.UserID{}
	assert.ErrorIs(t, n2.Validate(), ErrRecipientRequired)

	// Invalid title
	n3, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)
	n3.title = ""
	assert.ErrorIs(t, n3.Validate(), ErrTitleRequired)

	// Invalid type
	n4, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)
	n4.notifType = NotificationType("invalid")
	assert.ErrorIs(t, n4.Validate(), ErrInvalidNotificationType)
}

func TestNotification_Lifecycle(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)

	n.ClearEvents()
	assert.Empty(t, n.Events())

	// Test addEvent (private method, whitebox testing for coverage)
	event := shared.NewDomainEvent(identity.NewUserID(), "test.event", 1, nil)
	n.addEvent(event)
	assert.Len(t, n.Events(), 1)
	assert.Equal(t, event.ID(), n.Events()[0].ID())

	n.ClearEvents()
	assert.Empty(t, n.Events())
}

func TestNotification_MetadataRaw(t *testing.T) {
	t.Parallel()

	// Case 1: Metadata map is present (created via NewNotification)
	recipientID := identity.NewUserID()
	metadata := map[string]string{"key": "value"}
	n1, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", metadata)

	raw1 := n1.MetadataRaw()
	assert.JSONEq(t, `{"key":"value"}`, string(raw1))

	// Case 2: Metadata map is nil, but raw bytes are present (reconstructed)
	// Important: Do NOT call Metadata() before MetadataRaw() to test lazy loading
	expectedRaw := []byte(`{"foo":"bar"}`)
	n2 := ReconstructNotification(NewNotificationID(), recipientID, TypeNewFollower, "Title", "Body", expectedRaw, nil, time.Now())

	raw2 := n2.MetadataRaw()
	assert.Equal(t, expectedRaw, raw2)

	// Case 3: Both are nil
	n3 := ReconstructNotification(NewNotificationID(), recipientID, TypeNewFollower, "Title", "Body", nil, nil, time.Now())

	raw3 := n3.MetadataRaw()
	assert.Equal(t, []byte("{}"), raw3)
}
