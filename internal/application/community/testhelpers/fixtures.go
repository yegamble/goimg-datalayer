package testhelpers

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// Test constants.
const (
	ValidOwnerID      = "550e8400-e29b-41d4-a716-446655440000"
	ValidUserID       = "550e8400-e29b-41d4-a716-446655440001"
	ValidAdminID      = "550e8400-e29b-41d4-a716-446655440002"
	ValidGroupID      = "660e8400-e29b-41d4-a716-446655440003"
	ValidMembershipID = "770e8400-e29b-41d4-a716-446655440004"

	ValidGroupName        = "Photography Enthusiasts"
	ValidGroupSlug        = "photography-enthusiasts"
	ValidGroupDescription = "A group for photography lovers"
)

// TestSuite provides mock dependencies for community tests.
type TestSuite struct {
	GroupRepo        *MockGroupRepository
	MembershipRepo   *MockGroupMembershipRepository
	ImageRepo        *MockGroupImageRepository
	InvitationRepo   *MockGroupInvitationRepository
	AlbumRepo        *MockGroupAlbumRepository
	AlbumImageRepo   *MockGroupAlbumImageRepository
	EventPublisher   *MockEventPublisher
	Logger           zerolog.Logger
}

// NewTestSuite creates a new test suite with mocked dependencies.
func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	return &TestSuite{
		GroupRepo:        new(MockGroupRepository),
		MembershipRepo:   new(MockGroupMembershipRepository),
		ImageRepo:        new(MockGroupImageRepository),
		InvitationRepo:   new(MockGroupInvitationRepository),
		AlbumRepo:        new(MockGroupAlbumRepository),
		AlbumImageRepo:   new(MockGroupAlbumImageRepository),
		EventPublisher:   new(MockEventPublisher),
		Logger:           zerolog.Nop(),
	}
}

// AssertExpectations verifies all mock expectations were met.
func (s *TestSuite) AssertExpectations(t *testing.T) {
	t.Helper()

	s.GroupRepo.AssertExpectations(t)
	s.MembershipRepo.AssertExpectations(t)
	s.ImageRepo.AssertExpectations(t)
	s.InvitationRepo.AssertExpectations(t)
	s.AlbumRepo.AssertExpectations(t)
	s.AlbumImageRepo.AssertExpectations(t)
	s.EventPublisher.AssertExpectations(t)
}

// ValidOwnerIDParsed returns a parsed owner UserID for testing.
func ValidOwnerIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidOwnerID)
	return userID
}

// ValidUserIDParsed returns a parsed UserID for testing.
func ValidUserIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidUserID)
	return userID
}

// ValidAdminIDParsed returns a parsed admin UserID for testing.
func ValidAdminIDParsed() identity.UserID {
	userID, _ := identity.ParseUserID(ValidAdminID)
	return userID
}

// ValidGroupIDParsed returns a parsed GroupID for testing.
func ValidGroupIDParsed() community.GroupID {
	groupID, _ := community.ParseGroupID(ValidGroupID)
	return groupID
}

// ValidPublicGroup creates a valid public Group aggregate for testing.
func ValidPublicGroup(t *testing.T) *community.Group {
	t.Helper()

	ownerID := ValidOwnerIDParsed()
	name, err := community.NewGroupName(ValidGroupName)
	require.NoError(t, err)
	slug, err := community.NewGroupSlug(ValidGroupSlug)
	require.NoError(t, err)

	group, err := community.NewGroup(ownerID, name, slug, community.GroupTypePublic)
	require.NoError(t, err)
	group.ClearEvents()

	return group
}

// ValidPrivateGroup creates a valid private Group aggregate for testing.
func ValidPrivateGroup(t *testing.T) *community.Group {
	t.Helper()

	ownerID := ValidOwnerIDParsed()
	name, err := community.NewGroupName(ValidGroupName)
	require.NoError(t, err)
	slug, err := community.NewGroupSlug(ValidGroupSlug)
	require.NoError(t, err)

	group, err := community.NewGroup(ownerID, name, slug, community.GroupTypePrivate)
	require.NoError(t, err)
	group.ClearEvents()

	return group
}

// ValidInviteOnlyGroup creates a valid invite-only Group aggregate for testing.
func ValidInviteOnlyGroup(t *testing.T) *community.Group {
	t.Helper()

	ownerID := ValidOwnerIDParsed()
	name, err := community.NewGroupName(ValidGroupName)
	require.NoError(t, err)
	slug, err := community.NewGroupSlug(ValidGroupSlug)
	require.NoError(t, err)

	group, err := community.NewGroup(ownerID, name, slug, community.GroupTypeInviteOnly)
	require.NoError(t, err)
	group.ClearEvents()

	return group
}

// ValidActiveMembership creates a valid active GroupMembership for testing.
func ValidActiveMembership(t *testing.T) *community.GroupMembership {
	t.Helper()

	groupID := ValidGroupIDParsed()
	userID := ValidUserIDParsed()

	membership, err := community.NewGroupMembership(groupID, userID, community.GroupRoleMember)
	require.NoError(t, err)
	membership.ClearEvents()

	return membership
}

// ValidBannedMembership creates a banned GroupMembership for testing.
func ValidBannedMembership(t *testing.T) *community.GroupMembership {
	t.Helper()

	groupID := ValidGroupIDParsed()
	userID := ValidUserIDParsed()
	adminID := ValidAdminIDParsed()

	membership, err := community.NewGroupMembership(groupID, userID, community.GroupRoleMember)
	require.NoError(t, err)

	err = membership.Ban(adminID, "Policy violation")
	require.NoError(t, err)
	membership.ClearEvents()

	return membership
}

// ValidRequestedMembership creates a requested (pending) GroupMembership for testing.
func ValidRequestedMembership(t *testing.T) *community.GroupMembership {
	t.Helper()

	groupID := ValidGroupIDParsed()
	userID := ValidUserIDParsed()

	membership, err := community.NewRequestedMembership(groupID, userID)
	require.NoError(t, err)
	membership.ClearEvents()

	return membership
}

// ValidOwnerMembership creates an owner GroupMembership for testing.
func ValidOwnerMembership(t *testing.T) *community.GroupMembership {
	t.Helper()

	groupID := ValidGroupIDParsed()
	ownerID := ValidOwnerIDParsed()

	membership, err := community.NewGroupMembership(groupID, ownerID, community.GroupRoleOwner)
	require.NoError(t, err)
	membership.ClearEvents()

	return membership
}
