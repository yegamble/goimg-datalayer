package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTOTPSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		encryptedSecret []byte
		accountName     string
		wantErr         error
	}{
		{
			name:            "valid TOTP secret",
			encryptedSecret: []byte("encrypted-secret-data"),
			accountName:     "user@example.com",
			wantErr:         nil,
		},
		{
			name:            "empty encrypted secret",
			encryptedSecret: []byte{},
			accountName:     "user@example.com",
			wantErr:         ErrTOTPSecretEmpty,
		},
		{
			name:            "nil encrypted secret",
			encryptedSecret: nil,
			accountName:     "user@example.com",
			wantErr:         ErrTOTPSecretEmpty,
		},
		{
			name:            "empty account name",
			encryptedSecret: []byte("encrypted-secret-data"),
			accountName:     "",
			wantErr:         ErrEmailEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret, err := NewTOTPSecret(tt.encryptedSecret, tt.accountName)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, secret.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.encryptedSecret, secret.EncryptedSecret())
				assert.Equal(t, tt.accountName, secret.AccountName())
				assert.Equal(t, "goimg", secret.Issuer())
				assert.False(t, secret.IsEnabled())
				assert.True(t, secret.IsSetupPending())
				assert.True(t, secret.VerifiedAt().IsZero())
			}
		})
	}
}

func TestReconstructTOTPSecret(t *testing.T) {
	t.Parallel()

	encryptedSecret := []byte("encrypted-secret-data")
	issuer := "custom-issuer"
	accountName := "user@example.com"
	enabled := true
	verifiedAt := time.Now().UTC()

	secret := ReconstructTOTPSecret(encryptedSecret, issuer, accountName, enabled, verifiedAt)

	assert.Equal(t, encryptedSecret, secret.EncryptedSecret())
	assert.Equal(t, issuer, secret.Issuer())
	assert.Equal(t, accountName, secret.AccountName())
	assert.True(t, secret.IsEnabled())
	assert.False(t, secret.IsSetupPending())
	assert.Equal(t, verifiedAt, secret.VerifiedAt())
}

func TestTOTPSecret_EncryptedSecret(t *testing.T) {
	t.Parallel()

	encryptedData := []byte("encrypted-totp-secret-12345")
	secret, err := NewTOTPSecret(encryptedData, "user@example.com")
	require.NoError(t, err)

	assert.Equal(t, encryptedData, secret.EncryptedSecret())
}

func TestTOTPSecret_Issuer(t *testing.T) {
	t.Parallel()

	secret, err := NewTOTPSecret([]byte("secret"), "user@example.com")
	require.NoError(t, err)

	assert.Equal(t, "goimg", secret.Issuer())
}

func TestTOTPSecret_AccountName(t *testing.T) {
	t.Parallel()

	accountName := "testuser@example.com"
	secret, err := NewTOTPSecret([]byte("secret"), accountName)
	require.NoError(t, err)

	assert.Equal(t, accountName, secret.AccountName())
}

func TestTOTPSecret_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		enabled    bool
		verifiedAt time.Time
		want       bool
	}{
		{
			name:       "enabled and verified",
			enabled:    true,
			verifiedAt: time.Now().UTC(),
			want:       true,
		},
		{
			name:       "enabled but not verified",
			enabled:    true,
			verifiedAt: time.Time{},
			want:       false,
		},
		{
			name:       "not enabled but verified",
			enabled:    false,
			verifiedAt: time.Now().UTC(),
			want:       false,
		},
		{
			name:       "not enabled and not verified",
			enabled:    false,
			verifiedAt: time.Time{},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret := ReconstructTOTPSecret([]byte("secret"), "goimg", "user@example.com", tt.enabled, tt.verifiedAt)
			assert.Equal(t, tt.want, secret.IsEnabled())
		})
	}
}

func TestTOTPSecret_IsSetupPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		encryptedSecret []byte
		enabled         bool
		want            bool
	}{
		{
			name:            "setup pending - has secret but not enabled",
			encryptedSecret: []byte("secret"),
			enabled:         false,
			want:            true,
		},
		{
			name:            "not pending - enabled",
			encryptedSecret: []byte("secret"),
			enabled:         true,
			want:            false,
		},
		{
			name:            "not pending - no secret",
			encryptedSecret: []byte{},
			enabled:         false,
			want:            false,
		},
		{
			name:            "not pending - nil secret",
			encryptedSecret: nil,
			enabled:         false,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret := ReconstructTOTPSecret(tt.encryptedSecret, "goimg", "user@example.com", tt.enabled, time.Time{})
			assert.Equal(t, tt.want, secret.IsSetupPending())
		})
	}
}

