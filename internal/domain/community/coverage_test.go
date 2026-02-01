package community_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestGroupReconstruction(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	ownerID := identity.NewUserID()
	name, _ := community.NewGroupName("Reconstructed Group")
	slug, _ := community.NewGroupSlug("reconstructed-group")
	settings := community.DefaultGroupSettings()
	now := time.Now()

	imageID := gallery.NewImageID()

	group := community.ReconstructGroup(
		groupID,
		name,
		slug,
		"Description",
		community.GroupTypePublic,
		ownerID,
		settings,
		10, // member count
		5,  // image count
		2,  // album count
		&imageID,
		now,
		now,
	)

	assert.Equal(t, groupID, group.ID())
	assert.Equal(t, name, group.Name())
	assert.Equal(t, slug, group.Slug())
	assert.Equal(t, "Description", group.Description())
	assert.Equal(t, community.GroupTypePublic, group.GroupType())
	assert.Equal(t, ownerID, group.OwnerID())
	assert.True(t, group.Settings().Equals(settings))
	assert.Equal(t, 10, group.MemberCount())
	assert.Equal(t, 5, group.ImageCount())
	assert.Equal(t, 2, group.AlbumCount())
	assert.NotNil(t, group.CoverImageID())
	assert.Equal(t, imageID, *group.CoverImageID())
	assert.Equal(t, now, group.CreatedAt())
	assert.Equal(t, now, group.UpdatedAt())
	assert.Empty(t, group.Events())
}

func TestGroupAlbumReconstruction(t *testing.T) {
	t.Parallel()

	albumID := community.NewGroupAlbumID()
	groupID := community.NewGroupID()
	creatorID := identity.NewUserID()
	title := "Album Title"
	now := time.Now()

	album := community.ReconstructGroupAlbum(
		albumID,
		groupID,
		creatorID,
		title,
		"Description",
		nil,  // cover image
		5,    // image count
		true, // is public
		now,
		now,
	)

	assert.Equal(t, albumID, album.ID())
	assert.Equal(t, groupID, album.GroupID())
	assert.Equal(t, creatorID, album.CreatedBy())
	assert.Equal(t, title, album.Title())
	assert.Equal(t, "Description", album.Description())
	assert.Equal(t, 5, album.ImageCount())
	assert.True(t, album.IsPublic())
	assert.Equal(t, now, album.CreatedAt())
	assert.Equal(t, now, album.UpdatedAt())
	assert.Empty(t, album.Events())
}

func TestGroupImageReconstruction(t *testing.T) {
	t.Parallel()

	groupImageID := community.NewGroupImageID()
	groupID := community.NewGroupID()
	imageID := gallery.NewImageID()
	sharerID := identity.NewUserID()
	reviewerID := identity.NewUserID()
	now := time.Now()

	img := community.ReconstructGroupImage(
		groupImageID,
		groupID,
		imageID,
		sharerID,
		community.GroupImageStatusApproved,
		&reviewerID,
		now,  // shared at
		&now, // reviewed at
	)

	assert.Equal(t, groupImageID, img.ID())
	assert.Equal(t, groupID, img.GroupID())
	assert.Equal(t, imageID, img.ImageID())
	assert.Equal(t, sharerID, img.SharedBy())
	assert.Equal(t, community.GroupImageStatusApproved, img.Status())
	assert.NotNil(t, img.ReviewedBy())
	assert.Equal(t, reviewerID, *img.ReviewedBy())
	assert.Equal(t, now, img.SharedAt())
	assert.Equal(t, now, *img.ReviewedAt())
}

func TestGroupInvitationReconstruction(t *testing.T) {
	t.Parallel()

	invitationID := community.NewInvitationID()
	groupID := community.NewGroupID()
	inviterID := identity.NewUserID()
	userID := identity.NewUserID()
	token, _ := community.NewInvitationToken()
	now := time.Now()
	expires := now.Add(24 * time.Hour)

	invitation := community.ReconstructGroupInvitation(
		invitationID,
		groupID,
		inviterID,
		nil, // email
		&userID,
		token,
		expires,
		&now, // used at
		now,  // created at
	)

	assert.Equal(t, invitationID, invitation.ID())
	assert.Equal(t, groupID, invitation.GroupID())
	assert.Equal(t, inviterID, invitation.InvitedBy())
	assert.Nil(t, invitation.Email())
	assert.Equal(t, userID, *invitation.UserID())
	assert.Equal(t, token, invitation.Token())
	assert.Equal(t, expires, invitation.ExpiresAt())
	assert.NotNil(t, invitation.UsedAt())
	assert.Equal(t, now, *invitation.UsedAt())
	assert.Equal(t, now, invitation.CreatedAt())
}

