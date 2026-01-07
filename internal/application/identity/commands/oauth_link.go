package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// LinkOAuthAccountCommand represents the intent to link an OAuth provider to an existing user account.
// This command requires the user to be authenticated.
//
// Flow:
//  1. Exchange OAuth code for access token
//  2. Retrieve user info from OAuth provider
//  3. Verify OAuth account is not already linked to another user
//  4. Create OAuth account link for current user
//
// Security considerations:
//   - User must be authenticated (userID from session)
//   - Cannot link if OAuth account is already linked to another user
//   - OAuth tokens are encrypted before storage
type LinkOAuthAccountCommand struct {
	UserID   domainidentity.UserID
	Provider domainidentity.OAuthProvider
	Code     string
	State    string // Used for CSRF protection (validated by caller)
}

// LinkOAuthAccountHandler processes OAuth account linking commands.
type LinkOAuthAccountHandler struct {
	users           domainidentity.UserRepository
	oauthAccounts   domainidentity.OAuthAccountRepository
	providerFactory OAuthProviderFactory
	logger          *zerolog.Logger
}

// NewLinkOAuthAccountHandler creates a new OAuth account linking handler.
func NewLinkOAuthAccountHandler(
	users domainidentity.UserRepository,
	oauthAccounts domainidentity.OAuthAccountRepository,
	providerFactory OAuthProviderFactory,
	logger *zerolog.Logger,
) *LinkOAuthAccountHandler {
	return &LinkOAuthAccountHandler{
		users:           users,
		oauthAccounts:   oauthAccounts,
		providerFactory: providerFactory,
		logger:          logger,
	}
}

// Handle executes the OAuth account linking use case.
//
// Process flow:
//  1. Verify user exists and is active
//  2. Create OAuth provider instance
//  3. Exchange authorization code for access token
//  4. Retrieve user info from provider
//  5. Check if OAuth account is already linked to any user
//  6. If already linked to current user: update tokens and profile
//  7. If not linked: create new OAuth account link
//  8. Return success message
//
// Returns:
//   - OAuthAccountDTO with linked account details on success
//   - Error if user not found, OAuth exchange fails, or account already linked to another user
func (h *LinkOAuthAccountHandler) Handle(
	ctx context.Context,
	cmd LinkOAuthAccountCommand,
) (*dto.OAuthAccountDTO, error) {
	// 1. Verify user exists and can perform this operation
	user, err := h.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	// User must be active to link OAuth accounts
	if !user.CanLogin() {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Str("status", user.Status().String()).
			Str("provider", cmd.Provider.String()).
			Msg("inactive user attempted to link OAuth account")
		return nil, fmt.Errorf("user account is not active")
	}

	// 2. Create OAuth provider instance
	provider, err := h.providerFactory.CreateProvider(cmd.Provider)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("provider", cmd.Provider.String()).
			Msg("failed to create OAuth provider")
		return nil, fmt.Errorf("create OAuth provider: %w", err)
	}

	// 3. Exchange authorization code for access token
	token, err := provider.ExchangeCode(ctx, cmd.Code)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("provider", cmd.Provider.String()).
			Msg("failed to exchange OAuth code")
		return nil, fmt.Errorf("exchange OAuth code: %w", err)
	}

	// 4. Retrieve user info from provider
	userInfo, err := provider.GetUserInfo(ctx, token)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("provider", cmd.Provider.String()).
			Msg("failed to get user info from OAuth provider")
		return nil, fmt.Errorf("get OAuth user info: %w", err)
	}

	// Validate required fields from provider
	if userInfo.ProviderUserID == "" {
		return nil, fmt.Errorf("OAuth provider returned empty user ID")
	}
	if userInfo.Email == "" {
		return nil, fmt.Errorf("OAuth provider returned empty email")
	}

	// Parse provider user ID
	providerUserID, err := domainidentity.NewProviderUserID(userInfo.ProviderUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider user ID: %w", err)
	}

	// Parse email from provider
	email, err := domainidentity.NewEmail(userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email from OAuth provider: %w", err)
	}

	// 5. Check if OAuth account is already linked
	existingAccount, err := h.oauthAccounts.FindByProviderAndUserID(ctx, cmd.Provider, providerUserID)
	if err != nil && !errors.Is(err, domainidentity.ErrOAuthAccountNotFound) {
		return nil, fmt.Errorf("find existing OAuth account: %w", err)
	}

	var oauthAccount *domainidentity.OAuthAccount

	if existingAccount != nil {
		// OAuth account already exists
		if !existingAccount.UserID().Equals(cmd.UserID) {
			// Linked to a different user - not allowed
			h.logger.Warn().
				Str("user_id", cmd.UserID.String()).
				Str("existing_user_id", existingAccount.UserID().String()).
				Str("provider", cmd.Provider.String()).
				Str("provider_user_id", providerUserID.String()).
				Msg("attempted to link OAuth account already linked to another user")
			return nil, domainidentity.ErrOAuthAccountExists
		}

		// Already linked to current user - update tokens and profile
		oauthAccount = existingAccount

		if err := oauthAccount.UpdateProfile(email, userInfo.DisplayName, userInfo.AvatarURL); err != nil {
			return nil, fmt.Errorf("update OAuth profile: %w", err)
		}

		// Encrypt and store new OAuth tokens
		if err := h.encryptAndStoreTokens(oauthAccount, token); err != nil {
			return nil, fmt.Errorf("encrypt OAuth tokens: %w", err)
		}

		h.logger.Info().
			Str("user_id", user.ID().String()).
			Str("oauth_account_id", oauthAccount.ID().String()).
			Str("provider", cmd.Provider.String()).
			Msg("updated existing OAuth account link")
	} else {
		// 7. Create new OAuth account link
		oauthAccount, err = h.createOAuthAccountLink(ctx, user.ID(), cmd.Provider, providerUserID, email, userInfo, token)
		if err != nil {
			return nil, fmt.Errorf("create OAuth account link: %w", err)
		}

		h.logger.Info().
			Str("user_id", user.ID().String()).
			Str("oauth_account_id", oauthAccount.ID().String()).
			Str("provider", cmd.Provider.String()).
			Msg("created new OAuth account link")
	}

	// Save OAuth account
	if err := h.oauthAccounts.Save(ctx, oauthAccount); err != nil {
		return nil, fmt.Errorf("save OAuth account: %w", err)
	}

	// 8. Return success response
	return &dto.OAuthAccountDTO{
		Provider:    oauthAccount.Provider().String(),
		Email:       oauthAccount.Email().String(),
		DisplayName: oauthAccount.DisplayName(),
		AvatarURL:   oauthAccount.AvatarURL(),
		LinkedAt:    oauthAccount.CreatedAt(),
	}, nil
}

