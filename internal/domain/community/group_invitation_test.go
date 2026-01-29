package community_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewGroupInvitation(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	invitedBy := identity.NewUserID()
	userID := identity.NewUserID()
	email, _ := identity.NewEmail("test@example.com")

	t.Run("valid invitation by user ID", func(t *testing.T) {
		inv, err := community.NewGroupInvitation(groupID, invitedBy, nil, &userID, 24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, inv)
		assert.Equal(t, groupID, inv.GroupID())
		assert.Equal(t, invitedBy, inv.InvitedBy())
		assert.Equal(t, &userID, inv.UserID())
		assert.Nil(t, inv.Email())
		assert.False(t, inv.IsExpired())
		assert.False(t, inv.IsUsed())
	})

	t.Run("valid invitation by email", func(t *testing.T) {
		inv, err := community.NewGroupInvitation(groupID, invitedBy, &email, nil, 24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, inv)
		assert.Equal(t, &email, inv.Email())
		assert.Nil(t, inv.UserID())
	})

	t.Run("invalid - missing recipient", func(t *testing.T) {
		_, err := community.NewGroupInvitation(groupID, invitedBy, nil, nil, 24*time.Hour)
		require.ErrorIs(t, err, community.ErrInvitationRecipientRequired)
	})
}

func TestGroupInvitation_MarkUsed(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	invitedBy := identity.NewUserID()
	userID := identity.NewUserID()
	inv, _ := community.NewGroupInvitation(groupID, invitedBy, nil, &userID, 24*time.Hour)

	err := inv.MarkUsed()
	require.NoError(t, err)
	assert.True(t, inv.IsUsed())

	// Idempotent
	err = inv.MarkUsed()
	require.NoError(t, err)
	assert.True(t, inv.IsUsed())
}

func TestReconstructGroupInvitation(t *testing.T) {
	t.Parallel()

	id := community.NewInvitationID()
	groupID := community.NewGroupID()
	invitedBy := identity.NewUserID()
	userID := identity.NewUserID()
	token, _ := community.NewInvitationToken()
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	inv := community.ReconstructGroupInvitation(
		id, groupID, invitedBy, nil, &userID, token, expiresAt, nil, now,
	)

	assert.Equal(t, id, inv.ID())
	assert.Equal(t, groupID, inv.GroupID())
	assert.Equal(t, token, inv.Token())
	assert.Equal(t, expiresAt, inv.ExpiresAt())
}