func TestGroupMembershipReconstruction(t *testing.T) {
	t.Parallel()

	membershipID := community.NewMembershipID()
	groupID := community.NewGroupID()
	userID := identity.NewUserID()
	now := time.Now()

	membership := community.ReconstructGroupMembership(
		membershipID,
		groupID,
		userID,
		community.GroupRoleMember,
		community.MemberStatusActive,
		nil, // invited by
		now,
		now,
	)

	assert.Equal(t, membershipID, membership.ID())
	assert.Equal(t, groupID, membership.GroupID())
	assert.Equal(t, userID, membership.UserID())
	assert.Equal(t, community.GroupRoleMember, membership.Role())
	assert.Equal(t, community.MemberStatusActive, membership.Status())
	assert.Equal(t, now, membership.JoinedAt())
	assert.Equal(t, now, membership.UpdatedAt())
}

func TestGroupSettings(t *testing.T) {
	t.Parallel()

	t.Run("Validation", func(t *testing.T) {
		_, err := community.NewGroupSettings(true, true, true, -1)
		assert.ErrorIs(t, err, community.ErrInvalidMaxMembers)

		settings, err := community.NewGroupSettings(true, true, true, 100)
		require.NoError(t, err)
		assert.True(t, settings.RequireApproval())
		assert.True(t, settings.AllowMemberInvites())
		assert.True(t, settings.AllowMemberAlbums())
		assert.Equal(t, 100, settings.MaxMembers())
		assert.True(t, settings.HasMemberLimit())
	})

	t.Run("Default", func(t *testing.T) {
		settings := community.DefaultGroupSettings()
		assert.False(t, settings.RequireApproval())
		assert.False(t, settings.HasMemberLimit())
	})

	t.Run("With Methods", func(t *testing.T) {
		s := community.DefaultGroupSettings()
		s = s.WithRequireApproval(true)
		assert.True(t, s.RequireApproval())

		s = s.WithAllowMemberInvites(false)
		assert.False(t, s.AllowMemberInvites())

		s = s.WithAllowMemberAlbums(false)
		assert.False(t, s.AllowMemberAlbums())

		var err error
		s, err = s.WithMaxMembers(50)
		require.NoError(t, err)
		assert.Equal(t, 50, s.MaxMembers())
	})
}

// Additional Group Coverage Tests

func TestGroup_EdgeCases(t *testing.T) {
	t.Parallel()

	ownerID := identity.NewUserID()
	name, _ := community.NewGroupName("Edge Case Group")
	slug, _ := community.NewGroupSlug("edge-case-group")
	group, err := community.NewGroup(ownerID, name, slug, community.GroupTypePublic)
	require.NoError(t, err)

	t.Run("UpdateDescription - Max Length", func(t *testing.T) {
		// Valid length
		err := group.UpdateDescription("Valid description")
		assert.NoError(t, err)

		// Too long
		longDesc := make([]byte, community.MaxDescriptionLength+1)
		for i := range longDesc {
			longDesc[i] = 'a'
		}
		err = group.UpdateDescription(string(longDesc))
		assert.ErrorIs(t, err, community.ErrGroupDescTooLong)
	})

	t.Run("UpdateSettings - No Change", func(t *testing.T) {
		group.ClearEvents()
		err := group.UpdateSettings(group.Settings())
		assert.NoError(t, err)
		assert.Empty(t, group.Events())
	})

	t.Run("SetCoverImage - No Change", func(t *testing.T) {
		// Set image
		imgID := gallery.NewImageID()
		group.SetCoverImage(&imgID)
		group.ClearEvents()

		// Set same image
		group.SetCoverImage(&imgID)
		assert.Empty(t, group.Events())

		// Remove image
		group.SetCoverImage(nil)
		group.ClearEvents()

		// Remove nil (no-op)
		group.SetCoverImage(nil)
		assert.Empty(t, group.Events())
	})

	t.Run("Decrement Counts Below Zero", func(t *testing.T) {
		// Reset to 0
		for group.ImageCount() > 0 {
			group.DecrementImageCount()
		}
		assert.Equal(t, 0, group.ImageCount())

		// Try decrementing 0
		group.DecrementImageCount()
		assert.Equal(t, 0, group.ImageCount())

		// Same for albums
		for group.AlbumCount() > 0 {
			group.DecrementAlbumCount()
		}
		assert.Equal(t, 0, group.AlbumCount())

		group.DecrementAlbumCount()
		assert.Equal(t, 0, group.AlbumCount())
	})
}

