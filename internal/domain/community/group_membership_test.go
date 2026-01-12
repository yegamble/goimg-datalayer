package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewGroupMembership(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("valid membership", func(t *testing.T) {
		membership, err := NewGroupMembership(groupID, userID, GroupRoleMember)

		require.NoError(t, err)
		assert.False(t, membership.ID().IsZero())
		assert.True(t, membership.GroupID().Equals(groupID))
		assert.True(t, membership.UserID().Equals(userID))
		assert.Equal(t, GroupRoleMember, membership.Role())
		assert.Equal(t, MemberStatusActive, membership.Status())
		assert.Nil(t, membership.InvitedBy())
		assert.False(t, membership.JoinedAt().IsZero())
		assert.False(t, membership.UpdatedAt().IsZero())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberJoined)
		assert.True(t, ok)
		assert.True(t, event.GroupID.Equals(groupID))
		assert.True(t, event.UserID.Equals(userID))
	})

	t.Run("invalid - zero group ID", func(t *testing.T) {
		_, err := NewGroupMembership(GroupID{}, userID, GroupRoleMember)
		require.Error(t, err)
	})

	t.Run("invalid - zero user ID", func(t *testing.T) {
		_, err := NewGroupMembership(groupID, identity.UserID{}, GroupRoleMember)
		require.Error(t, err)
	})

	t.Run("invalid - invalid role", func(t *testing.T) {
		_, err := NewGroupMembership(groupID, userID, GroupRole(999))
		require.ErrorIs(t, err, ErrInvalidGroupRole)
	})
}

func TestNewInvitedMembership(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()
	inviterID := identity.NewUserID()

	t.Run("valid invitation", func(t *testing.T) {
		membership, err := NewInvitedMembership(groupID, userID, inviterID, GroupRoleMember)

		require.NoError(t, err)
		assert.False(t, membership.ID().IsZero())
		assert.Equal(t, MemberStatusInvited, membership.Status())
		assert.NotNil(t, membership.InvitedBy())
		assert.True(t, membership.InvitedBy().Equals(inviterID))

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberInvited)
		assert.True(t, ok)
		assert.True(t, event.InvitedBy.Equals(inviterID))
	})

	t.Run("invalid - zero inviter ID", func(t *testing.T) {
		_, err := NewInvitedMembership(groupID, userID, identity.UserID{}, GroupRoleMember)
		require.Error(t, err)
	})
}

func TestNewRequestedMembership(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	membership, err := NewRequestedMembership(groupID, userID)

	require.NoError(t, err)
	assert.Equal(t, MemberStatusRequested, membership.Status())
	assert.Equal(t, GroupRoleMember, membership.Role()) // Always starts as member
	assert.Nil(t, membership.InvitedBy())

	// Check event
	assert.Len(t, membership.Events(), 1)
	event, ok := membership.Events()[0].(*MemberRequested)
	assert.True(t, ok)
	assert.True(t, event.GroupID.Equals(groupID))
	assert.True(t, event.UserID.Equals(userID))
}

func TestGroupMembership_Activate(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("activate invited member", func(t *testing.T) {
		inviterID := identity.NewUserID()
		membership, _ := NewInvitedMembership(groupID, userID, inviterID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.Activate()
		require.NoError(t, err)
		assert.Equal(t, MemberStatusActive, membership.Status())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberActivated)
		assert.True(t, ok)
		assert.True(t, event.GroupID.Equals(groupID))
		assert.True(t, event.UserID.Equals(userID))
	})

	t.Run("activate requested member", func(t *testing.T) {
		membership, _ := NewRequestedMembership(groupID, userID)
		membership.ClearEvents()

		err := membership.Activate()
		require.NoError(t, err)
		assert.Equal(t, MemberStatusActive, membership.Status())
	})

	t.Run("already active", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)

		err := membership.Activate()
		require.ErrorIs(t, err, ErrMemberAlreadyActive)
	})

	t.Run("banned member cannot be activated", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		bannerID := identity.NewUserID()
		_ = membership.Ban(bannerID, "spam")

		err := membership.Activate()
		require.ErrorIs(t, err, ErrMemberBanned)
	})
}

func TestGroupMembership_PromoteToAdmin(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("promote member to admin", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.PromoteToAdmin()
		require.NoError(t, err)
		assert.Equal(t, GroupRoleAdmin, membership.Role())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberRoleChanged)
		assert.True(t, ok)
		assert.Equal(t, GroupRoleMember, event.OldRole)
		assert.Equal(t, GroupRoleAdmin, event.NewRole)
	})

	t.Run("already admin - no change", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleAdmin)
		membership.ClearEvents()

		err := membership.PromoteToAdmin()
		require.NoError(t, err)
		assert.Len(t, membership.Events(), 0) // No event when no change
	})

	t.Run("owner - no change", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleOwner)
		membership.ClearEvents()

		err := membership.PromoteToAdmin()
		require.NoError(t, err)
		assert.Equal(t, GroupRoleOwner, membership.Role()) // Stays owner
		assert.Len(t, membership.Events(), 0)
	})
}

