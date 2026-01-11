package identity

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllOAuthProviders(t *testing.T) {
	t.Parallel()

	providers := AllOAuthProviders()

	assert.Len(t, providers, 2)
	assert.Contains(t, providers, OAuthProviderGoogle)
	assert.Contains(t, providers, OAuthProviderGitHub)
}

func TestOAuthProvider_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider OAuthProvider
		want     bool
	}{
		{
			name:     "google is valid",
			provider: OAuthProviderGoogle,
			want:     true,
		},
		{
			name:     "github is valid",
			provider: OAuthProviderGitHub,
			want:     true,
		},
		{
			name:     "empty is invalid",
			provider: OAuthProvider(""),
			want:     false,
		},
		{
			name:     "random string is invalid",
			provider: OAuthProvider("facebook"),
			want:     false,
		},
		{
			name:     "uppercase is invalid",
			provider: OAuthProvider("GOOGLE"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.provider.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOAuthProvider_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider OAuthProvider
		want     string
	}{
		{
			name:     "google",
			provider: OAuthProviderGoogle,
			want:     "google",
		},
		{
			name:     "github",
			provider: OAuthProviderGitHub,
			want:     "github",
		},
		{
			name:     "empty",
			provider: OAuthProvider(""),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.provider.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseOAuthProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    OAuthProvider
		wantErr bool
	}{
		{
			name:    "valid google",
			input:   "google",
			want:    OAuthProviderGoogle,
			wantErr: false,
		},
		{
			name:    "valid github",
			input:   "github",
			want:    OAuthProviderGitHub,
			wantErr: false,
		},
		{
			name:    "uppercase normalized",
			input:   "GOOGLE",
			want:    OAuthProviderGoogle,
			wantErr: false,
		},
		{
			name:    "mixed case normalized",
			input:   "GitHub",
			want:    OAuthProviderGitHub,
			wantErr: false,
		},
		{
			name:    "with whitespace trimmed",
			input:   "  google  ",
			want:    OAuthProviderGoogle,
			wantErr: false,
		},
		{
			name:    "invalid provider",
			input:   "facebook",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseOAuthProvider(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid OAuth provider")
				assert.Equal(t, OAuthProvider(""), got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewOAuthAccountID(t *testing.T) {
	t.Parallel()

	id := NewOAuthAccountID()

	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
	// UUID should be 36 characters (with dashes)
	assert.Len(t, id.String(), 36)
}

func TestParseOAuthAccountID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := ParseOAuthAccountID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
			}
		})
	}
}

func TestOAuthAccountID_String(t *testing.T) {
	t.Parallel()

	id := NewOAuthAccountID()
	str := id.String()

	// Should be valid UUID format
	_, err := uuid.Parse(str)
	assert.NoError(t, err)
}

func TestOAuthAccountID_IsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   OAuthAccountID
		want bool
	}{
		{
			name: "newly generated ID is not zero",
			id:   NewOAuthAccountID(),
			want: false,
		},
		{
			name: "zero value ID is zero",
			id:   OAuthAccountID{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.id.IsZero()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOAuthAccountID_Equals(t *testing.T) {
	t.Parallel()

	id1 := NewOAuthAccountID()
	id2 := NewOAuthAccountID()
	id1Copy, _ := ParseOAuthAccountID(id1.String())

	t.Run("same ID equals itself", func(t *testing.T) {
		assert.True(t, id1.Equals(id1))
	})

	t.Run("parsed copy equals original", func(t *testing.T) {
		assert.True(t, id1.Equals(id1Copy))
	})

	t.Run("different IDs are not equal", func(t *testing.T) {
		assert.False(t, id1.Equals(id2))
	})
}

func TestNewProviderUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid provider user ID",
			input:   "123456789",
			wantErr: nil,
		},
		{
			name:    "valid with text",
			input:   "user_abc123",
			wantErr: nil,
		},
		{
			name:    "whitespace trimmed",
			input:   "  user123  ",
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrProviderUserIDEmpty,
		},
		{
			name:    "only whitespace",
			input:   "   ",
			wantErr: ErrProviderUserIDEmpty,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 256),
			wantErr: ErrProviderUserIDTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := NewProviderUserID(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, id.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.input), id.String())
			}
		})
	}
}

func TestProviderUserID_String(t *testing.T) {
	t.Parallel()

	id, err := NewProviderUserID("test123")
	require.NoError(t, err)

	assert.Equal(t, "test123", id.String())
}