func TestGroupInvitation(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	inviterID := identity.NewUserID()
	email := "test@example.com"

	t.Run("NewGroupInvitation", func(t *testing.T) {
		invitation, err := community.NewGroupInvitation(groupID, inviterID, &email, nil)
		require.NoError(t, err)

		assert.False(t, invitation.IsUsed())
		assert.False(t, invitation.IsExpired())
		assert.True(t, invitation.CanBeAccepted())
		assert.Equal(t, &email, invitation.Email())
	})

	t.Run("Accept", func(t *testing.T) {
		invitation, _ := community.NewGroupInvitation(groupID, inviterID, &email, nil)

		err := invitation.Accept()
		require.NoError(t, err)

		assert.True(t, invitation.IsUsed())
		assert.False(t, invitation.CanBeAccepted())

		// Accept again
		err = invitation.Accept()
		assert.ErrorIs(t, err, community.ErrInvitationAlreadyUsed)
	})

	t.Run("Expired", func(t *testing.T) {
		invitation, _ := community.NewGroupInvitation(groupID, inviterID, &email, nil)

		// Hack to expire it (using reconstruct for test)
		expired := community.ReconstructGroupInvitation(
			invitation.ID(), invitation.GroupID(), invitation.InvitedBy(), invitation.Email(),
			nil, invitation.Token(), time.Now().Add(-1*time.Hour), nil, invitation.CreatedAt(),
		)

		assert.True(t, expired.IsExpired())
		assert.False(t, expired.CanBeAccepted())

		err := expired.Accept()
		assert.ErrorIs(t, err, community.ErrInvitationExpired)
	})
}

func TestInvitationID(t *testing.T) {
	t.Parallel()

	id := community.NewInvitationID()
	assert.False(t, id.IsZero())

	parsed, err := community.ParseInvitationID(id.String())
	require.NoError(t, err)
	assert.True(t, id.Equals(parsed))
	assert.Equal(t, id.String(), parsed.String())

	// MustParse
	require.NotPanics(t, func() {
		community.MustParseInvitationID(id.String())
	})

	require.Panics(t, func() {
		community.MustParseInvitationID("invalid")
	})

	_, err = community.ParseInvitationID("invalid")
	assert.Error(t, err)
}

func TestInvitationToken(t *testing.T) {
	t.Parallel()

	token, err := community.NewInvitationToken()
	require.NoError(t, err)
	assert.False(t, token.IsEmpty())

	parsed, err := community.ParseInvitationToken(token.String())
	require.NoError(t, err)
	assert.True(t, token.Equals(parsed))

	_, err = community.ParseInvitationToken("")
	assert.Error(t, err)

	_, err = community.ParseInvitationToken("short")
	assert.Error(t, err)
}

func TestMembershipID(t *testing.T) {
	t.Parallel()

	id := community.NewMembershipID()
	assert.False(t, id.IsZero())

	parsed, err := community.ParseMembershipID(id.String())
	require.NoError(t, err)
	assert.True(t, id.Equals(parsed))
	assert.Equal(t, id.String(), parsed.String())

	// MustParse
	require.NotPanics(t, func() {
		community.MustParseMembershipID(id.String())
	})

	require.Panics(t, func() {
		community.MustParseMembershipID("invalid")
	})

	_, err = community.ParseMembershipID("invalid")
	assert.Error(t, err)
}

