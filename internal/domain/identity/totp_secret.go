package identity

import (
	"encoding/base32"
	"time"
)

// TOTPSecret is a value object representing a TOTP secret for two-factor authentication.
// The secret is stored encrypted at rest and decrypted only during verification.
type TOTPSecret struct {
	encryptedSecret []byte    // AES-256-GCM encrypted secret
	issuer          string    // App name shown in authenticator (e.g., "goimg")
	accountName     string    // Account identifier (usually email)
	enabled         bool      // Whether 2FA is currently active
	verifiedAt      time.Time // When 2FA was first successfully verified (zero if not verified)
}

// TOTPSecret configuration constants.
const (
	totpSecretLength = 20      // 160 bits of entropy (RFC 6238 recommendation)
	totpIssuer       = "goimg" // Default issuer name for QR codes
)

// NewTOTPSecret creates a new TOTP secret from encrypted bytes.
// The secret should be pre-encrypted using AES-256-GCM before calling this.
func NewTOTPSecret(encryptedSecret []byte, accountName string) (TOTPSecret, error) {
	if len(encryptedSecret) == 0 {
		return TOTPSecret{}, ErrTOTPSecretEmpty
	}

	if accountName == "" {
		return TOTPSecret{}, ErrEmailEmpty
	}

	return TOTPSecret{
		encryptedSecret: encryptedSecret,
		issuer:          totpIssuer,
		accountName:     accountName,
		enabled:         false, // Not enabled until verified
		verifiedAt:      time.Time{},
	}, nil
}

// ReconstructTOTPSecret reconstitutes a TOTP secret from storage.
// This should only be used by the repository layer when loading from the database.
func ReconstructTOTPSecret(
	encryptedSecret []byte,
	issuer, accountName string,
	enabled bool,
	verifiedAt time.Time,
) TOTPSecret {
	return TOTPSecret{
		encryptedSecret: encryptedSecret,
		issuer:          issuer,
		accountName:     accountName,
		enabled:         enabled,
		verifiedAt:      verifiedAt,
	}
}

// EncryptedSecret returns the encrypted TOTP secret bytes.
// This should only be used for persistence.
func (s TOTPSecret) EncryptedSecret() []byte {
	return s.encryptedSecret
}

// Issuer returns the issuer name for the TOTP (shown in authenticator apps).
func (s TOTPSecret) Issuer() string {
	return s.issuer
}

// AccountName returns the account name for the TOTP (usually the user's email).
func (s TOTPSecret) AccountName() string {
	return s.accountName
}

// IsEnabled returns whether 2FA is enabled and verified.
func (s TOTPSecret) IsEnabled() bool {
	return s.enabled && !s.verifiedAt.IsZero()
}

// IsSetupPending returns whether 2FA setup was started but not verified.
func (s TOTPSecret) IsSetupPending() bool {
	return !s.enabled && len(s.encryptedSecret) > 0
}

// VerifiedAt returns when 2FA was first successfully verified.
func (s TOTPSecret) VerifiedAt() time.Time {
	return s.verifiedAt
}

// IsEmpty returns true if the TOTP secret is not set.
func (s TOTPSecret) IsEmpty() bool {
	return len(s.encryptedSecret) == 0
}

// Enable marks the TOTP as enabled and sets the verification timestamp.
// This should be called after the user successfully verifies their first TOTP code.
func (s *TOTPSecret) Enable() {
	s.enabled = true
	if s.verifiedAt.IsZero() {
		s.verifiedAt = time.Now().UTC()
	}
}

// Disable marks the TOTP as disabled (but preserves the secret for audit purposes).
func (s *TOTPSecret) Disable() {
	s.enabled = false
}

// ValidateBase32Secret validates that a plaintext secret is valid base32.
// This is used during TOTP setup before encryption.
func ValidateBase32Secret(plaintext string) error {
	if plaintext == "" {
		return ErrTOTPSecretEmpty
	}

	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(plaintext)
	if err != nil {
		return ErrTOTPSecretEmpty
	}

	return nil
}