func TestTOTPSecret_VerifiedAt(t *testing.T) {
	t.Parallel()

	verifiedAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	secret := ReconstructTOTPSecret([]byte("secret"), "goimg", "user@example.com", true, verifiedAt)

	assert.Equal(t, verifiedAt, secret.VerifiedAt())
}

func TestTOTPSecret_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		encryptedSecret []byte
		want            bool
	}{
		{
			name:            "not empty - has secret",
			encryptedSecret: []byte("secret"),
			want:            false,
		},
		{
			name:            "empty - zero length",
			encryptedSecret: []byte{},
			want:            true,
		},
		{
			name:            "empty - nil",
			encryptedSecret: nil,
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret := ReconstructTOTPSecret(tt.encryptedSecret, "goimg", "user@example.com", false, time.Time{})
			assert.Equal(t, tt.want, secret.IsEmpty())
		})
	}
}

func TestTOTPSecret_Enable(t *testing.T) {
	t.Parallel()

	t.Run("enable sets verified timestamp first time", func(t *testing.T) {
		t.Parallel()

		secret, err := NewTOTPSecret([]byte("secret"), "user@example.com")
		require.NoError(t, err)

		before := time.Now().UTC()
		secret.Enable()
		after := time.Now().UTC()

		assert.True(t, secret.IsEnabled())
		assert.False(t, secret.VerifiedAt().IsZero())
		assert.True(t, secret.VerifiedAt().After(before.Add(-time.Second)))
		assert.True(t, secret.VerifiedAt().Before(after.Add(time.Second)))
	})

	t.Run("enable is idempotent - preserves original verified timestamp", func(t *testing.T) {
		t.Parallel()

		originalVerified := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		secret := ReconstructTOTPSecret([]byte("secret"), "goimg", "user@example.com", true, originalVerified)

		secret.Enable()

		assert.True(t, secret.IsEnabled())
		assert.Equal(t, originalVerified, secret.VerifiedAt())
	})
}

func TestTOTPSecret_Disable(t *testing.T) {
	t.Parallel()

	t.Run("disable marks TOTP as disabled", func(t *testing.T) {
		t.Parallel()

		verifiedAt := time.Now().UTC()
		secret := ReconstructTOTPSecret([]byte("secret"), "goimg", "user@example.com", true, verifiedAt)

		assert.True(t, secret.IsEnabled())

		secret.Disable()

		assert.False(t, secret.IsEnabled())
		// VerifiedAt is preserved for audit purposes
		assert.Equal(t, verifiedAt, secret.VerifiedAt())
	})

	t.Run("disable is idempotent", func(t *testing.T) {
		t.Parallel()

		secret, err := NewTOTPSecret([]byte("secret"), "user@example.com")
		require.NoError(t, err)

		secret.Disable()
		secret.Disable()

		assert.False(t, secret.IsEnabled())
	})
}

func TestValidateBase32Secret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		plaintext string
		wantErr   error
	}{
		{
			name:      "valid base32 secret",
			plaintext: "JBSWY3DPEHPK3PXP",
			wantErr:   nil,
		},
		{
			name:      "valid base32 longer secret",
			plaintext: "MFRGGZDFMZTWQ2LKNNWG23TPOBYXE43UOV3HO6DZPI",
			wantErr:   nil,
		},
		{
			name:      "valid base32 short secret",
			plaintext: "ABCD2345",
			wantErr:   nil,
		},
		{
			name:      "empty string",
			plaintext: "",
			wantErr:   ErrTOTPSecretEmpty,
		},
		{
			name:      "invalid base32 characters - lowercase not allowed",
			plaintext: "jbswy3dpehpk3pxp",
			wantErr:   ErrTOTPSecretEmpty,
		},
		{
			name:      "invalid base32 characters - special chars",
			plaintext: "INVALID123!@#",
			wantErr:   ErrTOTPSecretEmpty,
		},
		{
			name:      "contains invalid characters - padding not allowed",
			plaintext: "JBSWY3DP=EHPK3PXP",
			wantErr:   ErrTOTPSecretEmpty,
		},
		{
			name:      "invalid character - number 1",
			plaintext: "JBSWY3DP1EHPK3PXP",
			wantErr:   ErrTOTPSecretEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateBase32Secret(tt.plaintext)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
