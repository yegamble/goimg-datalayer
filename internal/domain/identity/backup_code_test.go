package identity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBackupCodes(t *testing.T) {
	t.Parallel()

	t.Run("generates correct number of codes", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()

		require.NoError(t, err)
		assert.Len(t, plaintext, 10)
		assert.Len(t, codes, 10)
	})

	t.Run("plaintext codes are uppercase and 8 characters", func(t *testing.T) {
		t.Parallel()

		plaintext, _, err := GenerateBackupCodes()
		require.NoError(t, err)

		for i, code := range plaintext {
			assert.Len(t, code, 8, "code %d should be 8 characters", i)
			assert.Equal(t, strings.ToUpper(code), code, "code %d should be uppercase", i)
			// Should only contain valid base32 characters (A-Z, 2-7)
			for _, ch := range code {
				isValid := (ch >= 'A' && ch <= 'Z') || (ch >= '2' && ch <= '7')
				assert.True(t, isValid, "code %d contains invalid character %c", i, ch)
			}
		}
	})

	t.Run("codes are unique", func(t *testing.T) {
		t.Parallel()

		plaintext, _, err := GenerateBackupCodes()
		require.NoError(t, err)

		seen := make(map[string]bool)
		for _, code := range plaintext {
			assert.False(t, seen[code], "duplicate code found: %s", code)
			seen[code] = true
		}
	})

	t.Run("backup codes are hashed", func(t *testing.T) {
		t.Parallel()

		_, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		for i, code := range codes {
			assert.NotEmpty(t, code.HashedCode(), "code %d should have hash", i)
			assert.True(t, strings.HasPrefix(code.HashedCode(), "$argon2id$"), "code %d hash should use argon2id", i)
			assert.False(t, code.IsUsed(), "code %d should not be used initially", i)
			assert.True(t, code.UsedAt().IsZero(), "code %d usedAt should be zero", i)
		}
	})

	t.Run("multiple generations produce different codes", func(t *testing.T) {
		t.Parallel()

		plaintext1, _, err := GenerateBackupCodes()
		require.NoError(t, err)

		plaintext2, _, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Very unlikely to get the same set of random codes
		same := true
		for i := range plaintext1 {
			if plaintext1[i] != plaintext2[i] {
				same = false
				break
			}
		}
		assert.False(t, same, "two generations should produce different codes")
	})
}

func TestReconstructBackupCode(t *testing.T) {
	t.Parallel()

	hashedCode := "$argon2id$v=19$m=32768,t=1,p=2$salt$hash"
	used := true
	usedAt := time.Now().UTC()

	code := ReconstructBackupCode(hashedCode, used, usedAt)

	assert.Equal(t, hashedCode, code.HashedCode())
	assert.Equal(t, used, code.IsUsed())
	assert.Equal(t, usedAt, code.UsedAt())
}

func TestBackupCode_HashedCode(t *testing.T) {
	t.Parallel()

	hashedCode := "$argon2id$v=19$m=32768,t=1,p=2$test$test"
	code := ReconstructBackupCode(hashedCode, false, time.Time{})

	assert.Equal(t, hashedCode, code.HashedCode())
}

func TestBackupCode_IsUsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		used bool
	}{
		{
			name: "unused backup code",
			used: false,
		},
		{
			name: "used backup code",
			used: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code := ReconstructBackupCode("hash", tt.used, time.Time{})
			assert.Equal(t, tt.used, code.IsUsed())
		})
	}
}

func TestBackupCode_UsedAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		usedAt time.Time
	}{
		{
			name:   "zero time for unused code",
			usedAt: time.Time{},
		},
		{
			name:   "timestamp for used code",
			usedAt: time.Date(2023, 5, 15, 14, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code := ReconstructBackupCode("hash", false, tt.usedAt)
			assert.Equal(t, tt.usedAt, code.UsedAt())
		})
	}
}

func TestBackupCode_Verify(t *testing.T) {
	t.Parallel()

	t.Run("valid code verification succeeds", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Test first code
		err = codes[0].Verify(plaintext[0])
		assert.NoError(t, err)
	})

	t.Run("verify all generated codes", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		for i, code := range codes {
			err := code.Verify(plaintext[i])
			assert.NoError(t, err, "code %d should verify", i)
		}
	})

	t.Run("wrong code fails verification", func(t *testing.T) {
		t.Parallel()

		_, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Try to verify with wrong code
		err = codes[0].Verify("WRONGCOD")
		assert.ErrorIs(t, err, ErrBackupCodeInvalid)
	})

	t.Run("verification normalizes spaces", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Add spaces to the plaintext code
		codeWithSpaces := plaintext[0][:4] + " " + plaintext[0][4:]
		err = codes[0].Verify(codeWithSpaces)
		assert.NoError(t, err, "spaces should be normalized")
	})

	t.Run("verification is case insensitive", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Verify with lowercase
		err = codes[0].Verify(strings.ToLower(plaintext[0]))
		assert.NoError(t, err, "verification should be case insensitive")
	})

	t.Run("used code fails verification", func(t *testing.T) {
		t.Parallel()

		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Mark code as used
		codes[0].MarkUsed()

		// Try to verify used code
		err = codes[0].Verify(plaintext[0])
		assert.ErrorIs(t, err, ErrBackupCodeInvalid)
	})

	t.Run("empty code fails verification", func(t *testing.T) {
		t.Parallel()

		_, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		err = codes[0].Verify("")
		assert.ErrorIs(t, err, ErrBackupCodeInvalid)
	})
}

