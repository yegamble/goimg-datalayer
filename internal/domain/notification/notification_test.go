package notification

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewNotification(t *testing.T) {
	t.Parallel()

	recipientID := identity.NewUserID()
	notifType := TypeNewFollower
	title := "Welcome"
	body := "Welcome to the app!"
	metadata := map[string]string{"key": "value"}

	t.Run("Success", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

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
		t.Parallel() // Fix: Add t.Parallel()

		n, err := NewNotification(recipientID, notifType, title, body, nil)
		require.NoError(t, err)
		assert.NotNil(t, n.Metadata())
		assert.Empty(t, n.Metadata())
	})

	t.Run("Error_ZeroRecipient", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

		n, err := NewNotification(identity.UserID{}, notifType, title, body, metadata)
		assert.ErrorIs(t, err, ErrRecipientRequired)
		assert.Nil(t, n)
	})

	t.Run("Error_EmptyTitle", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

		n, err := NewNotification(recipientID, notifType, "", body, metadata)
		assert.ErrorIs(t, err, ErrTitleRequired)
		assert.Nil(t, n)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

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
}

func TestNotification_MetadataRaw(t *testing.T) {
	t.Parallel()

	t.Run("Generates JSON from map", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

		metadata := map[string]string{"foo": "bar"}
		recipientID := identity.NewUserID()
		n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", metadata)

		raw := n.MetadataRaw()
		assert.Contains(t, string(raw), `"foo":"bar"`)
	})

	t.Run("Returns empty JSON object for nil metadata", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

		recipientID := identity.NewUserID()
		n, _ := NewNotification(recipientID, TypeNewFollower, "Title", "Body", nil)

		// Manually set metadata to nil to simulate loaded state where map isn't hydrated
		// (NewNotification creates an empty map, so we override it)
		n.metadata = nil
		n.metadataRaw = nil

		raw := n.MetadataRaw()
		assert.Equal(t, []byte("{}"), raw)
	})

	t.Run("Returns cached raw bytes if map is nil", func(t *testing.T) {
		t.Parallel() // Fix: Add t.Parallel()

		expectedRaw := []byte(`{"cached":"true"}`)
		n := ReconstructNotification(
			NewNotificationID(),
			identity.NewUserID(),
			TypeNewFollower,
			"Title",
			"Body",
			expectedRaw,
			nil,
			time.Now(),
		)

		assert.Equal(t, expectedRaw, n.MetadataRaw())
	})
}