func TestProviderUserID_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   ProviderUserID
		want bool
	}{
		{
			name: "non-empty ID",
			id:   ProviderUserID{value: "123"},
			want: false,
		},
		{
			name: "empty ID",
			id:   ProviderUserID{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.id.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderUserID_Equals(t *testing.T) {
	t.Parallel()

	id1, _ := NewProviderUserID("user123")
	id2, _ := NewProviderUserID("user123")
	id3, _ := NewProviderUserID("user456")

	t.Run("same value equals", func(t *testing.T) {
		assert.True(t, id1.Equals(id2))
	})

	t.Run("different values not equal", func(t *testing.T) {
		assert.False(t, id1.Equals(id3))
	})
}

func TestNewOAuthAccount(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("google-user-123")
	email, _ := NewEmail("user@example.com")
	displayName := "Test User"
	avatarURL := "https://example.com/avatar.jpg"
	encryptedAccessToken := []byte("encrypted-access-token")
	encryptedRefreshToken := []byte("encrypted-refresh-token")
	expiresAt := time.Now().Add(1 * time.Hour).UTC()

	t.Run("valid OAuth account", func(t *testing.T) {
		t.Parallel()

		account, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			email,
			displayName,
			avatarURL,
			encryptedAccessToken,
			encryptedRefreshToken,
			&expiresAt,
		)

		require.NoError(t, err)
		assert.False(t, account.ID().IsZero())
		assert.Equal(t, userID, account.UserID())
		assert.Equal(t, provider, account.Provider())
		assert.Equal(t, providerUserID, account.ProviderUserID())
		assert.Equal(t, email, account.Email())
		assert.Equal(t, displayName, account.DisplayName())
		assert.Equal(t, avatarURL, account.AvatarURL())
		assert.Equal(t, encryptedAccessToken, account.EncryptedAccessToken())
		assert.Equal(t, encryptedRefreshToken, account.EncryptedRefreshToken())
		assert.Equal(t, &expiresAt, account.TokenExpiresAt())
		assert.False(t, account.CreatedAt().IsZero())
		assert.False(t, account.UpdatedAt().IsZero())
	})

	t.Run("zero user ID fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			UserID{},
			provider,
			providerUserID,
			email,
			displayName,
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is required")
	})

	t.Run("invalid provider fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			userID,
			OAuthProvider("invalid"),
			providerUserID,
			email,
			displayName,
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid OAuth provider")
	})

	t.Run("empty provider user ID fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			userID,
			provider,
			ProviderUserID{},
			email,
			displayName,
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.ErrorIs(t, err, ErrProviderUserIDEmpty)
	})

	t.Run("empty email fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			Email{},
			displayName,
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.ErrorIs(t, err, ErrEmailEmpty)
	})

	t.Run("display name too long fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			email,
			strings.Repeat("a", 256),
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "display name exceeds")
	})

	t.Run("avatar URL too long fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			email,
			displayName,
			"https://example.com/"+strings.Repeat("a", 512),
			nil,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "avatar URL exceeds")
	})

	t.Run("nil tokens are allowed", func(t *testing.T) {
		t.Parallel()

		account, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			email,
			displayName,
			avatarURL,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Nil(t, account.EncryptedAccessToken())
		assert.Nil(t, account.EncryptedRefreshToken())
		assert.Nil(t, account.TokenExpiresAt())
		assert.False(t, account.HasStoredTokens())
	})
}

func TestReconstructOAuthAccount(t *testing.T) {
	t.Parallel()

	id := NewOAuthAccountID()
	userID := NewUserID()
	provider := OAuthProviderGitHub
	providerUserID, _ := NewProviderUserID("github-user-456")
	email, _ := NewEmail("github@example.com")
	displayName := "GitHub User"
	avatarURL := "https://github.com/avatar.png"
	encryptedAccessToken := []byte("access")
	encryptedRefreshToken := []byte("refresh")
	expiresAt := time.Now().Add(2 * time.Hour).UTC()
	createdAt := time.Now().Add(-24 * time.Hour).UTC()
	updatedAt := time.Now().UTC()

	account := ReconstructOAuthAccount(
		id,
		userID,
		provider,
		providerUserID,
		email,
		displayName,
		avatarURL,
		encryptedAccessToken,
		encryptedRefreshToken,
		&expiresAt,
		createdAt,
		updatedAt,
	)

	assert.Equal(t, id, account.ID())
	assert.Equal(t, userID, account.UserID())
	assert.Equal(t, provider, account.Provider())
	assert.Equal(t, providerUserID, account.ProviderUserID())
	assert.Equal(t, email, account.Email())
	assert.Equal(t, displayName, account.DisplayName())
	assert.Equal(t, avatarURL, account.AvatarURL())
	assert.Equal(t, encryptedAccessToken, account.EncryptedAccessToken())
	assert.Equal(t, encryptedRefreshToken, account.EncryptedRefreshToken())
	assert.Equal(t, &expiresAt, account.TokenExpiresAt())
	assert.Equal(t, createdAt, account.CreatedAt())
	assert.Equal(t, updatedAt, account.UpdatedAt())
}

