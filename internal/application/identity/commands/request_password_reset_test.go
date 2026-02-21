package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestRequestPasswordResetHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		email   string
		setup   func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender)
		wantErr error
	}{
		{
			name:  "valid email sends reset token",
			email: testhelpers.ValidEmail,
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender) {
				emailVO, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUser()

				suite.UserRepo.On("FindByEmail", mock.Anything, emailVO).
					Return(user, nil).Once()

				tokenRepo.On("InvalidateAllPasswordResetTokens", mock.Anything, user.ID()).
					Return(nil).Once()

				tokenRepo.On("CreatePasswordResetToken", mock.Anything, user.ID(), mock.MatchedBy(func(exp time.Time) bool {
					return exp.After(time.Now().UTC())
				})).Return("test-token-uuid", nil).Once()

				emailSender.On("SendPasswordResetEmail", mock.Anything, testhelpers.ValidEmail, "test-token-uuid").
					Return(nil).Once()
			},
			wantErr: nil,
		},
		{
			name:  "non-existent email returns no error (anti-enumeration)",
			email: "nonexistent@example.com",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender) {
				emailVO, _ := identity.NewEmail("nonexistent@example.com")

				suite.UserRepo.On("FindByEmail", mock.Anything, emailVO).
					Return(nil, identity.ErrUserNotFound).Once()

				tokenRepo.AssertNotCalled(t, "CreatePasswordResetToken")
				emailSender.AssertNotCalled(t, "SendPasswordResetEmail")
			},
			wantErr: nil,
		},
		{
			name:  "smtp disabled logs warning but does not fail",
			email: testhelpers.ValidEmail,
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender) {
				emailVO, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUser()

				suite.UserRepo.On("FindByEmail", mock.Anything, emailVO).
					Return(user, nil).Once()

				tokenRepo.On("InvalidateAllPasswordResetTokens", mock.Anything, user.ID()).
					Return(nil).Once()

				tokenRepo.On("CreatePasswordResetToken", mock.Anything, user.ID(), mock.Anything).
					Return("test-token-uuid", nil).Once()

				emailSender.On("SendPasswordResetEmail", mock.Anything, testhelpers.ValidEmail, "test-token-uuid").
					Return(appidentity.ErrSMTPDisabled).Once()
			},
			wantErr: nil,
		},
		{
			name:  "invalid email format returns error",
			email: "not-an-email",
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender) {
			},
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:  "token repository error propagates",
			email: testhelpers.ValidEmail,
			setup: func(t *testing.T, suite *testhelpers.TestSuite, tokenRepo *testhelpers.MockTokenRepository, emailSender *testhelpers.MockEmailSender) {
				emailVO, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUser()

				suite.UserRepo.On("FindByEmail", mock.Anything, emailVO).
					Return(user, nil).Once()

				tokenRepo.On("InvalidateAllPasswordResetTokens", mock.Anything, user.ID()).
					Return(nil).Once()

				tokenRepo.On("CreatePasswordResetToken", mock.Anything, user.ID(), mock.Anything).
					Return("", errors.New("db error")).Once()
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tokenRepo := new(testhelpers.MockTokenRepository)
			emailSender := new(testhelpers.MockEmailSender)

			if tt.setup != nil {
				tt.setup(t, suite, tokenRepo, emailSender)
			}

			handler := commands.NewRequestPasswordResetHandler(
				suite.UserRepo,
				tokenRepo,
				emailSender,
				&suite.Logger,
			)

			err := handler.Handle(context.Background(), commands.RequestPasswordResetCommand{
				Email: tt.email,
			})

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, identity.ErrEmailInvalid) {
					assert.ErrorIs(t, err, identity.ErrEmailInvalid)
				}
			} else {
				require.NoError(t, err)
			}

			suite.UserRepo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
			emailSender.AssertExpectations(t)
		})
	}
}
