package community

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const (
	// InvitationTokenLength is the length of the token in bytes (32 bytes = 64 hex chars).
	InvitationTokenLength = 32
)

// InvitationToken is a value object representing a cryptographically secure invitation token.
// Tokens are generated using crypto/rand and hex encoded to 64 characters.
type InvitationToken struct {
	value string
}

// NewInvitationToken generates a new cryptographically secure invitation token.
// The token is 32 random bytes hex-encoded to 64 characters.
// Returns an error if random generation fails.
func NewInvitationToken() (InvitationToken, error) {
	b := make([]byte, InvitationTokenLength)
	if _, err := rand.Read(b); err != nil {
		return InvitationToken{}, fmt.Errorf("generate invitation token: %w", err)
	}
	return InvitationToken{value: hex.EncodeToString(b)}, nil
}

// ParseInvitationToken creates an InvitationToken from an existing token string.
// Validates that the token is 64 hex characters.
// Returns an error if validation fails.
func ParseInvitationToken(s string) (InvitationToken, error) {
	if len(s) != InvitationTokenLength*2 { // 32 bytes = 64 hex chars
		return InvitationToken{}, fmt.Errorf("invalid invitation token: must be 64 characters")
	}

	// Validate it's valid hex
	if _, err := hex.DecodeString(s); err != nil {
		return InvitationToken{}, fmt.Errorf("invalid invitation token: must be hexadecimal: %w", err)
	}

	return InvitationToken{value: s}, nil
}

// String returns the string representation of the token.
func (t InvitationToken) String() string {
	return t.value
}

// IsEmpty returns true if the token is empty.
func (t InvitationToken) IsEmpty() bool {
	return t.value == ""
}

// Equals returns true if this token equals the other token.
// Uses constant-time comparison to prevent timing attacks.
func (t InvitationToken) Equals(other InvitationToken) bool {
	// Use constant-time comparison to prevent timing attacks.
	return subtle.ConstantTimeCompare([]byte(t.value), []byte(other.value)) == 1
}
