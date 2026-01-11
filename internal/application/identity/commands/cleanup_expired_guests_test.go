package commands_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestCleanupExpiredGuestsHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.CleanupExpiredGuestsCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr bool
		assert  func(t *testing.T, result *commands.CleanupResult, err error)
	}{
		{
			name: "successful cleanup with multiple expired guests",
			cmd:  commands.CleanupExpiredGuestsCommand{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Create expired guest users
				guest1, err := identity.NewGuestUser("192.168.1.1")
				require.NoError(t, err)
				guest2, err := identity.NewGuestUser("192.168.1.2")
				require.NoError(t, err)

				expiredGuests := []*identity.User{guest1, guest2}

				// Mock finding expired guests
				suite.UserRepo.On("FindExpiredGuests",
					mock.Anything,
					mock.AnythingOfType("time.Time"),
					1000,
				).Return(expiredGuests, nil).Once()

				// Mock successful deletions
				suite.UserRepo.On("Delete", mock.Anything, guest1.ID()).
					Return(nil).Once()
				suite.UserRepo.On("Delete", mock.Anything, guest2.ID()).
					Return(nil).Once()
			},
			wantErr: false,
			assert: func(t *testing.T, result *commands.CleanupResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 2, result.DeletedCount)
				assert.Equal(t, 0, result.ErrorCount)
				assert.Greater(t, result.Duration, time.Duration(0))
			},
		},
		{
			name: "successful cleanup with no expired guests",
			cmd:  commands.CleanupExpiredGuestsCommand{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No expired guests found
				suite.UserRepo.On("FindExpiredGuests",
					mock.Anything,
					mock.AnythingOfType("time.Time"),
					1000,
				).Return([]*identity.User{}, nil).Once()
			},
			wantErr: false,
			assert: func(t *testing.T, result *commands.CleanupResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 0, result.DeletedCount)
				assert.Equal(t, 0, result.ErrorCount)
			},
		},
		{
			name: "error finding expired guests",
			cmd:  commands.CleanupExpiredGuestsCommand{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Simulate database error
				suite.UserRepo.On("FindExpiredGuests",
					mock.Anything,
					mock.AnythingOfType("time.Time"),
					1000,
				).Return(nil, fmt.Errorf("database connection error")).Once()
			},
			wantErr: true,
			assert: func(t *testing.T, result *commands.CleanupResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find expired guests")
			},
		},
		{
			name: "partial failure during deletion",
			cmd:  commands.CleanupExpiredGuestsCommand{},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Create expired guest users
				guest1, err := identity.NewGuestUser("192.168.1.1")
				require.NoError(t, err)
				guest2, err := identity.NewGuestUser("192.168.1.2")
				require.NoError(t, err)
				guest3, err := identity.NewGuestUser("192.168.1.3")
				require.NoError(t, err)

				expiredGuests := []*identity.User{guest1, guest2, guest3}

				// Mock finding expired guests
				suite.UserRepo.On("FindExpiredGuests",
					mock.Anything,
					mock.AnythingOfType("time.Time"),
					1000,
				).Return(expiredGuests, nil).Once()

				// First deletion succeeds
				suite.UserRepo.On("Delete", mock.Anything, guest1.ID()).
					Return(nil).Once()

				// Second deletion fails
				suite.UserRepo.On("Delete", mock.Anything, guest2.ID()).
					Return(fmt.Errorf("foreign key constraint violation")).Once()

				// Third deletion succeeds
				suite.UserRepo.On("Delete", mock.Anything, guest3.ID()).
					Return(nil).Once()
			},
			wantErr: false,
			assert: func(t *testing.T, result *commands.CleanupResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 2, result.DeletedCount, "should delete 2 guests successfully")
				assert.Equal(t, 1, result.ErrorCount, "should have 1 deletion error")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			suite := testhelpers.NewTestSuite(t)
			if tt.setup != nil {
				tt.setup(t, suite)
			}

			handler := commands.NewCleanupExpiredGuestsHandler(
				suite.UserRepo,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.assert != nil {
				tt.assert(t, result, err)
			}

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify all expectations were met
			suite.UserRepo.AssertExpectations(t)
		})
	}
}
