package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestResetPasswordHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   string
		newPass string
		setup   func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository)
		wantErr bool
	}{
		{
			name:    "valid token resets password",
			token:   "valid-uuid-token",
			newPass: "NewSecure@Pass123",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository) {
				user := testhelpers.ValidUser()
				expiresAt := time.Now().UTC().Add(time.Hour)
				prt := &identity.PasswordResetToken{
					Token:     "valid-uuid-token",
					UserID:    user.ID(),
					ExpiresAt: expiresAt,
				}

				tokenRepo.On("FindValidPasswordResetToken", mock.Anything, "valid-uuid-token").
					Return(prt, nil).Once()

				suite.UserRepo.On("FindByID", mock.Anything, user.ID()).
					Return(user, nil).Once()

				suite.UserRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).
					Return(nil).Once()

				tokenRepo.On("MarkPasswordResetTokenUsed", mock.Anything, "valid-uuid-token").
					Return(nil).Once()

				suite.SessionStore.On("RevokeAll", mock.Anything, user.ID().String()).
					Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name:    "invalid token returns error",
			token:   "nonexistent-token",
			newPass: "NewSecure@Pass123",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository) {
				tokenRepo.On("FindValidPasswordResetToken", mock.Anything, "nonexistent-token").
					Return(nil, identity.ErrTokenNotFound).Once()
			},
			wantErr: true,
		},
		{
			name:    "weak password returns error",
			token:   "valid-uuid-token",
			newPass: "weak",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository) {
				user := testhelpers.ValidUser()
				expiresAt := time.Now().UTC().Add(time.Hour)
				prt := &identity.PasswordResetToken{
					Token:     "valid-uuid-token",
					UserID:    user.ID(),
					ExpiresAt: expiresAt,
				}

				tokenRepo.On("FindValidPasswordResetToken", mock.Anything, "valid-uuid-token").
					Return(prt, nil).Once()
			},
			wantErr: true,
		},
		{
			name:    "user not found returns error",
			token:   "valid-uuid-token",
			newPass: "NewSecure@Pass123",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository) {
				user := testhelpers.ValidUser()
				expiresAt := time.Now().UTC().Add(time.Hour)
				prt := &identity.PasswordResetToken{
					Token:     "valid-uuid-token",
					UserID:    user.ID(),
					ExpiresAt: expiresAt,
				}

				tokenRepo.On("FindValidPasswordResetToken", mock.Anything, "valid-uuid-token").
					Return(prt, nil).Once()

				suite.UserRepo.On("FindByID", mock.Anything, user.ID()).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: true,
		},
		{
			name:    "token repository error propagates",
			token:   "valid-uuid-token",
			newPass: "NewSecure@Pass123",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository) {
				tokenRepo.On("FindValidPasswordResetToken", mock.Anything, "valid-uuid-token").
					Return(nil, errors.New("db error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tokenRepo := new(testhelpers.MockTokenRepository)

			if tt.setup != nil {
				tt.setup(t, suite, tokenRepo)
			}

			handler := commands.NewResetPasswordHandler(
				tokenRepo,
				suite.UserRepo,
				suite.SessionStore,
				suite.Logger,
			)

			err := handler.Handle(context.Background(), commands.ResetPasswordCommand{
				Token:       tt.token,
				NewPassword: tt.newPass,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			suite.UserRepo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
			suite.SessionStore.AssertExpectations(t)
		})
	}
}
