package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// BackupCode is a value object representing a one-time use recovery code for 2FA.
// Backup codes are hashed using Argon2id (same as passwords) and can only be used once.
type BackupCode struct {
	hashedCode string    // Argon2id hash of the backup code
	used       bool      // Whether this code has been consumed
	usedAt     time.Time // When the code was used (zero if unused)
}

// Backup code configuration constants.
const (
	backupCodeLength      = 8         // 8 characters per code (readable format)
	backupCodeCount       = 10        // Generate 10 backup codes per user
	backupCodeSaltLen     = 16        // Salt length for hashing
	backupCodeArgonTime   = 1         // Faster than password hashing (still secure for 8-char codes)
	backupCodeArgonMem    = 32 * 1024 // 32 MB (less than password, sufficient for backup codes)
	backupCodeArgonPar    = 2         // Parallelism
	backupCodeArgonKeyLen = 32        // Output key length
)

// GenerateBackupCodes creates a set of backup codes for 2FA recovery.
// Returns both the plaintext codes (to show to the user once) and the hashed codes (to store).
func GenerateBackupCodes() (plaintext []string, codes []BackupCode, err error) {
	plaintext = make([]string, backupCodeCount)
	codes = make([]BackupCode, backupCodeCount)

	for i := 0; i < backupCodeCount; i++ {
		// Generate 5 random bytes (40 bits of entropy, base32 encodes to 8 chars)
		randomBytes := make([]byte, 5)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, nil, fmt.Errorf("generate backup code: %w", err)
		}

		// Encode to base32 and take first 8 characters (uppercase for readability)
		code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
		code = strings.ToUpper(code[:backupCodeLength])
		plaintext[i] = code

		// Hash the code using Argon2id
		hashedCode, err := hashBackupCode(code)
		if err != nil {
			return nil, nil, fmt.Errorf("hash backup code: %w", err)
		}

		codes[i] = BackupCode{
			hashedCode: hashedCode,
			used:       false,
			usedAt:     time.Time{},
		}
	}

	return plaintext, codes, nil
}

// ReconstructBackupCode reconstitutes a backup code from storage.
// This should only be used by the repository layer when loading from the database.
func ReconstructBackupCode(hashedCode string, used bool, usedAt time.Time) BackupCode {
	return BackupCode{
		hashedCode: hashedCode,
		used:       used,
		usedAt:     usedAt,
	}
}

// HashedCode returns the Argon2id hash of the backup code.
// This should only be used for persistence.
func (c BackupCode) HashedCode() string {
	return c.hashedCode
}

// IsUsed returns whether this backup code has been consumed.
func (c BackupCode) IsUsed() bool {
	return c.used
}

// UsedAt returns when the backup code was used (zero time if unused).
func (c BackupCode) UsedAt() time.Time {
	return c.usedAt
}

// Verify checks if the given plaintext matches this backup code's hash.
// Uses constant-time comparison to prevent timing attacks.
func (c BackupCode) Verify(plaintext string) error {
	if c.used {
		return ErrBackupCodeInvalid
	}

	// Normalize input (uppercase, no spaces)
	normalized := strings.ToUpper(strings.ReplaceAll(plaintext, " ", ""))

	// Parse the stored hash
	parts := strings.Split(c.hashedCode, "$")
	if len(parts) != 6 {
		return fmt.Errorf("invalid backup code hash format")
	}

	// Extract salt
	salt, err := base64DecodeRaw(parts[4])
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}

	// Extract expected hash
	expectedHash, err := base64DecodeRaw(parts[5])
	if err != nil {
		return fmt.Errorf("decode hash: %w", err)
	}

	// Hash the input with the same salt and parameters
	actualHash := argon2.IDKey(
		[]byte(normalized),
		salt,
		backupCodeArgonTime,
		backupCodeArgonMem,
		backupCodeArgonPar,
		backupCodeArgonKeyLen,
	)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(expectedHash, actualHash) != 1 {
		return ErrBackupCodeInvalid
	}

	return nil
}

// MarkUsed marks this backup code as used.
func (c *BackupCode) MarkUsed() {
	c.used = true
	c.usedAt = time.Now().UTC()
}

// hashBackupCode hashes a plaintext backup code using Argon2id.
func hashBackupCode(plaintext string) (string, error) {
	// Generate salt
	salt := make([]byte, backupCodeSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	// Hash using Argon2id
	hash := argon2.IDKey(
		[]byte(plaintext),
		salt,
		backupCodeArgonTime,
		backupCodeArgonMem,
		backupCodeArgonPar,
		backupCodeArgonKeyLen,
	)

	// Encode in PHC string format
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		backupCodeArgonMem,
		backupCodeArgonTime,
		backupCodeArgonPar,
		base64EncodeRaw(salt),
		base64EncodeRaw(hash),
	)

	return encoded, nil
}

// CountUnusedBackupCodes counts how many backup codes are still available.
func CountUnusedBackupCodes(codes []BackupCode) int {
	count := 0
	for _, code := range codes {
		if !code.IsUsed() {
			count++
		}
	}
	return count
}

// base64 encoding helpers (same as used in password.go)
func base64EncodeRaw(data []byte) string {
	return base64.RawStdEncoding.EncodeToString(data)
}

func base64DecodeRaw(s string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(s)
}
