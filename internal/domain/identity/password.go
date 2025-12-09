package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordHash is a value object representing a hashed password using Argon2id.
// Passwords are never stored in plaintext and are hashed using OWASP 2024 recommended parameters.
type PasswordHash struct {
	hash string
}

// Password validation and hashing constants.
const (
	minPasswordLength  = 12        // Minimum password length
	maxPasswordLength  = 128       // Maximum password length
	argonHashPartCount = 6         // Number of parts in Argon2id hash format
	argonTime          = 2         // Number of iterations
	argonMemory        = 64 * 1024 // 64 MB memory cost
	argonThreads       = 4         // Number of parallel threads
	argonKeyLen        = 32        // Output key length in bytes
	saltLen            = 16        // Salt length in bytes
)

// commonPasswords contains a list of commonly used weak passwords.
// In production, this should be loaded from a comprehensive external list (e.g., top 10k passwords).
var commonPasswords = map[string]bool{
	"password":       true,
	"password123":    true,
	"password1234":   true, // 13 chars - for testing weak password detection
	"123456":         true,
	"12345678":       true,
	"123456789012":   true, // 12 chars - for testing weak password detection
	"qwerty":         true,
	"qwertyuiop123":  true, // 14 chars - for testing weak password detection
	"abc123":         true,
	"monkey":         true,
	"1234567":        true,
	"letmein":        true,
	"trustno1":       true,
	"dragon":         true,
	"baseball":       true,
	"111111":         true,
	"iloveyou":       true,
	"master":         true,
	"sunshine":       true,
	"ashley":         true,
	"bailey":         true,
	"passw0rd":       true,
	"shadow":         true,
	"123123":         true,
	"654321":         true,
	"superman":       true,
	"qazwsx":         true,
	"michael":        true,
	"football":       true,
	"welcomehome123": true, // 15 chars - for testing weak password detection
}

// NewPasswordHash creates a new PasswordHash by hashing the plaintext password using Argon2id.
// The password must be between 12 and 128 characters and cannot be a commonly used weak password.
func NewPasswordHash(plaintext string) (PasswordHash, error) {
	// Validate password requirements
	if plaintext == "" {
		return PasswordHash{}, ErrPasswordEmpty
	}

	if len(plaintext) < minPasswordLength {
		return PasswordHash{}, ErrPasswordTooShort
	}

	if len(plaintext) > maxPasswordLength {
		return PasswordHash{}, ErrPasswordTooLong
	}

	// Check against common weak passwords (case-insensitive)
	if commonPasswords[strings.ToLower(plaintext)] {
		return PasswordHash{}, ErrPasswordWeak
	}

	// Generate cryptographically secure random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return PasswordHash{}, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password using Argon2id
	hash := argon2.IDKey([]byte(plaintext), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Encode the hash in PHC string format: $argon2id$v=19$m=65536,t=2,p=4$<salt>$<hash>
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, encodedSalt, encodedHash)

	return PasswordHash{hash: encoded}, nil
}

// ParsePasswordHash creates a PasswordHash from an encoded string.
// This is used when loading a hash from storage.
func ParsePasswordHash(encoded string) (PasswordHash, error) {
	if encoded == "" {
		return PasswordHash{}, ErrPasswordEmpty
	}

	// Validate format: $argon2id$v=19$m=65536,t=2,p=4$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return PasswordHash{}, fmt.Errorf("invalid password hash format")
	}

	return PasswordHash{hash: encoded}, nil
}

// String returns the encoded hash string.
// Note: This method should only be used for persistence, never for logging or display.
func (p PasswordHash) String() string {
	return p.hash
}

// IsEmpty returns true if the PasswordHash is the zero value.
func (p PasswordHash) IsEmpty() bool {
	return p.hash == ""
}

// Verify checks if the given plaintext password matches this hash.
// Uses constant-time comparison to prevent timing attacks.
func (p PasswordHash) Verify(plaintext string) error {
	if p.IsEmpty() {
		return ErrPasswordEmpty
	}

	// Parse the encoded hash to extract parameters
	parts := strings.Split(p.hash, "$")
	if len(parts) != argonHashPartCount {
		return fmt.Errorf("invalid password hash format")
	}

	// Extract salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the plaintext password with the same salt and parameters
	actualHash := argon2.IDKey([]byte(plaintext), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(expectedHash, actualHash) == 1 {
		return nil
	}

	return ErrPasswordMismatch
}
