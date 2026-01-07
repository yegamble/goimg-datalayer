package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// OAuthProviderType represents the type of OAuth provider.
type OAuthProviderType string

// Supported OAuth provider types.
const (
	ProviderGoogle OAuthProviderType = "google"
	ProviderGitHub OAuthProviderType = "github"
)

// OAuthProviderConfig contains configuration for an OAuth provider.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OAuthUserInfo contains user information retrieved from an OAuth provider.
type OAuthUserInfo struct {
	ProviderUserID string // Stable user identifier from provider
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
}

// OAuthProvider defines the interface for OAuth 2.0 provider implementations.
//
// Security considerations:
// - State parameter must be validated to prevent CSRF attacks
// - Authorization code should be used immediately and only once
// - Token exchange must use HTTPS
// - User info should be fetched from authenticated endpoints only
type OAuthProvider interface {
	// GetAuthorizationURL generates the OAuth authorization URL with state parameter.
	// The state parameter should be cryptographically random and stored in session
	// to prevent CSRF attacks (see S12-OAUTH-001).
	GetAuthorizationURL(state string) string

	// ExchangeCode exchanges an authorization code for an access token.
	// This should be called from the OAuth callback handler.
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)

	// GetUserInfo retrieves user information using the access token.
	// Returns standardized user info regardless of provider.
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error)

	// RefreshToken refreshes an access token using a refresh token.
	// Returns nil if the provider doesn't support refresh tokens.
	RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error)

	// RevokeToken revokes an access or refresh token.
	// Not all providers support revocation.
	RevokeToken(ctx context.Context, token string) error
}

// GoogleOAuthProvider implements OAuth 2.0 for Google.
type GoogleOAuthProvider struct {
	config *oauth2.Config
	client *http.Client
}

// NewGoogleOAuthProvider creates a new Google OAuth provider.
//
// Required scopes:
// - openid: OpenID Connect authentication
// - profile: User's profile info (name, picture)
// - email: User's email address
func NewGoogleOAuthProvider(cfg OAuthProviderConfig) *GoogleOAuthProvider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{
			"openid",
			"profile",
			"email",
		}
	}

	return &GoogleOAuthProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the Google OAuth authorization URL.
func (p *GoogleOAuthProvider) GetAuthorizationURL(state string) string {
	return p.config.AuthCodeURL(state,
		oauth2.AccessTypeOffline, // Request refresh token
		oauth2.ApprovalForce,     // Force approval prompt to get refresh token
	)
}

// ExchangeCode exchanges the authorization code for an access token.
func (p *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange Google authorization code: %w", err)
	}
	return token, nil
}

// googleUserInfoResponse represents the Google UserInfo API response.
// See: https://developers.google.com/identity/openid-connect/openid-connect#an-id-tokens-payload
type googleUserInfoResponse struct {
	Sub           string `json:"sub"`            // Unique user ID (stable identifier)
	Email         string `json:"email"`          // Email address
	EmailVerified bool   `json:"email_verified"` // Whether email is verified
	Name          string `json:"name"`           // Full name
	Picture       string `json:"picture"`        // Profile picture URL
	GivenName     string `json:"given_name"`     // First name
	FamilyName    string `json:"family_name"`    // Last name
	Locale        string `json:"locale"`         // Locale preference
}

// GetUserInfo retrieves user information from Google's UserInfo endpoint.
func (p *GoogleOAuthProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := p.config.Client(ctx, token)
	client.Timeout = 10 * time.Second

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo googleUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	// Validate required fields
	if userInfo.Sub == "" {
		return nil, fmt.Errorf("Google user info missing required field: sub")
	}
	if userInfo.Email == "" {
		return nil, fmt.Errorf("Google user info missing required field: email")
	}

	return &OAuthUserInfo{
		ProviderUserID: userInfo.Sub,
		Email:          userInfo.Email,
		EmailVerified:  userInfo.EmailVerified,
		DisplayName:    userInfo.Name,
		AvatarURL:      userInfo.Picture,
	}, nil
}

// RefreshToken refreshes the access token using a refresh token.
func (p *GoogleOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Google token: %w", err)
	}

	return newToken, nil
}

// RevokeToken revokes a Google access or refresh token.
// See: https://developers.google.com/identity/protocols/oauth2/web-server#tokenrevoke
func (p *GoogleOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	revokeURL := fmt.Sprintf("https://oauth2.googleapis.com/revoke?token=%s", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke Google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GitHubOAuthProvider implements OAuth 2.0 for GitHub.
type GitHubOAuthProvider struct {
	config *oauth2.Config
	client *http.Client
}

// NewGitHubOAuthProvider creates a new GitHub OAuth provider.
//
// Required scopes:
// - user:email: Access to user's email addresses (required)
// - read:user: Read user profile data
func NewGitHubOAuthProvider(cfg OAuthProviderConfig) *GitHubOAuthProvider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{
			"user:email",
			"read:user",
		}
	}

	return &GitHubOAuthProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     github.Endpoint,
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the GitHub OAuth authorization URL.
func (p *GitHubOAuthProvider) GetAuthorizationURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// ExchangeCode exchanges the authorization code for an access token.
func (p *GitHubOAuthProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange GitHub authorization code: %w", err)
	}
	return token, nil
}

