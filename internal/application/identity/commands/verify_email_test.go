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

func TestSendVerificationEmailHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	userID := user.ID().String()

	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(user, nil)
	suite.TokenRepo.On("CreateEmailVerificationToken", mock.Anything, user.ID(), mock.AnythingOfType("time.Time")).
		Return("test-token-uuid", nil)
	suite.EmailSender.On("SendVerificationEmail", mock.Anything, user.Email().String(), "test-token-uuid").
		Return(nil)

	handler := commands.NewSendVerificationEmailHandler(
		suite.UserRepo,
		suite.TokenRepo,
		suite.EmailSender,
		suite.Logger,
	)

	err := handler.Handle(context.Background(), commands.SendVerificationEmailCommand{UserID: userID})

	require.NoError(t, err)
	suite.AssertExpectations()
}

func TestSendVerificationEmailHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).
		Return(nil, identity.ErrUserNotFound)

	handler := commands.NewSendVerificationEmailHandler(
		suite.UserRepo,
		suite.TokenRepo,
		suite.EmailSender,
		suite.Logger,
	)

	err := handler.Handle(context.Background(), commands.SendVerificationEmailCommand{UserID: user.ID().String()})

	require.Error(t, err)
	suite.TokenRepo.AssertNotCalled(t, "CreateEmailVerificationToken", mock.Anything, mock.Anything, mock.Anything)
}

func TestSendVerificationEmailHandler_Handle_AlreadyVerified(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	user.VerifyEmail()

	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(user, nil)

	handler := commands.NewSendVerificationEmailHandler(
		suite.UserRepo,
		suite.TokenRepo,
		suite.EmailSender,
		suite.Logger,
	)

	err := handler.Handle(context.Background(), commands.SendVerificationEmailCommand{UserID: user.ID().String()})

	require.NoError(t, err)
	suite.TokenRepo.AssertNotCalled(t, "CreateEmailVerificationToken", mock.Anything, mock.Anything, mock.Anything)
	suite.EmailSender.AssertNotCalled(t, "SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything)
}

func TestSendVerificationEmailHandler_Handle_EmailSendFailureSuppressed(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()

	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(user, nil)
	suite.TokenRepo.On("CreateEmailVerificationToken", mock.Anything, user.ID(), mock.AnythingOfType("time.Time")).
		Return("test-token-uuid", nil)
	suite.EmailSender.On("SendVerificationEmail", mock.Anything, user.Email().String(), "test-token-uuid").
		Return(errors.New("smtp unavailable"))

	handler := commands.NewSendVerificationEmailHandler(
		suite.UserRepo,
		suite.TokenRepo,
		suite.EmailSender,
		suite.Logger,
	)

	err := handler.Handle(context.Background(), commands.SendVerificationEmailCommand{UserID: user.ID().String()})

	require.NoError(t, err)
	suite.AssertExpectations()
}

func TestSendVerificationEmailHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	handler := commands.NewSendVerificationEmailHandler(
		suite.UserRepo,
		suite.TokenRepo,
		suite.EmailSender,
		suite.Logger,
	)

	err := handler.Handle(context.Background(), commands.SendVerificationEmailCommand{UserID: "not-a-uuid"})

	require.Error(t, err)
}

func TestVerifyEmailHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	tokenStr := "valid-token-uuid"
	evToken := &identity.EmailVerificationToken{
		ID:        "some-id",
		UserID:    user.ID(),
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	suite.TokenRepo.On("FindValidEmailVerificationToken", mock.Anything, tokenStr).Return(evToken, nil)
	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(user, nil)
	suite.UserRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
		return u.EmailVerified()
	})).Return(nil)
	suite.TokenRepo.On("MarkEmailVerificationTokenUsed", mock.Anything, tokenStr).Return(nil)

	handler := commands.NewVerifyEmailHandler(suite.TokenRepo, suite.UserRepo, suite.Logger)

	err := handler.Handle(context.Background(), commands.VerifyEmailCommand{Token: tokenStr})

	require.NoError(t, err)
	assert.True(t, user.EmailVerified())
	suite.AssertExpectations()
}

func TestVerifyEmailHandler_Handle_TokenNotFound(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	suite.TokenRepo.On("FindValidEmailVerificationToken", mock.Anything, "bad-token").
		Return(nil, identity.ErrTokenNotFound)

	handler := commands.NewVerifyEmailHandler(suite.TokenRepo, suite.UserRepo, suite.Logger)

	err := handler.Handle(context.Background(), commands.VerifyEmailCommand{Token: "bad-token"})

	require.Error(t, err)
	require.ErrorIs(t, err, identity.ErrTokenNotFound)
}

func TestVerifyEmailHandler_Handle_UserNotFound(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	tokenStr := "valid-token-uuid"
	evToken := &identity.EmailVerificationToken{
		ID:        "some-id",
		UserID:    user.ID(),
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	suite.TokenRepo.On("FindValidEmailVerificationToken", mock.Anything, tokenStr).Return(evToken, nil)
	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(nil, identity.ErrUserNotFound)

	handler := commands.NewVerifyEmailHandler(suite.TokenRepo, suite.UserRepo, suite.Logger)

	err := handler.Handle(context.Background(), commands.VerifyEmailCommand{Token: tokenStr})

	require.Error(t, err)
	suite.UserRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestVerifyEmailHandler_Handle_SaveFailure(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	user := testhelpers.ValidActiveUser()
	tokenStr := "valid-token-uuid"
	evToken := &identity.EmailVerificationToken{
		ID:        "some-id",
		UserID:    user.ID(),
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	suite.TokenRepo.On("FindValidEmailVerificationToken", mock.Anything, tokenStr).Return(evToken, nil)
	suite.UserRepo.On("FindByID", mock.Anything, user.ID()).Return(user, nil)
	suite.UserRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

	handler := commands.NewVerifyEmailHandler(suite.TokenRepo, suite.UserRepo, suite.Logger)

	err := handler.Handle(context.Background(), commands.VerifyEmailCommand{Token: tokenStr})

	require.Error(t, err)
	suite.TokenRepo.AssertNotCalled(t, "MarkEmailVerificationTokenUsed", mock.Anything, mock.Anything)
}