func TestOAuthAccount_HasStoredTokens(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("user123")
	email, _ := NewEmail("user@example.com")

	tests := []struct {
		name         string
		accessToken  []byte
		refreshToken []byte
		want         bool
	}{
		{
			name:         "has access token",
			accessToken:  []byte("token"),
			refreshToken: nil,
			want:         true,
		},
		{
			name:         "has refresh token",
			accessToken:  nil,
			refreshToken: []byte("token"),
			want:         false, // Only access token counts
		},
		{
			name:         "has both tokens",
			accessToken:  []byte("access"),
			refreshToken: []byte("refresh"),
			want:         true,
		},
		{
			name:         "no tokens",
			accessToken:  nil,
			refreshToken: nil,
			want:         false,
		},
		{
			name:         "empty access token",
			accessToken:  []byte{},
			refreshToken: nil,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account, _ := NewOAuthAccount(
				userID,
				provider,
				providerUserID,
				email,
				"Name",
				"url",
				tt.accessToken,
				tt.refreshToken,
				nil,
			)

			assert.Equal(t, tt.want, account.HasStoredTokens())
		})
	}
}

func TestOAuthAccount_IsTokenExpired(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("user123")
	email, _ := NewEmail("user@example.com")

	t.Run("nil expiry is expired", func(t *testing.T) {
		t.Parallel()

		account, _ := NewOAuthAccount(
			userID, provider, providerUserID, email, "Name", "url",
			[]byte("token"), nil, nil,
		)

		assert.True(t, account.IsTokenExpired())
	})

	t.Run("future expiry is not expired", func(t *testing.T) {
		t.Parallel()

		future := time.Now().Add(1 * time.Hour).UTC()
		account, _ := NewOAuthAccount(
			userID, provider, providerUserID, email, "Name", "url",
			[]byte("token"), nil, &future,
		)

		assert.False(t, account.IsTokenExpired())
	})

	t.Run("past expiry is expired", func(t *testing.T) {
		t.Parallel()

		past := time.Now().Add(-1 * time.Hour).UTC()
		account, _ := NewOAuthAccount(
			userID, provider, providerUserID, email, "Name", "url",
			[]byte("token"), nil, &past,
		)

		assert.True(t, account.IsTokenExpired())
	})
}

func TestOAuthAccount_UpdateProfile(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("user123")
	email, _ := NewEmail("old@example.com")

	account, _ := NewOAuthAccount(
		userID, provider, providerUserID, email, "Old Name", "old-url",
		nil, nil, nil,
	)

	t.Run("successful profile update", func(t *testing.T) {
		newEmail, _ := NewEmail("new@example.com")
		newDisplayName := "New Name"
		newAvatarURL := "new-avatar-url"

		oldUpdatedAt := account.UpdatedAt()
		time.Sleep(10 * time.Millisecond)

		err := account.UpdateProfile(newEmail, newDisplayName, newAvatarURL)
		require.NoError(t, err)

		assert.Equal(t, newEmail, account.Email())
		assert.Equal(t, newDisplayName, account.DisplayName())
		assert.Equal(t, newAvatarURL, account.AvatarURL())
		assert.True(t, account.UpdatedAt().After(oldUpdatedAt))
	})

	t.Run("empty email fails", func(t *testing.T) {
		err := account.UpdateProfile(Email{}, "Name", "url")
		require.ErrorIs(t, err, ErrEmailEmpty)
	})

	t.Run("display name too long fails", func(t *testing.T) {
		email, _ := NewEmail("user@example.com")
		err := account.UpdateProfile(email, strings.Repeat("a", 256), "url")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "display name exceeds")
	})

	t.Run("avatar URL too long fails", func(t *testing.T) {
		email, _ := NewEmail("user@example.com")
		err := account.UpdateProfile(email, "Name", strings.Repeat("a", 513))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "avatar URL exceeds")
	})
}