func TestGroupRole(t *testing.T) {
	t.Parallel()

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, community.GroupRoleOwner.IsValid())
		assert.False(t, community.GroupRole(999).IsValid())
	})

	t.Run("Permissions", func(t *testing.T) {
		assert.True(t, community.GroupRoleOwner.CanManageMembers())
		assert.True(t, community.GroupRoleAdmin.CanManageMembers())
		assert.False(t, community.GroupRoleMember.CanManageMembers())

		assert.True(t, community.GroupRoleOwner.CanDeleteGroup())
		assert.False(t, community.GroupRoleAdmin.CanDeleteGroup())

		assert.True(t, community.GroupRoleOwner.CanUpdateSettings())
		assert.True(t, community.GroupRoleAdmin.CanUpdateSettings())
		assert.False(t, community.GroupRoleMember.CanUpdateSettings())

		assert.True(t, community.GroupRoleOwner.CanModerateContent())

		assert.True(t, community.GroupRoleOwner.IsOwner())
		assert.True(t, community.GroupRoleAdmin.IsAdmin())

		assert.True(t, community.GroupRoleOwner.IsHigherThan(community.GroupRoleAdmin))
		assert.False(t, community.GroupRoleAdmin.IsHigherThan(community.GroupRoleOwner))
	})

	t.Run("Parse", func(t *testing.T) {
		role, err := community.ParseGroupRole("admin")
		require.NoError(t, err)
		assert.Equal(t, community.GroupRoleAdmin, role)

		_, err = community.ParseGroupRole("invalid")
		assert.Error(t, err)
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "owner", community.GroupRoleOwner.String())
		assert.Equal(t, "unknown", community.GroupRole(999).String())
	})
}

func TestMemberStatus(t *testing.T) {
	t.Parallel()

	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, community.MemberStatusActive.IsValid())
		assert.False(t, community.MemberStatus(999).IsValid())
	})

	t.Run("Checks", func(t *testing.T) {
		assert.True(t, community.MemberStatusActive.IsActive())
		assert.True(t, community.MemberStatusInvited.IsPending())
		assert.True(t, community.MemberStatusBanned.IsBanned())
		assert.True(t, community.MemberStatusInvited.CanAcceptInvitation())
	})

	t.Run("Parse", func(t *testing.T) {
		status, err := community.ParseMemberStatus("active")
		require.NoError(t, err)
		assert.Equal(t, community.MemberStatusActive, status)

		_, err = community.ParseMemberStatus("invalid")
		assert.Error(t, err)
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "active", community.MemberStatusActive.String())
		assert.Equal(t, "unknown", community.MemberStatus(999).String())
	})
}

func TestGroupMembership_Constructors(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	userID := identity.NewUserID()
	inviterID := identity.NewUserID()

	t.Run("NewInvitedMembership", func(t *testing.T) {
		m, err := community.NewInvitedMembership(groupID, userID, inviterID, community.GroupRoleMember)
		require.NoError(t, err)
		assert.Equal(t, community.MemberStatusInvited, m.Status())
		assert.Equal(t, &inviterID, m.InvitedBy())

		// Validation
		_, err = community.NewInvitedMembership(community.GroupID{}, userID, inviterID, community.GroupRoleMember)
		assert.Error(t, err)
		_, err = community.NewInvitedMembership(groupID, identity.UserID{}, inviterID, community.GroupRoleMember)
		assert.Error(t, err)
		_, err = community.NewInvitedMembership(groupID, userID, identity.UserID{}, community.GroupRoleMember)
		assert.Error(t, err)
		_, err = community.NewInvitedMembership(groupID, userID, inviterID, community.GroupRole(999))
		assert.Error(t, err)
	})

	t.Run("NewRequestedMembership", func(t *testing.T) {
		m, err := community.NewRequestedMembership(groupID, userID)
		require.NoError(t, err)
		assert.Equal(t, community.MemberStatusRequested, m.Status())
		assert.Equal(t, community.GroupRoleMember, m.Role())

		// Validation
		_, err = community.NewRequestedMembership(community.GroupID{}, userID)
		assert.Error(t, err)
		_, err = community.NewRequestedMembership(groupID, identity.UserID{})
		assert.Error(t, err)
	})

	t.Run("NewGroupMembership", func(t *testing.T) {
		m, err := community.NewGroupMembership(groupID, userID, community.GroupRoleOwner)
		require.NoError(t, err)
		assert.Equal(t, community.MemberStatusActive, m.Status())
		assert.Equal(t, community.GroupRoleOwner, m.Role())

		// Validation
		_, err = community.NewGroupMembership(community.GroupID{}, userID, community.GroupRoleOwner)
		assert.Error(t, err)
		_, err = community.NewGroupMembership(groupID, identity.UserID{}, community.GroupRoleOwner)
		assert.Error(t, err)
		_, err = community.NewGroupMembership(groupID, userID, community.GroupRole(999))
		assert.Error(t, err)
	})
}

