package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// ListOAuthAccountsQuery represents a request to list all OAuth accounts for a user.
type ListOAuthAccountsQuery struct {
	UserID string
}

// Implement Query interface.
func (ListOAuthAccountsQuery) isQuery() {}

// ListOAuthAccountsHandler handles queries to list OAuth accounts for a user.
type ListOAuthAccountsHandler struct {
	oauthAccounts domainidentity.OAuthAccountRepository
	logger        *zerolog.Logger
}

// NewListOAuthAccountsHandler creates a new ListOAuthAccountsHandler.
func NewListOAuthAccountsHandler(
	oauthAccounts domainidentity.OAuthAccountRepository,
	logger *zerolog.Logger,
) *ListOAuthAccountsHandler {
	return &ListOAuthAccountsHandler{
		oauthAccounts: oauthAccounts,
		logger:        logger,
	}
}

// Handle executes the query to list OAuth accounts for a user.
//
// Process flow:
//  1. Parse user ID
//  2. Retrieve all OAuth accounts for the user
//  3. Convert to DTOs
//  4. Return list
//
// Returns:
//   - OAuthAccountListDTO with all linked OAuth accounts
//   - Error if user ID is invalid or query fails
func (h *ListOAuthAccountsHandler) Handle(
	ctx context.Context,
	q ListOAuthAccountsQuery,
) (*dto.OAuthAccountListDTO, error) {
	// 1. Parse user ID
	userID, err := domainidentity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Retrieve all OAuth accounts for the user
	accounts, err := h.oauthAccounts.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find OAuth accounts for user: %w", err)
	}

	// 3. Convert to DTOs
	accountDTOs := make([]dto.OAuthAccountDTO, len(accounts))
	for i, account := range accounts {
		accountDTOs[i] = dto.OAuthAccountDTO{
			Provider:    account.Provider().String(),
			Email:       account.Email().String(),
			DisplayName: account.DisplayName(),
			AvatarURL:   account.AvatarURL(),
			LinkedAt:    account.CreatedAt(),
		}
	}

	// 4. Return list
	return &dto.OAuthAccountListDTO{
		Accounts: accountDTOs,
		Count:    len(accountDTOs),
	}, nil
}