// githubUserResponse represents the GitHub User API response.
// See: https://docs.github.com/en/rest/users/users#get-the-authenticated-user
type githubUserResponse struct {
	ID        int64  `json:"id"`         // Unique user ID
	Login     string `json:"login"`      // Username
	Name      string `json:"name"`       // Full name (may be empty)
	Email     string `json:"email"`      // Primary email (may be null)
	AvatarURL string `json:"avatar_url"` // Profile picture URL
}

// githubEmailResponse represents a GitHub email address.
// See: https://docs.github.com/en/rest/users/emails#list-email-addresses-for-the-authenticated-user
type githubEmailResponse struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

// GetUserInfo retrieves user information from GitHub's API.
func (p *GitHubOAuthProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := p.config.Client(ctx, token)
	client.Timeout = 10 * time.Second

	// Fetch user profile
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub user request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user githubUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user: %w", err)
	}

	// Fetch email addresses (GitHub user.email may be null if private)
	email, emailVerified, err := p.fetchPrimaryEmail(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub email: %w", err)
	}

	// Use user.Email as fallback if emails endpoint doesn't return one
	if email == "" && user.Email != "" {
		email = user.Email
		emailVerified = false // We don't know if it's verified
	}

	if email == "" {
		return nil, fmt.Errorf("GitHub user has no email address")
	}

	// Use login as display name if name is empty
	displayName := user.Name
	if displayName == "" {
		displayName = user.Login
	}

	return &OAuthUserInfo{
		ProviderUserID: fmt.Sprintf("%d", user.ID), // Convert GitHub ID to string
		Email:          email,
		EmailVerified:  emailVerified,
		DisplayName:    displayName,
		AvatarURL:      user.AvatarURL,
	}, nil
}

// fetchPrimaryEmail retrieves the user's primary verified email from GitHub.
func (p *GitHubOAuthProvider) fetchPrimaryEmail(ctx context.Context, client *http.Client) (string, bool, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", false, fmt.Errorf("failed to fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("GitHub emails request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var emails []githubEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false, fmt.Errorf("failed to decode emails: %w", err)
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true, nil
		}
	}

	// Fallback: find any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, true, nil
		}
	}

	// Fallback: find primary email (even if not verified)
	for _, e := range emails {
		if e.Primary {
			return e.Email, false, nil
		}
	}

	// Last resort: return first email
	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}

	return "", false, fmt.Errorf("no email addresses found")
}

// RefreshToken refreshes the access token.
// Note: GitHub OAuth does not support refresh tokens as of 2024.
// Access tokens are long-lived (do not expire).
func (p *GitHubOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("GitHub OAuth does not support refresh tokens")
}

// RevokeToken revokes a GitHub access token.
// See: https://docs.github.com/en/rest/apps/oauth-applications#delete-an-app-authorization
func (p *GitHubOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	// GitHub requires basic auth with client credentials to revoke tokens
	revokeURL := fmt.Sprintf("https://api.github.com/applications/%s/token", p.config.ClientID)

	reqBody := strings.NewReader(fmt.Sprintf(`{"access_token":"%s"}`, token))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, revokeURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.SetBasicAuth(p.config.ClientID, p.config.ClientSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke GitHub token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// OAuthProviderFactory creates OAuth provider instances.
type OAuthProviderFactory struct {
	encryptor *SecretEncryptor
}

// NewOAuthProviderFactory creates a new OAuth provider factory.
func NewOAuthProviderFactory(encryptor *SecretEncryptor) *OAuthProviderFactory {
	return &OAuthProviderFactory{
		encryptor: encryptor,
	}
}

// CreateProvider creates an OAuth provider instance based on the provider type.
func (f *OAuthProviderFactory) CreateProvider(
	providerType identity.OAuthProvider,
	cfg OAuthProviderConfig,
) (OAuthProvider, error) {
	switch providerType {
	case identity.OAuthProviderGoogle:
		return NewGoogleOAuthProvider(cfg), nil
	case identity.OAuthProviderGitHub:
		return NewGitHubOAuthProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", providerType)
	}
}

// Encryptor returns the secret encryptor for encrypting/decrypting OAuth tokens.
func (f *OAuthProviderFactory) Encryptor() *SecretEncryptor {
	return f.encryptor
}
