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

func TestLeaveGroupHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, err error)
	}{
		{
			name: "successful leave from group",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				membership := testhelpers.ValidActiveMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(membership, nil).Once()
				suite.MembershipRepo.On("Delete", mock.Anything, membership.ID()).Return(nil).Once()
				suite.GroupRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.NoError(t, err)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "owner cannot leave group",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				ownerID := testhelpers.ValidOwnerIDParsed()
				ownerMembership := testhelpers.ValidOwnerMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, ownerID).Return(ownerMembership, nil).Once()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  ownerID,
				}
			},
			wantErr: "owner",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, community.ErrCannotLeaveAsOwner)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "group not found",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				groupID := testhelpers.ValidGroupIDParsed()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(nil, community.ErrGroupNotFound).Once()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "find group",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "find group")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "membership not found",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(nil, community.ErrMembershipNotFound).Once()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "find membership",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "find membership")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "delete membership error",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				membership := testhelpers.ValidActiveMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(membership, nil).Once()
				suite.MembershipRepo.On("Delete", mock.Anything, membership.ID()).Return(errors.New("database error")).Once()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "delete membership",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "delete membership")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "save group error",
			setup: func(t *testing.T, suite *testhelpers.TestSuite) commands.LeaveGroupCommand {
				group := testhelpers.ValidPublicGroup(t)
				groupID := group.ID()
				userID := testhelpers.ValidUserIDParsed()
				membership := testhelpers.ValidActiveMembership(t)

				suite.GroupRepo.On("FindByID", mock.Anything, groupID).Return(group, nil).Once()
				suite.MembershipRepo.On("FindByGroupAndUser", mock.Anything, groupID, userID).Return(membership, nil).Once()
				suite.MembershipRepo.On("Delete", mock.Anything, membership.ID()).Return(nil).Once()
				suite.GroupRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()

				return commands.LeaveGroupCommand{
					GroupID: groupID,
					UserID:  userID,
				}
			},
			wantErr: "save group",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save group")
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			cmd := tt.setup(t, suite)

			handler := commands.NewLeaveGroupHandler(
				suite.GroupRepo,
				suite.MembershipRepo,
				suite.EventPublisher,
				&suite.Logger,
			)

			err := handler.Handle(context.Background(), cmd)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, err)
		})
	}
}
