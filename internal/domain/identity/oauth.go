package identity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OAuthProvider represents a supported OAuth 2.0 provider.
type OAuthProvider string

// Supported OAuth providers.
const (
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderGitHub OAuthProvider = "github"
)

// AllOAuthProviders returns all supported OAuth providers.
func AllOAuthProviders() []OAuthProvider {
	return []OAuthProvider{
		OAuthProviderGoogle,
		OAuthProviderGitHub,
	}
}

// IsValid returns whether the provider is valid.
func (p OAuthProvider) IsValid() bool {
	switch p {
	case OAuthProviderGoogle, OAuthProviderGitHub:
		return true
	default:
		return false
	}
}

// String returns the string representation of the provider.
func (p OAuthProvider) String() string {
	return string(p)
}

// ParseOAuthProvider parses a string into an OAuthProvider.
func ParseOAuthProvider(s string) (OAuthProvider, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	provider := OAuthProvider(s)

	if !provider.IsValid() {
		return "", fmt.Errorf("invalid OAuth provider: %s", s)
	}

	return provider, nil
}

// OAuthAccountID is a value object representing a unique OAuth account identifier.
type OAuthAccountID struct {
	value uuid.UUID
}

// NewOAuthAccountID generates a new OAuth account ID.
func NewOAuthAccountID() OAuthAccountID {
	return OAuthAccountID{value: uuid.New()}
}

// ParseOAuthAccountID parses a string into an OAuthAccountID.
func ParseOAuthAccountID(s string) (OAuthAccountID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return OAuthAccountID{}, fmt.Errorf("invalid OAuth account id: %w", err)
	}
	return OAuthAccountID{value: id}, nil
}

// String returns the string representation of the OAuth account ID.
func (id OAuthAccountID) String() string {
	return id.value.String()
}

// IsZero returns whether this ID is the zero value.
func (id OAuthAccountID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns whether two OAuth account IDs are equal.
func (id OAuthAccountID) Equals(other OAuthAccountID) bool {
	return id.value == other.value
}

// ProviderUserID is a value object representing a user's ID from an OAuth provider.
// This is the stable identifier from the provider (e.g., Google's "sub" claim, GitHub's user ID).
type ProviderUserID struct {
	value string
}

// NewProviderUserID creates a new ProviderUserID.
func NewProviderUserID(id string) (ProviderUserID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ProviderUserID{}, ErrProviderUserIDEmpty
	}

	if len(id) > 255 {
		return ProviderUserID{}, ErrProviderUserIDTooLong
	}

	return ProviderUserID{value: id}, nil
}

// String returns the string representation of the provider user ID.
func (id ProviderUserID) String() string {
	return id.value
}

// IsEmpty returns whether the provider user ID is empty.
func (id ProviderUserID) IsEmpty() bool {
	return id.value == ""
}

// Equals returns whether two provider user IDs are equal.
func (id ProviderUserID) Equals(other ProviderUserID) bool {
	return id.value == other.value
}

// OAuthAccount is an entity representing a linked OAuth account for a user.
// It stores the relationship between a local user and their OAuth provider account.
//
// Security notes:
// - Access and refresh tokens are encrypted at rest using AES-256-GCM
// - Provider user ID is the stable identifier (never use email alone)
// - Email is stored for convenience but provider user ID is the primary key
// - Avatar URL is stored but should be validated before use
type OAuthAccount struct {
	id                    OAuthAccountID
	userID                UserID
	provider              OAuthProvider
	providerUserID        ProviderUserID
	email                 Email
	displayName           string
	avatarURL             string
	encryptedAccessToken  []byte // Optional: stored if we need to call provider APIs
	encryptedRefreshToken []byte // Optional: stored for token refresh
	tokenExpiresAt        *time.Time
	createdAt             time.Time
	updatedAt             time.Time
}

// OAuthAccount validation constants.
const (
	maxOAuthDisplayNameLength = 255
	maxOAuthAvatarURLLength   = 512
)

