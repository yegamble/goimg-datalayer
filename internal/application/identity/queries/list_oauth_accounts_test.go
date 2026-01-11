package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestListOAuthAccountsHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	userID := identity.NewUserID()
	email1, _ := identity.NewEmail("google@example.com")
	email2, _ := identity.NewEmail("github@example.com")
	providerUserID1, _ := identity.NewProviderUserID("google-user-123")
	providerUserID2, _ := identity.NewProviderUserID("github-user-456")

	now := time.Now().UTC()

	// Create OAuth accounts
	account1 := identity.ReconstructOAuthAccount(
		identity.NewOAuthAccountID(),
		userID,
		identity.OAuthProviderGoogle,
		providerUserID1,
		email1,
		"Google User",
		"https://example.com/avatar1.jpg",
		nil, // encrypted tokens
		nil,
		nil,
		now.Add(-24*time.Hour),
		now.Add(-24*time.Hour),
	)

	account2 := identity.ReconstructOAuthAccount(
		identity.NewOAuthAccountID(),
		userID,
		identity.OAuthProviderGitHub,
		providerUserID2,
		email2,
		"GitHub User",
		"https://example.com/avatar2.jpg",
		nil,
		nil,
		nil,
		now.Add(-12*time.Hour),
		now.Add(-12*time.Hour),
	)

	accounts := []*identity.OAuthAccount{account1, account2}

	// Mock expectations
	mockOAuthRepo.On("FindByUserID", mock.Anything, userID).
		Return(accounts, nil)

	query := queries.ListOAuthAccountsQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Count)
	assert.Len(t, result.Accounts, 2)

	// Verify first account
	assert.Equal(t, "google", result.Accounts[0].Provider)
	assert.Equal(t, "google@example.com", result.Accounts[0].Email)
	assert.Equal(t, "Google User", result.Accounts[0].DisplayName)
	assert.Equal(t, "https://example.com/avatar1.jpg", result.Accounts[0].AvatarURL)

	// Verify second account
	assert.Equal(t, "github", result.Accounts[1].Provider)
	assert.Equal(t, "github@example.com", result.Accounts[1].Email)
	assert.Equal(t, "GitHub User", result.Accounts[1].DisplayName)
	assert.Equal(t, "https://example.com/avatar2.jpg", result.Accounts[1].AvatarURL)

	mockOAuthRepo.AssertExpectations(t)
}

func TestListOAuthAccountsHandler_Handle_EmptyList(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	userID := identity.NewUserID()

	// Mock expectations - user has no OAuth accounts
	mockOAuthRepo.On("FindByUserID", mock.Anything, userID).
		Return([]*identity.OAuthAccount{}, nil)

	query := queries.ListOAuthAccountsQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Count)
	assert.Empty(t, result.Accounts)

	mockOAuthRepo.AssertExpectations(t)
}

func TestListOAuthAccountsHandler_Handle_InvalidUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	query := queries.ListOAuthAccountsQuery{
		UserID: "invalid-uuid",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user id")

	mockOAuthRepo.AssertNotCalled(t, "FindByUserID", mock.Anything, mock.Anything)
}

func TestListOAuthAccountsHandler_Handle_RepositoryError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	userID := identity.NewUserID()

	// Mock expectations - repository returns error
	mockOAuthRepo.On("FindByUserID", mock.Anything, userID).
		Return(nil, assert.AnError)

	query := queries.ListOAuthAccountsQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find OAuth accounts for user")

	mockOAuthRepo.AssertExpectations(t)
}

func TestListOAuthAccountsHandler_Handle_SingleAccount(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("user@google.com")
	providerUserID, _ := identity.NewProviderUserID("google-123")

	now := time.Now().UTC()

	account := identity.ReconstructOAuthAccount(
		identity.NewOAuthAccountID(),
		userID,
		identity.OAuthProviderGoogle,
		providerUserID,
		email,
		"Single User",
		"",
		nil,
		nil,
		nil,
		now,
		now,
	)

	// Mock expectations
	mockOAuthRepo.On("FindByUserID", mock.Anything, userID).
		Return([]*identity.OAuthAccount{account}, nil)

	query := queries.ListOAuthAccountsQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Count)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, "google", result.Accounts[0].Provider)
	assert.Equal(t, "user@google.com", result.Accounts[0].Email)
	assert.Equal(t, "Single User", result.Accounts[0].DisplayName)
	assert.Empty(t, result.Accounts[0].AvatarURL)

	mockOAuthRepo.AssertExpectations(t)
}

func TestListOAuthAccountsHandler_Handle_EmptyAvatarURL(t *testing.T) {
	t.Parallel()

	// Arrange
	mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
	logger := zerolog.Nop()
	handler := queries.NewListOAuthAccountsHandler(mockOAuthRepo, &logger)

	userID := identity.NewUserID()
	email, _ := identity.NewEmail("user@github.com")
	providerUserID, _ := identity.NewProviderUserID("github-789")

	now := time.Now().UTC()

	// Create account with empty avatar URL
	account := identity.ReconstructOAuthAccount(
		identity.NewOAuthAccountID(),
		userID,
		identity.OAuthProviderGitHub,
		providerUserID,
		email,
		"No Avatar User",
		"", // Empty avatar URL
		nil,
		nil,
		nil,
		now,
		now,
	)

	// Mock expectations
	mockOAuthRepo.On("FindByUserID", mock.Anything, userID).
		Return([]*identity.OAuthAccount{account}, nil)

	query := queries.ListOAuthAccountsQuery{
		UserID: userID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Count)
	assert.Len(t, result.Accounts, 1)
	assert.Empty(t, result.Accounts[0].AvatarURL)

	mockOAuthRepo.AssertExpectations(t)
}