// createOAuthAccountLink creates a new OAuth account link for the user.
func (h *LinkOAuthAccountHandler) createOAuthAccountLink(
	ctx context.Context,
	userID domainidentity.UserID,
	provider domainidentity.OAuthProvider,
	providerUserID domainidentity.ProviderUserID,
	email domainidentity.Email,
	userInfo *security.OAuthUserInfo,
	token *oauth2.Token,
) (*domainidentity.OAuthAccount, error) {
	// Encrypt OAuth tokens for storage
	encryptor := h.providerFactory.Encryptor()
	encryptedAccessToken, err := encryptor.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}

	var encryptedRefreshToken []byte
	if token.RefreshToken != "" {
		encryptedRefreshToken, err = encryptor.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	var tokenExpiresAt *time.Time
	if !token.Expiry.IsZero() {
		tokenExpiresAt = &token.Expiry
	}

	oauthAccount, err := domainidentity.NewOAuthAccount(
		userID,
		provider,
		providerUserID,
		email,
		userInfo.DisplayName,
		userInfo.AvatarURL,
		encryptedAccessToken,
		encryptedRefreshToken,
		tokenExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create OAuth account: %w", err)
	}

	return oauthAccount, nil
}

// encryptAndStoreTokens encrypts and stores OAuth tokens in the OAuth account.
func (h *LinkOAuthAccountHandler) encryptAndStoreTokens(
	account *domainidentity.OAuthAccount,
	token *oauth2.Token,
) error {
	encryptor := h.providerFactory.Encryptor()

	encryptedAccessToken, err := encryptor.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	var encryptedRefreshToken []byte
	if token.RefreshToken != "" {
		encryptedRefreshToken, err = encryptor.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	var expiresAt *time.Time
	if !token.Expiry.IsZero() {
		expiresAt = &token.Expiry
	}

	account.UpdateTokens(encryptedAccessToken, encryptedRefreshToken, expiresAt)
	return nil
}