func TestOAuthAccount_UpdateTokens(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("user123")
	email, _ := NewEmail("user@example.com")

	account, _ := NewOAuthAccount(
		userID, provider, providerUserID, email, "Name", "url",
		nil, nil, nil,
	)

	t.Run("update both tokens", func(t *testing.T) {
		newAccess := []byte("new-access")
		newRefresh := []byte("new-refresh")
		newExpiry := time.Now().Add(1 * time.Hour).UTC()

		oldUpdatedAt := account.UpdatedAt()
		time.Sleep(10 * time.Millisecond)

		account.UpdateTokens(newAccess, newRefresh, &newExpiry)

		assert.Equal(t, newAccess, account.EncryptedAccessToken())
		assert.Equal(t, newRefresh, account.EncryptedRefreshToken())
		assert.Equal(t, &newExpiry, account.TokenExpiresAt())
		assert.True(t, account.UpdatedAt().After(oldUpdatedAt))
	})

	t.Run("nil tokens are ignored", func(t *testing.T) {
		existingAccess := account.EncryptedAccessToken()
		existingRefresh := account.EncryptedRefreshToken()

		account.UpdateTokens(nil, nil, nil)

		// Tokens should not change
		assert.Equal(t, existingAccess, account.EncryptedAccessToken())
		assert.Equal(t, existingRefresh, account.EncryptedRefreshToken())
	})

	t.Run("empty byte slices are ignored", func(t *testing.T) {
		existingAccess := account.EncryptedAccessToken()

		account.UpdateTokens([]byte{}, []byte{}, nil)

		assert.Equal(t, existingAccess, account.EncryptedAccessToken())
	})
}

func TestOAuthAccount_ClearTokens(t *testing.T) {
	t.Parallel()

	userID := NewUserID()
	provider := OAuthProviderGoogle
	providerUserID, _ := NewProviderUserID("user123")
	email, _ := NewEmail("user@example.com")
	expiresAt := time.Now().Add(1 * time.Hour).UTC()

	account, _ := NewOAuthAccount(
		userID, provider, providerUserID, email, "Name", "url",
		[]byte("access"), []byte("refresh"), &expiresAt,
	)

	assert.True(t, account.HasStoredTokens())

	oldUpdatedAt := account.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	account.ClearTokens()

	assert.Nil(t, account.EncryptedAccessToken())
	assert.Nil(t, account.EncryptedRefreshToken())
	assert.Nil(t, account.TokenExpiresAt())
	assert.False(t, account.HasStoredTokens())
	assert.True(t, account.UpdatedAt().After(oldUpdatedAt))
}

func TestOAuthAccount_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full OAuth account lifecycle", func(t *testing.T) {
		t.Parallel()

		// User links Google account
		userID := NewUserID()
		provider := OAuthProviderGoogle
		providerUserID, _ := NewProviderUserID("google-123456")
		email, _ := NewEmail("user@gmail.com")
		displayName := "John Doe"
		avatarURL := "https://example.com/photo.jpg"
		accessToken := []byte("encrypted-access-token")
		refreshToken := []byte("encrypted-refresh-token")
		expiresAt := time.Now().Add(1 * time.Hour).UTC()

		// Create OAuth account
		account, err := NewOAuthAccount(
			userID,
			provider,
			providerUserID,
			email,
			displayName,
			avatarURL,
			accessToken,
			refreshToken,
			&expiresAt,
		)
		require.NoError(t, err)

		// Verify initial state
		assert.True(t, account.HasStoredTokens())
		assert.False(t, account.IsTokenExpired())

		// Simulate persistence
		reconstructed := ReconstructOAuthAccount(
			account.ID(),
			account.UserID(),
			account.Provider(),
			account.ProviderUserID(),
			account.Email(),
			account.DisplayName(),
			account.AvatarURL(),
			account.EncryptedAccessToken(),
			account.EncryptedRefreshToken(),
			account.TokenExpiresAt(),
			account.CreatedAt(),
			account.UpdatedAt(),
		)

		// User logs in again, profile updated
		newEmail, _ := NewEmail("newemail@gmail.com")
		newDisplayName := "John Updated"
		newAvatarURL := "https://example.com/newphoto.jpg"

		err = reconstructed.UpdateProfile(newEmail, newDisplayName, newAvatarURL)
		require.NoError(t, err)

		assert.Equal(t, newEmail, reconstructed.Email())
		assert.Equal(t, newDisplayName, reconstructed.DisplayName())
		assert.Equal(t, newAvatarURL, reconstructed.AvatarURL())

		// Token refresh
		newAccessToken := []byte("new-access-token")
		newRefreshToken := []byte("new-refresh-token")
		newExpiresAt := time.Now().Add(2 * time.Hour).UTC()

		reconstructed.UpdateTokens(newAccessToken, newRefreshToken, &newExpiresAt)

		assert.Equal(t, newAccessToken, reconstructed.EncryptedAccessToken())
		assert.Equal(t, newRefreshToken, reconstructed.EncryptedRefreshToken())

		// User revokes OAuth permission
		reconstructed.ClearTokens()

		assert.False(t, reconstructed.HasStoredTokens())
		assert.Nil(t, reconstructed.EncryptedAccessToken())
		assert.Nil(t, reconstructed.EncryptedRefreshToken())
	})
}