func TestGroupActivity(t *testing.T) {
	t.Parallel()

	groupID := community.NewGroupID()
	actorID := identity.NewUserID()
	targetID := "target-123"
	targetType := community.TargetTypeUser
	meta := map[string]interface{}{"key": "value"}

	t.Run("NewGroupActivity", func(t *testing.T) {
		activity, err := community.NewGroupActivity(
			groupID, actorID, community.ActivityTypeMemberJoined,
			&targetID, &targetType, meta,
		)
		require.NoError(t, err)
		assert.False(t, activity.ID().IsZero())
		assert.Equal(t, groupID, activity.GroupID())
		assert.Equal(t, actorID, activity.ActorID())
		assert.Equal(t, community.ActivityTypeMemberJoined, activity.ActivityType())
		assert.Equal(t, &targetID, activity.TargetID())
		assert.Equal(t, &targetType, activity.TargetType())
		assert.Equal(t, meta, activity.Metadata())
		assert.False(t, activity.CreatedAt().IsZero())
	})

	t.Run("Validation", func(t *testing.T) {
		// Invalid GroupID
		_, err := community.NewGroupActivity(
			community.GroupID{}, actorID, community.ActivityTypeMemberJoined,
			&targetID, &targetType, meta,
		)
		assert.Error(t, err)

		// Invalid ActivityType
		_, err = community.NewGroupActivity(
			groupID, actorID, community.ActivityType("invalid"),
			&targetID, &targetType, meta,
		)
		assert.Error(t, err)

		// Mismatched Target (ID but no Type)
		_, err = community.NewGroupActivity(
			groupID, actorID, community.ActivityTypeMemberJoined,
			&targetID, nil, meta,
		)
		assert.Error(t, err)

		// Invalid TargetType
		invalidType := community.TargetType("invalid")
		_, err = community.NewGroupActivity(
			groupID, actorID, community.ActivityTypeMemberJoined,
			&targetID, &invalidType, meta,
		)
		assert.Error(t, err)
	})

	t.Run("Factories", func(t *testing.T) {
		targetUser := identity.NewUserID()

		// NewMemberBannedActivity
		activity, err := community.NewMemberBannedActivity(groupID, actorID, targetUser, "Reason")
		require.NoError(t, err)
		assert.Equal(t, community.ActivityTypeMemberBanned, activity.ActivityType())

		// NewMemberRemovedActivity
		activity, err = community.NewMemberRemovedActivity(groupID, actorID, targetUser)
		require.NoError(t, err)
		assert.Equal(t, community.ActivityTypeMemberRemoved, activity.ActivityType())

		// NewMemberRoleChangedActivity
		activity, err = community.NewMemberRoleChangedActivity(groupID, actorID, targetUser, community.GroupRoleMember, community.GroupRoleAdmin)
		require.NoError(t, err)
		assert.Equal(t, community.ActivityTypeMemberPromoted, activity.ActivityType())

		// NewGroupDeletedActivity
		activity, err = community.NewGroupDeletedActivity(groupID, actorID)
		require.NoError(t, err)
		assert.Equal(t, community.ActivityTypeGroupDeleted, activity.ActivityType())
	})

	t.Run("Enums", func(t *testing.T) {
		assert.Equal(t, "member_joined", community.ActivityTypeMemberJoined.String())
		assert.True(t, community.ActivityTypeMemberJoined.IsValid())

		assert.Equal(t, "user", community.TargetTypeUser.String())
		assert.True(t, community.TargetTypeUser.IsValid())
	})

	t.Run("Reconstruct", func(t *testing.T) {
		id := community.NewGroupActivityID()
		now := time.Now()

		activity := community.ReconstructGroupActivity(
			id, groupID, actorID, community.ActivityTypeMemberJoined,
			&targetID, &targetType, meta, now,
		)

		assert.Equal(t, id, activity.ID())
		assert.Equal(t, groupID, activity.GroupID())
		assert.Equal(t, actorID, activity.ActorID())
		assert.Equal(t, now, activity.CreatedAt())
	})
}
