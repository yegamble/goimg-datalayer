package testhelpers

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

type MockFollowRepository struct {
	mock.Mock
}

func (m *MockFollowRepository) Save(ctx context.Context, follow *identity.Follow) error {
	args := m.Called(ctx, follow)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

func (m *MockFollowRepository) Delete(ctx context.Context, followerID, followedID identity.UserID) error {
	args := m.Called(ctx, followerID, followedID)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

func (m *MockFollowRepository) Exists(ctx context.Context, followerID, followedID identity.UserID) (bool, error) {
	args := m.Called(ctx, followerID, followedID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFollowRepository) FindFollowers(ctx context.Context, userID identity.UserID, limit, offset int) ([]*identity.Follow, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	var follows []*identity.Follow
	if args.Get(0) != nil {
		follows = args.Get(0).([]*identity.Follow)
	}
	return follows, args.Int(1), args.Error(2)
}

func (m *MockFollowRepository) FindFollowing(ctx context.Context, userID identity.UserID, limit, offset int) ([]*identity.Follow, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	var follows []*identity.Follow
	if args.Get(0) != nil {
		follows = args.Get(0).([]*identity.Follow)
	}
	return follows, args.Int(1), args.Error(2)
}

func (m *MockFollowRepository) CountFollowers(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockFollowRepository) CountFollowing(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

type MockOAuthAccountRepository struct {
	mock.Mock
}

func (m *MockOAuthAccountRepository) FindByID(ctx context.Context, id identity.OAuthAccountID) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, id)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByID: %w", err)
	}
	return account, nil
}

func (m *MockOAuthAccountRepository) FindByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, provider, providerUserID)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByProviderAndUserID: %w", err)
	}
	return account, nil
}

func (m *MockOAuthAccountRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]*identity.OAuthAccount, error) {
	args := m.Called(ctx, userID)
	var accounts []*identity.OAuthAccount
	if args.Get(0) != nil {
		accounts = args.Get(0).([]*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserID: %w", err)
	}
	return accounts, nil
}

func (m *MockOAuthAccountRepository) FindByUserIDAndProvider(
	ctx context.Context,
	userID identity.UserID,
	provider identity.OAuthProvider,
) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, userID, provider)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserIDAndProvider: %w", err)
	}
	return account, nil
}

func (m *MockOAuthAccountRepository) Save(ctx context.Context, account *identity.OAuthAccount) error {
	args := m.Called(ctx, account)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

func (m *MockOAuthAccountRepository) Delete(ctx context.Context, id identity.OAuthAccountID) error {
	args := m.Called(ctx, id)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

func (m *MockOAuthAccountRepository) ExistsByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (bool, error) {
	args := m.Called(ctx, provider, providerUserID)
	return args.Bool(0), args.Error(1)
}

type MockTOTPRepository struct {
	mock.Mock
}

func (m *MockTOTPRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*identity.TOTPSecret, error) {
	args := m.Called(ctx, userID)
	var secret *identity.TOTPSecret
	if args.Get(0) != nil {
		secret = args.Get(0).(*identity.TOTPSecret)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserID: %w", err)
	}
	return secret, nil
}

func (m *MockTOTPRepository) IsEnabled(ctx context.Context, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

type MockBackupCodeRepository struct {
	mock.Mock
}

func (m *MockBackupCodeRepository) CountUnused(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) CreatePasswordResetToken(ctx context.Context, userID identity.UserID, expiresAt time.Time) (string, error) {
	args := m.Called(ctx, userID, expiresAt)
	return args.String(0), args.Error(1)
}

func (m *MockTokenRepository) FindValidPasswordResetToken(ctx context.Context, token string) (*identity.PasswordResetToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.PasswordResetToken), args.Error(1)
}

func (m *MockTokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) InvalidateAllPasswordResetTokens(ctx context.Context, userID identity.UserID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTokenRepository) CreateEmailVerificationToken(ctx context.Context, userID identity.UserID, expiresAt time.Time) (string, error) {
	args := m.Called(ctx, userID, expiresAt)
	return args.String(0), args.Error(1)
}

func (m *MockTokenRepository) FindValidEmailVerificationToken(ctx context.Context, token string) (*identity.EmailVerificationToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.EmailVerificationToken), args.Error(1)
}

func (m *MockTokenRepository) MarkEmailVerificationTokenUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendPasswordResetEmail(ctx context.Context, email, token string) error {
	args := m.Called(ctx, email, token)
	return args.Error(0)
}

func (m *MockEmailSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	args := m.Called(ctx, email, token)
	return args.Error(0)
}

var _ appidentity.EmailSender = (*MockEmailSender)(nil)
