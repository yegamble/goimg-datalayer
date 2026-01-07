package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UnlinkOAuthAccountCommand represents the intent to unlink an OAuth provider from a user account.
// This command requires the user to be authenticated.
//
// Flow:
//  1. Find OAuth account by user ID and provider
//  2. Verify user still has other authentication methods (password or other OAuth accounts)
//  3. Delete OAuth account link
//
// Security considerations:
//   - User must be authenticated (userID from session)
//   - Cannot unlink last authentication method (user must have password or other OAuth)
type UnlinkOAuthAccountCommand struct {
	UserID   domainidentity.UserID
	Provider domainidentity.OAuthProvider
}

// UnlinkOAuthAccountHandler processes OAuth account unlinking commands.
type UnlinkOAuthAccountHandler struct {
	users         domainidentity.UserRepository
	oauthAccounts domainidentity.OAuthAccountRepository
	logger        *zerolog.Logger
}

// NewUnlinkOAuthAccountHandler creates a new OAuth account unlinking handler.
func NewUnlinkOAuthAccountHandler(
	users domainidentity.UserRepository,
	oauthAccounts domainidentity.OAuthAccountRepository,
	logger *zerolog.Logger,
) *UnlinkOAuthAccountHandler {
	return &UnlinkOAuthAccountHandler{
		users:         users,
		oauthAccounts: oauthAccounts,
		logger:        logger,
	}
}

// Handle executes the OAuth account unlinking use case.
//
// Process flow:
//  1. Verify user exists and is active
//  2. Find OAuth account by user ID and provider
//  3. Check user has other authentication methods (password or other OAuth accounts)
//  4. Delete OAuth account link
//  5. Return success message
//
// Returns:
//   - Success message on successful unlinking
//   - Error if user not found, OAuth account not found, or this is the last auth method
func (h *UnlinkOAuthAccountHandler) Handle(
	ctx context.Context,
	cmd UnlinkOAuthAccountCommand,
) (*dto.MessageDTO, error) {
	// 1. Verify user exists
	user, err := h.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 2. Find OAuth account by user ID and provider
	oauthAccount, err := h.oauthAccounts.FindByUserIDAndProvider(ctx, cmd.UserID, cmd.Provider)
	if err != nil {
		if errors.Is(err, domainidentity.ErrOAuthProviderNotLinked) {
			return nil, fmt.Errorf("OAuth provider not linked: %w", err)
		}
		return nil, fmt.Errorf("find OAuth account: %w", err)
	}

	// 3. Check user has other authentication methods
	canUnlink, err := h.canUnlinkOAuthAccount(ctx, user, cmd.Provider)
	if err != nil {
		return nil, fmt.Errorf("check if OAuth account can be unlinked: %w", err)
	}

	if !canUnlink {
		h.logger.Warn().
			Str("user_id", cmd.UserID.String()).
			Str("provider", cmd.Provider.String()).
			Msg("attempted to unlink last authentication method")
		return nil, fmt.Errorf("cannot unlink last authentication method")
	}

	// 4. Delete OAuth account link
	if err := h.oauthAccounts.Delete(ctx, oauthAccount.ID()); err != nil {
		return nil, fmt.Errorf("delete OAuth account: %w", err)
	}

	h.logger.Info().
		Str("user_id", cmd.UserID.String()).
		Str("oauth_account_id", oauthAccount.ID().String()).
		Str("provider", cmd.Provider.String()).
		Msg("unlinked OAuth account")

	// 5. Return success response
	message := fmt.Sprintf("OAuth account for %s has been unlinked successfully", cmd.Provider.String())
	return &dto.MessageDTO{Message: message}, nil
}

// canUnlinkOAuthAccount checks if the user can safely unlink the OAuth account.
// Returns true if the user has a password or at least one other OAuth account.
func (h *UnlinkOAuthAccountHandler) canUnlinkOAuthAccount(
	ctx context.Context,
	user *domainidentity.User,
	provider domainidentity.OAuthProvider,
) (bool, error) {
	// Check if user has a password set
	if !user.PasswordHash().IsEmpty() {
		// User can login with password after unlinking OAuth
		return true, nil
	}

	// User doesn't have a password - check for other OAuth accounts
	allOAuthAccounts, err := h.oauthAccounts.FindByUserID(ctx, user.ID())
	if err != nil {
		return false, fmt.Errorf("find user OAuth accounts: %w", err)
	}

	// Count OAuth accounts excluding the one being unlinked
	otherOAuthCount := 0
	for _, account := range allOAuthAccounts {
		if account.Provider() != provider {
			otherOAuthCount++
		}
	}

	// User needs at least one other OAuth account to unlink this one
	return otherOAuthCount > 0, nil
}