// NewOAuthAccount creates a new OAuth account link.
// Tokens are optional - pass nil if you don't need to store them for API access.
func NewOAuthAccount(
	userID UserID,
	provider OAuthProvider,
	providerUserID ProviderUserID,
	email Email,
	displayName string,
	avatarURL string,
	encryptedAccessToken []byte,
	encryptedRefreshToken []byte,
	tokenExpiresAt *time.Time,
) (*OAuthAccount, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	if !provider.IsValid() {
		return nil, fmt.Errorf("invalid OAuth provider")
	}

	if providerUserID.IsEmpty() {
		return nil, ErrProviderUserIDEmpty
	}

	if email.IsEmpty() {
		return nil, ErrEmailEmpty
	}

	if len(displayName) > maxOAuthDisplayNameLength {
		return nil, fmt.Errorf("display name exceeds %d characters", maxOAuthDisplayNameLength)
	}

	if len(avatarURL) > maxOAuthAvatarURLLength {
		return nil, fmt.Errorf("avatar URL exceeds %d characters", maxOAuthAvatarURLLength)
	}

	now := time.Now().UTC()

	return &OAuthAccount{
		id:                    NewOAuthAccountID(),
		userID:                userID,
		provider:              provider,
		providerUserID:        providerUserID,
		email:                 email,
		displayName:           displayName,
		avatarURL:             avatarURL,
		encryptedAccessToken:  encryptedAccessToken,
		encryptedRefreshToken: encryptedRefreshToken,
		tokenExpiresAt:        tokenExpiresAt,
		createdAt:             now,
		updatedAt:             now,
	}, nil
}

// ReconstructOAuthAccount reconstitutes an OAuth account from persistence.
// This should only be used by the repository layer when loading from storage.
func ReconstructOAuthAccount(
	id OAuthAccountID,
	userID UserID,
	provider OAuthProvider,
	providerUserID ProviderUserID,
	email Email,
	displayName string,
	avatarURL string,
	encryptedAccessToken []byte,
	encryptedRefreshToken []byte,
	tokenExpiresAt *time.Time,
	createdAt, updatedAt time.Time,
) *OAuthAccount {
	return &OAuthAccount{
		id:                    id,
		userID:                userID,
		provider:              provider,
		providerUserID:        providerUserID,
		email:                 email,
		displayName:           displayName,
		avatarURL:             avatarURL,
		encryptedAccessToken:  encryptedAccessToken,
		encryptedRefreshToken: encryptedRefreshToken,
		tokenExpiresAt:        tokenExpiresAt,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
	}
}

// ID returns the OAuth account ID.
func (a *OAuthAccount) ID() OAuthAccountID {
	return a.id
}

// UserID returns the local user ID this OAuth account is linked to.
func (a *OAuthAccount) UserID() UserID {
	return a.userID
}

// Provider returns the OAuth provider.
func (a *OAuthAccount) Provider() OAuthProvider {
	return a.provider
}

// ProviderUserID returns the provider's user identifier.
func (a *OAuthAccount) ProviderUserID() ProviderUserID {
	return a.providerUserID
}

// Email returns the email from the OAuth provider.
func (a *OAuthAccount) Email() Email {
	return a.email
}

// DisplayName returns the display name from the OAuth provider.
func (a *OAuthAccount) DisplayName() string {
	return a.displayName
}

// AvatarURL returns the avatar URL from the OAuth provider.
func (a *OAuthAccount) AvatarURL() string {
	return a.avatarURL
}

// EncryptedAccessToken returns the encrypted access token.
// This should only be used for persistence or decryption.
func (a *OAuthAccount) EncryptedAccessToken() []byte {
	return a.encryptedAccessToken
}

// EncryptedRefreshToken returns the encrypted refresh token.
// This should only be used for persistence or decryption.
func (a *OAuthAccount) EncryptedRefreshToken() []byte {
	return a.encryptedRefreshToken
}

// TokenExpiresAt returns when the access token expires.
func (a *OAuthAccount) TokenExpiresAt() *time.Time {
	return a.tokenExpiresAt
}