func TestBackupCode_MarkUsed(t *testing.T) {
	t.Parallel()

	t.Run("marking code as used sets fields", func(t *testing.T) {
		t.Parallel()

		code := ReconstructBackupCode("hash", false, time.Time{})

		assert.False(t, code.IsUsed())
		assert.True(t, code.UsedAt().IsZero())

		before := time.Now().UTC()
		code.MarkUsed()
		after := time.Now().UTC()

		assert.True(t, code.IsUsed())
		assert.False(t, code.UsedAt().IsZero())
		assert.True(t, code.UsedAt().After(before.Add(-time.Second)))
		assert.True(t, code.UsedAt().Before(after.Add(time.Second)))
	})

	t.Run("marking used code again updates timestamp", func(t *testing.T) {
		t.Parallel()

		code := ReconstructBackupCode("hash", false, time.Time{})

		code.MarkUsed()
		firstUsedAt := code.UsedAt()

		// Small delay to ensure different timestamp
		time.Sleep(10 * time.Millisecond)

		code.MarkUsed()
		secondUsedAt := code.UsedAt()

		assert.True(t, secondUsedAt.After(firstUsedAt))
	})
}

func TestCountUnusedBackupCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupCodes      func() []BackupCode
		wantUnusedCount int
	}{
		{
			name: "all codes unused",
			setupCodes: func() []BackupCode {
				_, codes, _ := GenerateBackupCodes()
				return codes
			},
			wantUnusedCount: 10,
		},
		{
			name: "some codes used",
			setupCodes: func() []BackupCode {
				_, codes, _ := GenerateBackupCodes()
				codes[0].MarkUsed()
				codes[2].MarkUsed()
				codes[5].MarkUsed()
				return codes
			},
			wantUnusedCount: 7,
		},
		{
			name: "all codes used",
			setupCodes: func() []BackupCode {
				_, codes, _ := GenerateBackupCodes()
				for i := range codes {
					codes[i].MarkUsed()
				}
				return codes
			},
			wantUnusedCount: 0,
		},
		{
			name: "empty list",
			setupCodes: func() []BackupCode {
				return []BackupCode{}
			},
			wantUnusedCount: 0,
		},
		{
			name: "single unused code",
			setupCodes: func() []BackupCode {
				return []BackupCode{
					ReconstructBackupCode("hash", false, time.Time{}),
				}
			},
			wantUnusedCount: 1,
		},
		{
			name: "single used code",
			setupCodes: func() []BackupCode {
				code := ReconstructBackupCode("hash", false, time.Time{})
				code.MarkUsed()
				return []BackupCode{code}
			},
			wantUnusedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codes := tt.setupCodes()
			count := CountUnusedBackupCodes(codes)

			assert.Equal(t, tt.wantUnusedCount, count)
		})
	}
}

func TestBackupCode_HashFormat(t *testing.T) {
	t.Parallel()

	t.Run("generated hash has correct PHC format", func(t *testing.T) {
		t.Parallel()

		_, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		hash := codes[0].HashedCode()

		// PHC format: $algorithm$parameters$salt$hash
		parts := strings.Split(hash, "$")
		assert.Len(t, parts, 6, "PHC format should have 6 parts")
		assert.Equal(t, "", parts[0], "first part should be empty")
		assert.Equal(t, "argon2id", parts[1], "algorithm should be argon2id")
		assert.True(t, strings.HasPrefix(parts[2], "v="), "should have version")
		assert.True(t, strings.Contains(parts[3], "m="), "should have memory parameter")
		assert.True(t, strings.Contains(parts[3], "t="), "should have time parameter")
		assert.True(t, strings.Contains(parts[3], "p="), "should have parallelism parameter")
		assert.NotEmpty(t, parts[4], "should have salt")
		assert.NotEmpty(t, parts[5], "should have hash")
	})
}

func TestBackupCode_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full lifecycle of backup code", func(t *testing.T) {
		t.Parallel()

		// Generate codes
		plaintext, codes, err := GenerateBackupCodes()
		require.NoError(t, err)

		// Select one code to test
		testCode := codes[3]
		testPlaintext := plaintext[3]

		// Verify initial state
		assert.False(t, testCode.IsUsed())
		assert.True(t, testCode.UsedAt().IsZero())

		// Verify the code works
		err = testCode.Verify(testPlaintext)
		assert.NoError(t, err)

		// Simulate persistence: reconstruct from storage
		reconstructed := ReconstructBackupCode(
			testCode.HashedCode(),
			testCode.IsUsed(),
			testCode.UsedAt(),
		)

		// Verify reconstructed code
		err = reconstructed.Verify(testPlaintext)
		assert.NoError(t, err)

		// Use the code
		reconstructed.MarkUsed()

		// Verify cannot be used again
		err = reconstructed.Verify(testPlaintext)
		assert.ErrorIs(t, err, ErrBackupCodeInvalid)

		// Verify state after use
		assert.True(t, reconstructed.IsUsed())
		assert.False(t, reconstructed.UsedAt().IsZero())
	})
}