func TestGroupMembership_DemoteToMember(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("demote admin to member", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleAdmin)
		membership.ClearEvents()

		err := membership.DemoteToMember()
		require.NoError(t, err)
		assert.Equal(t, GroupRoleMember, membership.Role())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberRoleChanged)
		assert.True(t, ok)
		assert.Equal(t, GroupRoleAdmin, event.OldRole)
		assert.Equal(t, GroupRoleMember, event.NewRole)
	})

	t.Run("already member - no change", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.DemoteToMember()
		require.NoError(t, err)
		assert.Len(t, membership.Events(), 0)
	})

	t.Run("cannot demote owner", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleOwner)

		err := membership.DemoteToMember()
		require.ErrorIs(t, err, ErrCannotDemoteOwner)
		assert.Equal(t, GroupRoleOwner, membership.Role()) // Unchanged
	})
}

func TestGroupMembership_ChangeRole(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("change from member to admin", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.ChangeRole(GroupRoleAdmin)
		require.NoError(t, err)
		assert.Equal(t, GroupRoleAdmin, membership.Role())
		assert.Len(t, membership.Events(), 1)
	})

	t.Run("cannot change owner role", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleOwner)

		err := membership.ChangeRole(GroupRoleAdmin)
		require.ErrorIs(t, err, ErrCannotDemoteOwner)
	})

	t.Run("invalid role", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)

		err := membership.ChangeRole(GroupRole(999))
		require.ErrorIs(t, err, ErrInvalidGroupRole)
	})

	t.Run("no change - no event", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.ChangeRole(GroupRoleMember)
		require.NoError(t, err)
		assert.Len(t, membership.Events(), 0)
	})
}

func TestGroupMembership_Ban(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()
	bannerID := identity.NewUserID()

	t.Run("ban active member", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.Ban(bannerID, "spamming")
		require.NoError(t, err)
		assert.Equal(t, MemberStatusBanned, membership.Status())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberBanned)
		assert.True(t, ok)
		assert.True(t, event.BannedBy.Equals(bannerID))
		assert.Equal(t, "spamming", event.Reason)
	})

	t.Run("cannot ban owner", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleOwner)

		err := membership.Ban(bannerID, "test")
		require.ErrorIs(t, err, ErrCannotBanOwner)
		assert.Equal(t, MemberStatusActive, membership.Status())
	})

	t.Run("already banned", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		_ = membership.Ban(bannerID, "first ban")

		err := membership.Ban(bannerID, "second ban")
		require.ErrorIs(t, err, ErrMemberAlreadyBanned)
	})
}

func TestGroupMembership_Unban(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()
	bannerID := identity.NewUserID()

	t.Run("unban banned member", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		_ = membership.Ban(bannerID, "spam")
		membership.ClearEvents()

		err := membership.Unban()
		require.NoError(t, err)
		assert.Equal(t, MemberStatusActive, membership.Status())

		// Check event
		assert.Len(t, membership.Events(), 1)
		event, ok := membership.Events()[0].(*MemberUnbanned)
		assert.True(t, ok)
		assert.True(t, event.GroupID.Equals(groupID))
		assert.True(t, event.UserID.Equals(userID))
	})

	t.Run("not banned - no change", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		membership.ClearEvents()

		err := membership.Unban()
		require.NoError(t, err)
		assert.Len(t, membership.Events(), 0)
	})
}

func TestGroupMembership_HelperMethods(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()

	t.Run("IsActive", func(t *testing.T) {
		active, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		invited, _ := NewInvitedMembership(groupID, userID, identity.NewUserID(), GroupRoleMember)

		assert.True(t, active.IsActive())
		assert.False(t, invited.IsActive())
	})

	t.Run("IsBanned", func(t *testing.T) {
		membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		assert.False(t, membership.IsBanned())

		_ = membership.Ban(identity.NewUserID(), "test")
		assert.True(t, membership.IsBanned())
	})

	t.Run("role checks", func(t *testing.T) {
		member, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		admin, _ := NewGroupMembership(groupID, userID, GroupRoleAdmin)
		owner, _ := NewGroupMembership(groupID, userID, GroupRoleOwner)

		assert.False(t, member.IsOwner())
		assert.False(t, member.IsAdmin())
		assert.True(t, member.IsMember())

		assert.False(t, admin.IsOwner())
		assert.True(t, admin.IsAdmin())
		assert.False(t, admin.IsMember())

		assert.True(t, owner.IsOwner())
		assert.False(t, owner.IsAdmin())
		assert.False(t, owner.IsMember())
	})

	t.Run("permission checks", func(t *testing.T) {
		member, _ := NewGroupMembership(groupID, userID, GroupRoleMember)
		admin, _ := NewGroupMembership(groupID, userID, GroupRoleAdmin)
		bannedMember, _ := NewGroupMembership(groupID, userID, GroupRoleAdmin)
		_ = bannedMember.Ban(identity.NewUserID(), "test")

		assert.False(t, member.CanManageMembers())
		assert.True(t, admin.CanManageMembers())
		assert.False(t, bannedMember.CanManageMembers()) // Banned can't manage

		assert.False(t, member.CanModerateContent())
		assert.True(t, admin.CanModerateContent())
		assert.False(t, bannedMember.CanModerateContent()) // Banned can't moderate
	})
}

func TestGroupMembership_ClearEvents(t *testing.T) {
	t.Parallel()

	groupID := NewGroupID()
	userID := identity.NewUserID()
	membership, _ := NewGroupMembership(groupID, userID, GroupRoleMember)

	assert.Len(t, membership.Events(), 1)

	membership.ClearEvents()
	assert.Len(t, membership.Events(), 0)
}
