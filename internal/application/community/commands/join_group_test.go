package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/community/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestJoinGroupHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error)
	}{
		{
			name: "successful join to public group",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, community.ErrMembershipNotFound).Once()
				suite.GroupRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.MembershipRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.IsActive())
				assert.Equal(t, community.GroupRoleMember, result.Role())
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful join request to invite-only group",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidInviteOnlyGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, community.ErrMembershipNotFound).Once()
				suite.MembershipRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, community.MemberStatusRequested, result.Status())
				suite.AssertExpectations(t)
			},
		},
		{
			name: "cannot join private group",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPrivateGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, community.ErrMembershipNotFound).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "private",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, community.ErrPrivateGroupNoAccess)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "already a member",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				existingMembership := testhelpers.ValidActiveMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(existingMembership, nil).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "already",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, community.ErrAlreadyGroupMember)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "banned user cannot rejoin",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				bannedMembership := testhelpers.ValidBannedMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(bannedMembership, nil).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "banned",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, community.ErrMemberBanned)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "pending request cannot request again",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidInviteOnlyGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				requestedMembership := testhelpers.ValidRequestedMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(requestedMembership, nil).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "already",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, community.ErrAlreadyGroupMember)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "group not found",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				groupID := testhelpers.ValidGroupIDParsed()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(nil, community.ErrGroupNotFound).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "find group",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find group")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "membership repository error",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, errors.New("database error")).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "check existing membership",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "check existing membership")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "save membership error",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.JoinGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, community.ErrMembershipNotFound).Once()
				suite.GroupRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.MembershipRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()

				return commands.JoinGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "save membership",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *community.GroupMembership, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "save membership")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			cmd := tt.setup(t, suite)

			handler := commands.NewJoinGroupHandler(
				suite.GroupRepo,
				suite.MembershipRepo,
				suite.EventPublisher,
				&suite.Logger,
			)

			result, err := handler.Handle(context.Background(), cmd)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
