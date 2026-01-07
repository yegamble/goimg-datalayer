package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// OAuthProvider defines the interface for OAuth 2.0 provider implementations.
// This is a facade over infrastructure/security.OAuthProvider to avoid direct infrastructure dependencies.
type OAuthProvider interface {
	// GetAuthorizationURL generates the OAuth authorization URL with state parameter.
	// The state parameter should be cryptographically random for CSRF protection.
	GetAuthorizationURL(state string) string

	// ExchangeCode exchanges an authorization code for an access token.
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)

	// GetUserInfo retrieves user information using the access token.
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*security.OAuthUserInfo, error)
}

// OAuthProviderFactory creates OAuth provider instances.
type OAuthProviderFactory interface {
	// CreateProvider creates an OAuth provider for the given provider type.
	CreateProvider(providerType domainidentity.OAuthProvider) (OAuthProvider, error)

	// Encryptor returns the secret encryptor for encrypting OAuth tokens.
	Encryptor() TokenEncryptor
}

// TokenEncryptor encrypts and decrypts OAuth tokens for storage.
type TokenEncryptor interface {
	// Encrypt encrypts plaintext data.
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt decrypts encrypted data.
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AuthenticateWithOAuthCommand represents the intent to authenticate a user via OAuth.
// This command handles both new user registration and existing user login through OAuth providers.
//
// Flow:
//  1. Exchange OAuth code for access token
//  2. Retrieve user info from OAuth provider
//  3. Check if OAuth account exists in database
//  4. If exists: Load user, update OAuth profile, generate JWT tokens
//  5. If not exists: Create new user + OAuth account link, generate JWT tokens
//
// Security considerations:
//   - OAuth tokens are encrypted before storage using AES-256-GCM
//   - Email verification status from provider is tracked
//   - Provider user ID is the stable identifier (not email)
type AuthenticateWithOAuthCommand struct {
	Provider  domainidentity.OAuthProvider
	Code      string
	State     string // Used for CSRF protection (validated by caller)
	IPAddress string
	UserAgent string
}

// AuthenticateWithOAuthHandler processes OAuth authentication commands.
type AuthenticateWithOAuthHandler struct {
	users            domainidentity.UserRepository
	oauthAccounts    domainidentity.OAuthAccountRepository
	providerFactory  OAuthProviderFactory
	jwtService       services.JWTService
	refreshService   services.RefreshTokenService
	sessionStore     services.SessionStore
	logger           *zerolog.Logger
}

// NewAuthenticateWithOAuthHandler creates a new OAuth authentication handler.
func NewAuthenticateWithOAuthHandler(
	users domainidentity.UserRepository,
	oauthAccounts domainidentity.OAuthAccountRepository,
	providerFactory OAuthProviderFactory,
	jwtService services.JWTService,
	refreshService services.RefreshTokenService,
	sessionStore services.SessionStore,
	logger *zerolog.Logger,
) *AuthenticateWithOAuthHandler {
	return &AuthenticateWithOAuthHandler{
		users:           users,
		oauthAccounts:   oauthAccounts,
		providerFactory: providerFactory,
		jwtService:      jwtService,
		refreshService:  refreshService,
		sessionStore:    sessionStore,
		logger:          logger,
	}
}

// Handle executes the OAuth authentication use case.
//
// Process flow:
//  1. Create OAuth provider instance
//  2. Exchange authorization code for access token
//  3. Retrieve user info from provider
//  4. Check if OAuth account exists (by provider + provider user ID)
//  5. If exists: Load user, update OAuth profile
//  6. If not exists: Create user (or link to existing email), create OAuth account
//  7. Generate JWT tokens and create session
//  8. Return AuthResponseDTO with user data and tokens
//
// Returns:
//   - AuthResponseDTO with user data and token pair on success
//   - Error if OAuth exchange fails, user creation fails, or token generation fails
//
//nolint:funlen,cyclop // OAuth flow has many sequential steps: code exchange, user info, account lookup, user creation, token gen
func (h *AuthenticateWithOAuthHandler) Handle(
	ctx context.Context,
	cmd AuthenticateWithOAuthCommand,
) (*dto.AuthResponseDTO, error) {
	// 1. Create OAuth provider instance
	provider, err := h.providerFactory.CreateProvider(cmd.Provider)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("provider", cmd.Provider.String()).
			Msg("failed to create OAuth provider")
		return nil, fmt.Errorf("create OAuth provider: %w", err)
	}

	// 2. Exchange authorization code for access token
	token, err := provider.ExchangeCode(ctx, cmd.Code)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("provider", cmd.Provider.String()).
			Msg("failed to exchange OAuth code")
		return nil, fmt.Errorf("exchange OAuth code: %w", err)
	}

	// 3. Retrieve user info from provider
	userInfo, err := provider.GetUserInfo(ctx, token)
	if err != nil {
		h.logger.Error().
			Err(err).
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

	// 4. Check if OAuth account exists
	oauthAccount, err := h.oauthAccounts.FindByProviderAndUserID(ctx, cmd.Provider, providerUserID)
	var user *domainidentity.User

	if err != nil && !errors.Is(err, domainidentity.ErrOAuthAccountNotFound) {
		return nil, fmt.Errorf("find OAuth account: %w", err)
	}

	if oauthAccount != nil {
		// 5a. OAuth account exists - load user and update profile
		user, err = h.users.FindByID(ctx, oauthAccount.UserID())
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", oauthAccount.UserID().String()).
				Str("provider", cmd.Provider.String()).
				Msg("OAuth account exists but user not found")
			return nil, fmt.Errorf("find user for OAuth account: %w", err)
		}

		// Update OAuth profile data (email, display name, avatar may have changed)
		email, emailErr := domainidentity.NewEmail(userInfo.Email)
		if emailErr != nil {
			return nil, fmt.Errorf("invalid email from OAuth provider: %w", emailErr)
		}

		if err := oauthAccount.UpdateProfile(email, userInfo.DisplayName, userInfo.AvatarURL); err != nil {
			return nil, fmt.Errorf("update OAuth profile: %w", err)
		}

		// Encrypt and store new OAuth tokens
		if err := h.encryptAndStoreTokens(oauthAccount, token); err != nil {
			return nil, fmt.Errorf("encrypt OAuth tokens: %w", err)
		}

		// Save updated OAuth account
		if err := h.oauthAccounts.Save(ctx, oauthAccount); err != nil {
			return nil, fmt.Errorf("save OAuth account: %w", err)
		}

		h.logger.Info().
			Str("user_id", user.ID().String()).
			Str("email", user.Email().String()).
			Str("provider", cmd.Provider.String()).
			Msg("user authenticated via OAuth")
	} else {
		// 5b. OAuth account does not exist - create user or link to existing
		user, oauthAccount, err = h.createOrLinkUser(ctx, cmd.Provider, providerUserID, userInfo, token)
		if err != nil {
			return nil, fmt.Errorf("create or link OAuth user: %w", err)
		}
	}

	// 6. Check user status (suspended users cannot login)
	if !user.CanLogin() {
		h.logger.Warn().
			Str("user_id", user.ID().String()).
			Str("email", user.Email().String()).
			Str("status", user.Status().String()).
			Str("provider", cmd.Provider.String()).
			Msg("OAuth login attempt for account that cannot login")

		switch user.Status() {
		case domainidentity.StatusSuspended:
			return nil, identity.ErrAccountSuspended
		case domainidentity.StatusDeleted:
			return nil, identity.ErrAccountDeleted
		default:
			return nil, identity.ErrInvalidCredentials
		}
	}

	// 7. Generate session ID
	sessionID := uuid.New().String()

	// 8. Generate JWT access token (15 min TTL)
	accessToken, err := h.jwtService.GenerateAccessToken(
		user.ID().String(),
		user.Email().String(),
		user.Role().String(),
		sessionID,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate access token")
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 9. Generate refresh token with family ID (7 day TTL)
	familyID := uuid.New().String()
	refreshToken, metadata, err := h.refreshService.GenerateToken(
		ctx,
		user.ID().String(),
		sessionID,
		familyID,
		"", // No parent hash for first token in family
		cmd.IPAddress,
		cmd.UserAgent,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to generate refresh token")
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 10. Create session in Redis
	session := services.Session{
		SessionID: sessionID,
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Role:      user.Role().String(),
		IP:        cmd.IPAddress,
		UserAgent: cmd.UserAgent,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: metadata.ExpiresAt,
	}
	if err := h.sessionStore.Create(ctx, session); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("session_id", sessionID).
			Msg("failed to create session")
		return nil, fmt.Errorf("create session: %w", err)
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("email", user.Email().String()).
		Str("session_id", sessionID).
		Str("provider", cmd.Provider.String()).
		Str("ip_address", cmd.IPAddress).
		Str("user_agent", cmd.UserAgent).
		Msg("OAuth authentication successful")

	// 11. Build response with tokens and user data
	expiresAt, err := h.jwtService.GetTokenExpiration(accessToken)
	if err != nil {
		// Non-critical - use default
		expiresAt = time.Now().UTC().Add(defaultTokenExpirationMinutes * time.Minute)
	}

	tokens := dto.NewTokenPairDTO(accessToken, refreshToken, expiresAt)
	authResponse := dto.NewAuthResponseDTO(user, tokens)

	return &authResponse, nil
}

// createOrLinkUser creates a new user or links OAuth to existing user by email.
//
//nolint:funlen // User creation with OAuth has many steps
func (h *AuthenticateWithOAuthHandler) createOrLinkUser(
	ctx context.Context,
	provider domainidentity.OAuthProvider,
	providerUserID domainidentity.ProviderUserID,
	userInfo *security.OAuthUserInfo,
	token *oauth2.Token,
) (*domainidentity.User, *domainidentity.OAuthAccount, error) {
	// Parse email from provider
	email, err := domainidentity.NewEmail(userInfo.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid email from OAuth provider: %w", err)
	}

	// Check if user exists with this email
	existingUser, err := h.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("check existing user by email: %w", err)
	}

	var user *domainidentity.User

	if existingUser != nil {
		// User exists with this email - link OAuth account
		user = existingUser

		h.logger.Info().
			Str("user_id", user.ID().String()).
			Str("email", email.String()).
			Str("provider", provider.String()).
			Msg("linking OAuth account to existing user")
	} else {
		// Create new user account
		// Generate username from email or display name
		username := h.generateUsernameFromOAuth(ctx, userInfo)

		// OAuth users don't have a password (empty hash)
		// They can only login via OAuth until they set a password
		passwordHash := domainidentity.PasswordHash{}

		user, err = domainidentity.NewUser(email, username, passwordHash)
		if err != nil {
			return nil, nil, fmt.Errorf("create user from OAuth: %w", err)
		}

		// Set display name from OAuth if provided
		if userInfo.DisplayName != "" {
			_ = user.UpdateProfile(userInfo.DisplayName, "")
		}

		// Mark as active (OAuth providers verify emails)
		if userInfo.EmailVerified {
			user.Activate()
		}

		// Save new user
		if err := h.users.Save(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("save new OAuth user: %w", err)
		}

		h.logger.Info().
			Str("user_id", user.ID().String()).
			Str("email", email.String()).
			Str("username", username.String()).
			Str("provider", provider.String()).
			Msg("created new user via OAuth")
	}

	// Create OAuth account link
	var tokenExpiresAt *time.Time
	if token.Expiry.IsZero() {
		tokenExpiresAt = nil
	} else {
		tokenExpiresAt = &token.Expiry
	}

	// Encrypt OAuth tokens for storage
	encryptor := h.providerFactory.Encryptor()
	encryptedAccessToken, err := encryptor.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt access token: %w", err)
	}

	var encryptedRefreshToken []byte
	if token.RefreshToken != "" {
		encryptedRefreshToken, err = encryptor.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return nil, nil, fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	oauthAccount, err := domainidentity.NewOAuthAccount(
		user.ID(),
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
		return nil, nil, fmt.Errorf("create OAuth account: %w", err)
	}

	// Save OAuth account
	if err := h.oauthAccounts.Save(ctx, oauthAccount); err != nil {
		return nil, nil, fmt.Errorf("save OAuth account: %w", err)
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("oauth_account_id", oauthAccount.ID().String()).
		Str("provider", provider.String()).
		Msg("created OAuth account link")

	return user, oauthAccount, nil
}

// generateUsernameFromOAuth generates a unique username from OAuth user info.
func (h *AuthenticateWithOAuthHandler) generateUsernameFromOAuth(
	ctx context.Context,
	userInfo *security.OAuthUserInfo,
) domainidentity.Username {
	// Try display name first
	if userInfo.DisplayName != "" {
		baseUsername := sanitizeUsername(userInfo.DisplayName)
		if username := h.tryUsername(ctx, baseUsername); username != nil {
			return *username
		}
	}

	// Try email prefix
	if userInfo.Email != "" {
		emailPrefix := userInfo.Email[:len(userInfo.Email)-len(userInfo.Email[indexOf(userInfo.Email, "@"):])]
		baseUsername := sanitizeUsername(emailPrefix)
		if username := h.tryUsername(ctx, baseUsername); username != nil {
			return *username
		}
	}

	// Fallback: generate random username
	randomSuffix := uuid.New().String()[:8]
	baseUsername := "user_" + randomSuffix
	username, _ := domainidentity.NewUsername(baseUsername)
	return username
}

// tryUsername attempts to create a username, appending numbers if taken.
func (h *AuthenticateWithOAuthHandler) tryUsername(ctx context.Context, base string) *domainidentity.Username {
	for i := 0; i < 10; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i)
		}

		username, err := domainidentity.NewUsername(candidate)
		if err != nil {
			continue
		}

		// Check if username is taken
		_, err = h.users.FindByUsername(ctx, username)
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return &username
		}
	}

	return nil
}

// encryptAndStoreTokens encrypts and stores OAuth tokens in the OAuth account.
func (h *AuthenticateWithOAuthHandler) encryptAndStoreTokens(
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

// Helper functions

// sanitizeUsername removes invalid characters from a string to create a valid username.
func sanitizeUsername(s string) string {
	// Remove non-alphanumeric characters except underscore
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result += string(r)
		}
	}

	// Limit length
	if len(result) > 32 {
		result = result[:32]
	}

	// Ensure minimum length
	if len(result) < 3 {
		result = "user_" + result
	}

	return result
}

// indexOf returns the index of the first occurrence of substr in s.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
