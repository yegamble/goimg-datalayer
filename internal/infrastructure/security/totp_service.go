package security

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPService errors.
var (
	ErrTOTPGenerationFailed    = errors.New("failed to generate TOTP secret")
	ErrTOTPValidationFailed    = errors.New("failed to validate TOTP code")
	ErrTOTPInvalidCode         = errors.New("invalid TOTP code")
	ErrTOTPSecretDecryptFailed = errors.New("failed to decrypt TOTP secret")
)

// TOTPConfig holds configuration for TOTP generation and validation.
type TOTPConfig struct {
	// Issuer is the organization name shown in authenticator apps.
	// Default: "goimg"
	Issuer string

	// Period is the time step in seconds (standard is 30).
	Period uint

	// Digits is the number of digits in the TOTP code (standard is 6).
	Digits otp.Digits

	// Algorithm is the hash algorithm (default: SHA1 per RFC 6238).
	Algorithm otp.Algorithm

	// SecretSize is the size of the generated secret in bytes.
	// Default: 20 bytes (160 bits) per RFC 6238 recommendation.
	SecretSize uint
}

// DefaultTOTPConfig returns the default TOTP configuration following RFC 6238.
func DefaultTOTPConfig() TOTPConfig {
	return TOTPConfig{
		Issuer:     "goimg",
		Period:     30,
		Digits:     otp.DigitsSix,
		Algorithm:  otp.AlgorithmSHA1, // RFC 6238 default
		SecretSize: 20,                // 160 bits
	}
}

// TOTPService provides TOTP (Time-based One-Time Password) functionality.
// It generates secrets, creates QR code URIs, and validates codes.
//
// Security properties:
//   - Uses RFC 6238 TOTP algorithm
//   - Secrets are 160 bits (20 bytes) of cryptographic randomness
//   - Supports SHA1, SHA256, SHA512 algorithms
//   - Constant-time comparison for code validation
type TOTPService struct {
	config    TOTPConfig
	encryptor *SecretEncryptor
}

// TOTPSetupResult contains the data needed for a user to set up TOTP.
type TOTPSetupResult struct {
	// Secret is the plaintext base32-encoded secret (for QR code).
	// This should be shown once during setup and never stored unencrypted.
	Secret string

	// EncryptedSecret is the AES-256-GCM encrypted secret for storage.
	EncryptedSecret []byte

	// ProvisioningURI is the otpauth:// URI for QR code generation.
	// Example: otpauth://totp/goimg:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=goimg
	ProvisioningURI string

	// Issuer is the organization name.
	Issuer string

	// AccountName is the user's account identifier (usually email).
	AccountName string
}

// NewTOTPService creates a new TOTP service with the provided configuration.
// The encryptor is used to encrypt secrets before storage.
func NewTOTPService(config TOTPConfig, encryptor *SecretEncryptor) (*TOTPService, error) {
	if encryptor == nil {
		return nil, errors.New("encryptor is required")
	}

	if config.Issuer == "" {
		config.Issuer = DefaultTOTPConfig().Issuer
	}
	if config.Period == 0 {
		config.Period = DefaultTOTPConfig().Period
	}
	if config.Digits == 0 {
		config.Digits = DefaultTOTPConfig().Digits
	}
	if config.SecretSize == 0 {
		config.SecretSize = DefaultTOTPConfig().SecretSize
	}

	return &TOTPService{
		config:    config,
		encryptor: encryptor,
	}, nil
}

// GenerateSecret creates a new TOTP secret for a user.
// Returns the setup data including encrypted secret and provisioning URI.
func (s *TOTPService) GenerateSecret(accountName string) (*TOTPSetupResult, error) {
	if accountName == "" {
		return nil, errors.New("account name is required")
	}

	// Generate cryptographically random secret
	secretBytes := make([]byte, s.config.SecretSize)
	if _, err := io.ReadFull(rand.Reader, secretBytes); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTOTPGenerationFailed, err)
	}

	// Encode as base32 (standard for TOTP secrets)
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Generate provisioning URI for QR code
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.config.Issuer,
		AccountName: accountName,
		Period:      s.config.Period,
		Digits:      s.config.Digits,
		Algorithm:   s.config.Algorithm,
		Secret:      secretBytes,
		SecretSize:  s.config.SecretSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTOTPGenerationFailed, err)
	}

	// Encrypt the secret for storage
	encryptedSecret, err := s.encryptor.Encrypt([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	return &TOTPSetupResult{
		Secret:          secret,
		EncryptedSecret: encryptedSecret,
		ProvisioningURI: key.URL(),
		Issuer:          s.config.Issuer,
		AccountName:     accountName,
	}, nil
}

// ValidateCode validates a TOTP code against an encrypted secret.
// Returns nil if the code is valid, ErrTOTPInvalidCode if invalid.
func (s *TOTPService) ValidateCode(encryptedSecret []byte, code string) error {
	// Validate code format before decryption (fail fast)
	if code == "" {
		return ErrTOTPInvalidCode
	}

	// Decrypt the secret
	secretBytes, err := s.encryptor.Decrypt(encryptedSecret)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTOTPSecretDecryptFailed, err)
	}

	secret := string(secretBytes)

	// Validate the code using constant-time comparison internally
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    s.config.Period,
		Skew:      1, // Allow 1 time step before/after (±30 seconds)
		Digits:    s.config.Digits,
		Algorithm: s.config.Algorithm,
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrTOTPValidationFailed, err)
	}

	if !valid {
		return ErrTOTPInvalidCode
	}

	return nil
}

// ValidateCodeWithSkew validates a TOTP code with custom time skew.
// Skew of 1 allows codes from ±30 seconds (one time step before/after).
func (s *TOTPService) ValidateCodeWithSkew(encryptedSecret []byte, code string, skew uint) error {
	// Decrypt the secret
	secretBytes, err := s.encryptor.Decrypt(encryptedSecret)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTOTPSecretDecryptFailed, err)
	}

	secret := string(secretBytes)

	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    s.config.Period,
		Skew:      skew,
		Digits:    s.config.Digits,
		Algorithm: s.config.Algorithm,
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrTOTPValidationFailed, err)
	}

	if !valid {
		return ErrTOTPInvalidCode
	}

	return nil
}

// GenerateCurrentCode generates the current valid TOTP code for an encrypted secret.
// This is primarily useful for testing.
func (s *TOTPService) GenerateCurrentCode(encryptedSecret []byte) (string, error) {
	// Decrypt the secret
	secretBytes, err := s.encryptor.Decrypt(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTOTPSecretDecryptFailed, err)
	}

	secret := string(secretBytes)

	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    s.config.Period,
		Digits:    s.config.Digits,
		Algorithm: s.config.Algorithm,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	return code, nil
}

// GetConfig returns the TOTP configuration.
func (s *TOTPService) GetConfig() TOTPConfig {
	return s.config
}