// CreatedAt returns when the OAuth account was created.
func (a *OAuthAccount) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt returns when the OAuth account was last updated.
func (a *OAuthAccount) UpdatedAt() time.Time {
	return a.updatedAt
}

// HasStoredTokens returns whether this OAuth account has stored tokens.
func (a *OAuthAccount) HasStoredTokens() bool {
	return len(a.encryptedAccessToken) > 0
}

// IsTokenExpired returns whether the stored access token is expired.
func (a *OAuthAccount) IsTokenExpired() bool {
	if a.tokenExpiresAt == nil {
		return true // No expiry info means we should refresh
	}
	return time.Now().UTC().After(*a.tokenExpiresAt)
}

// UpdateProfile updates the profile information from the OAuth provider.
// This should be called after each OAuth login to keep data fresh.
func (a *OAuthAccount) UpdateProfile(email Email, displayName, avatarURL string) error {
	if email.IsEmpty() {
		return ErrEmailEmpty
	}

	if len(displayName) > maxOAuthDisplayNameLength {
		return fmt.Errorf("display name exceeds %d characters", maxOAuthDisplayNameLength)
	}

	if len(avatarURL) > maxOAuthAvatarURLLength {
		return fmt.Errorf("avatar URL exceeds %d characters", maxOAuthAvatarURLLength)
	}

	a.email = email
	a.displayName = displayName
	a.avatarURL = avatarURL
	a.updatedAt = time.Now().UTC()

	return nil
}

// UpdateTokens updates the stored OAuth tokens.
// Pass nil for tokens you don't want to update.
func (a *OAuthAccount) UpdateTokens(
	encryptedAccessToken []byte,
	encryptedRefreshToken []byte,
	expiresAt *time.Time,
) {
	if len(encryptedAccessToken) > 0 {
		a.encryptedAccessToken = encryptedAccessToken
	}

	if len(encryptedRefreshToken) > 0 {
		a.encryptedRefreshToken = encryptedRefreshToken
	}

	a.tokenExpiresAt = expiresAt
	a.updatedAt = time.Now().UTC()
}

// ClearTokens removes stored tokens from the OAuth account.
// This is useful when tokens are no longer needed or have been revoked.
func (a *OAuthAccount) ClearTokens() {
	a.encryptedAccessToken = nil
	a.encryptedRefreshToken = nil
	a.tokenExpiresAt = nil
	a.updatedAt = time.Now().UTC()
}

// OAuthAccountRepository defines the interface for persisting and retrieving OAuth accounts.
// Implementations should be provided in the infrastructure layer.
type OAuthAccountRepository interface {
	// FindByID retrieves an OAuth account by its unique ID.
	FindByID(ctx context.Context, id OAuthAccountID) (*OAuthAccount, error)

	// FindByProviderAndUserID retrieves an OAuth account by provider and provider user ID.
	// This is the primary lookup method when a user authenticates via OAuth.
	FindByProviderAndUserID(ctx context.Context, provider OAuthProvider, providerUserID ProviderUserID) (*OAuthAccount, error)

	// FindByUserID retrieves all OAuth accounts for a user.
	FindByUserID(ctx context.Context, userID UserID) ([]*OAuthAccount, error)

	// FindByUserIDAndProvider retrieves a specific OAuth account for a user and provider.
	FindByUserIDAndProvider(ctx context.Context, userID UserID, provider OAuthProvider) (*OAuthAccount, error)

	// Save persists an OAuth account to the repository.
	// If the account already exists, it is updated; otherwise, it is created.
	Save(ctx context.Context, account *OAuthAccount) error

	// Delete removes an OAuth account from the repository.
	Delete(ctx context.Context, id OAuthAccountID) error

	// ExistsByProviderAndUserID checks if an OAuth account exists for the provider and provider user ID.
	ExistsByProviderAndUserID(ctx context.Context, provider OAuthProvider, providerUserID ProviderUserID) (bool, error)
}
