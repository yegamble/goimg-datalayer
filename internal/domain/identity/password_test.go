package identity_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewPasswordHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid strong password",
			input:   "MySecureP@ssw0rd123",
			wantErr: nil,
		},
		{
			name:    "valid long password",
			input:   "ThisIsAVeryLongButSecurePasswordThatMeetsAllRequirements2024!",
			wantErr: nil,
		},
		{
			name:    "exactly 12 characters",
			input:   "ValidPass123",
			wantErr: nil,
		},
		{
			name:    "exactly 128 characters",
			input:   strings.Repeat("a", 128),
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: identity.ErrPasswordEmpty,
		},
		{
			name:    "too short - 11 characters",
			input:   "Short123456",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "too short - 1 character",
			input:   "a",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "too long - 129 characters",
			input:   strings.Repeat("a", 129),
			wantErr: identity.ErrPasswordTooLong,
		},
		{
			name:    "common password - password (too short)",
			input:   "password",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password - password123 (too short)",
			input:   "password123",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password - 123456 (too short)",
			input:   "123456",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password - qwerty (too short)",
			input:   "qwerty",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password case insensitive - PASSWORD (too short)",
			input:   "PASSWORD",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password case insensitive - Password123 (too short)",
			input:   "Password123",
			wantErr: identity.ErrPasswordTooShort,
		},
		{
			name:    "common password >= 12 chars - password1234",
			input:   "password1234",
			wantErr: identity.ErrPasswordWeak,
		},
		{
			name:    "common password >= 12 chars - 123456789012",
			input:   "123456789012",
			wantErr: identity.ErrPasswordWeak,
		},
		{
			name:    "common password >= 12 chars case insensitive - PASSWORD1234",
			input:   "PASSWORD1234",
			wantErr: identity.ErrPasswordWeak,
		},
		{
			name:    "common password >= 12 chars - qwertyuiop123",
			input:   "qwertyuiop123",
			wantErr: identity.ErrPasswordWeak,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := identity.NewPasswordHash(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, hash.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.False(t, hash.IsEmpty())
				assert.NotEmpty(t, hash.String())

				// Verify the hash format
				assert.True(t, strings.HasPrefix(hash.String(), "$argon2id$"))
			}
		})
	}
}

func TestPasswordHash_Verify(t *testing.T) {
	t.Parallel()

	t.Run("correct password verifies successfully", func(t *testing.T) {
		t.Parallel()

		password := "MySecureP@ssw0rd123"
		hash, err := identity.NewPasswordHash(password)
		require.NoError(t, err)

		err = hash.Verify(password)
		assert.NoError(t, err)
	})

	t.Run("incorrect password fails verification", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		err = hash.Verify("WrongPassword123")
		require.ErrorIs(t, err, identity.ErrPasswordMismatch)
	})

	t.Run("empty password fails verification", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		err = hash.Verify("")
		require.Error(t, err)
	})

	t.Run("case sensitive verification", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		err = hash.Verify("mysecurep@ssw0rd123")
		require.ErrorIs(t, err, identity.ErrPasswordMismatch)
	})

	t.Run("empty hash fails verification", func(t *testing.T) {
		t.Parallel()

		var hash identity.PasswordHash
		err := hash.Verify("anypassword")
		require.ErrorIs(t, err, identity.ErrPasswordEmpty)
	})
}

func TestPasswordHash_UniqueHashes(t *testing.T) {
	t.Parallel()

	// Same password should produce different hashes due to unique salts
	password := "MySecureP@ssw0rd123"

	hash1, err := identity.NewPasswordHash(password)
	require.NoError(t, err)

	hash2, err := identity.NewPasswordHash(password)
	require.NoError(t, err)

	// Hashes should be different (different salts)
	assert.NotEqual(t, hash1.String(), hash2.String())

	// But both should verify the same password
	assert.NoError(t, hash1.Verify(password))
	assert.NoError(t, hash2.Verify(password))
}

func TestPasswordHash_HashFormat(t *testing.T) {
	t.Parallel()

	hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
	require.NoError(t, err)

	encoded := hash.String()

	// Verify PHC string format: $argon2id$v=19$m=65536,t=2,p=4$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	require.Len(t, parts, 6)
	assert.Empty(t, parts[0])
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, "v=19", parts[2])
	assert.Equal(t, "m=65536,t=2,p=4", parts[3])
	assert.NotEmpty(t, parts[4]) // salt
	assert.NotEmpty(t, parts[5]) // hash
}

func TestParsePasswordHash(t *testing.T) {
	t.Parallel()

	t.Run("valid hash string parses successfully", func(t *testing.T) {
		t.Parallel()

		original, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		parsed, err := identity.ParsePasswordHash(original.String())
		require.NoError(t, err)

		assert.Equal(t, original.String(), parsed.String())
	})

	t.Run("empty string fails", func(t *testing.T) {
		t.Parallel()

		_, err := identity.ParsePasswordHash("")
		require.ErrorIs(t, err, identity.ErrPasswordEmpty)
	})

	t.Run("invalid format fails", func(t *testing.T) {
		t.Parallel()

		_, err := identity.ParsePasswordHash("not-a-valid-hash")
		require.Error(t, err)
	})

	t.Run("wrong algorithm fails", func(t *testing.T) {
		t.Parallel()

		_, err := identity.ParsePasswordHash("$bcrypt$v=1$salt$hash")
		require.Error(t, err)
	})

	t.Run("parsed hash can verify password", func(t *testing.T) {
		t.Parallel()

		password := "MySecureP@ssw0rd123"
		original, err := identity.NewPasswordHash(password)
		require.NoError(t, err)

		parsed, err := identity.ParsePasswordHash(original.String())
		require.NoError(t, err)

		err = parsed.Verify(password)
		assert.NoError(t, err)
	})
}

func TestPasswordHash_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("zero value is empty", func(t *testing.T) {
		t.Parallel()

		var hash identity.PasswordHash
		assert.True(t, hash.IsEmpty())
	})

	t.Run("valid hash is not empty", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.False(t, hash.IsEmpty())
	})
}

func TestPasswordHash_Security(t *testing.T) {
	t.Parallel()

	t.Run("uses Argon2id algorithm", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.Contains(t, hash.String(), "$argon2id$")
	})

	t.Run("uses Argon2 version 19", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.Contains(t, hash.String(), "v=19")
	})

	t.Run("uses recommended memory cost", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.Contains(t, hash.String(), "m=65536") // 64 MB
	})

	t.Run("uses recommended time cost", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.Contains(t, hash.String(), "t=2")
	})

	t.Run("uses recommended parallelism", func(t *testing.T) {
		t.Parallel()

		hash, err := identity.NewPasswordHash("MySecureP@ssw0rd123")
		require.NoError(t, err)

		assert.Contains(t, hash.String(), "p=4")
	})
}

// Benchmark to ensure hashing performance is reasonable.
func BenchmarkNewPasswordHash(b *testing.B) {
	password := "MySecureP@ssw0rd123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := identity.NewPasswordHash(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPasswordHash_Verify(b *testing.B) {
	password := "MySecureP@ssw0rd123"
	hash, err := identity.NewPasswordHash(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := hash.Verify(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}
