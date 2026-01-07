package security

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func createTestEncryptor(t *testing.T) *SecretEncryptor {
	t.Helper()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	return encryptor
}

func TestNewTOTPService(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)

	tests := []struct {
		name      string
		config    TOTPConfig
		encryptor *SecretEncryptor
		wantErr   bool
	}{
		{
			name:      "valid with default config",
			config:    DefaultTOTPConfig(),
			encryptor: encryptor,
			wantErr:   false,
		},
		{
			name:      "valid with empty config (uses defaults)",
			config:    TOTPConfig{},
			encryptor: encryptor,
			wantErr:   false,
		},
		{
			name:      "nil encryptor",
			config:    DefaultTOTPConfig(),
			encryptor: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, err := NewTOTPService(tt.config, tt.encryptor)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if service == nil {
				t.Error("expected service but got nil")
			}
		})
	}
}

func TestTOTPService_GenerateSecret(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, err := NewTOTPService(DefaultTOTPConfig(), encryptor)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name        string
		accountName string
		wantErr     bool
	}{
		{
			name:        "valid account",
			accountName: "user@example.com",
			wantErr:     false,
		},
		{
			name:        "empty account name",
			accountName: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := service.GenerateSecret(tt.accountName)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify result fields
			if result.Secret == "" {
				t.Error("secret should not be empty")
			}

			if len(result.EncryptedSecret) == 0 {
				t.Error("encrypted secret should not be empty")
			}

			if result.ProvisioningURI == "" {
				t.Error("provisioning URI should not be empty")
			}

			// Verify provisioning URI format
			if !strings.HasPrefix(result.ProvisioningURI, "otpauth://totp/") {
				t.Errorf("invalid provisioning URI format: %s", result.ProvisioningURI)
			}

			if !strings.Contains(result.ProvisioningURI, tt.accountName) {
				t.Errorf("provisioning URI should contain account name: %s", result.ProvisioningURI)
			}

			if !strings.Contains(result.ProvisioningURI, "issuer=goimg") {
				t.Errorf("provisioning URI should contain issuer: %s", result.ProvisioningURI)
			}

			// Verify issuer and account
			if result.Issuer != "goimg" {
				t.Errorf("issuer: got %q, want %q", result.Issuer, "goimg")
			}

			if result.AccountName != tt.accountName {
				t.Errorf("account name: got %q, want %q", result.AccountName, tt.accountName)
			}
		})
	}
}

func TestTOTPService_GenerateSecret_UniqueSecrets(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, _ := NewTOTPService(DefaultTOTPConfig(), encryptor)

	// Generate multiple secrets for the same account
	secrets := make([]string, 10)
	for i := 0; i < 10; i++ {
		result, err := service.GenerateSecret("user@example.com")
		if err != nil {
			t.Fatalf("failed to generate secret %d: %v", i, err)
		}
		secrets[i] = result.Secret
	}

	// All secrets should be unique
	for i := 0; i < len(secrets); i++ {
		for j := i + 1; j < len(secrets); j++ {
			if secrets[i] == secrets[j] {
				t.Errorf("secrets %d and %d are identical", i, j)
			}
		}
	}
}

func TestTOTPService_ValidateCode(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, _ := NewTOTPService(DefaultTOTPConfig(), encryptor)

	// Generate a secret
	result, err := service.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Generate a valid code using the plaintext secret
	validCode, err := totp.GenerateCode(result.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to generate valid code: %v", err)
	}

	tests := []struct {
		name            string
		encryptedSecret []byte
		code            string
		wantErr         error
	}{
		{
			name:            "valid code",
			encryptedSecret: result.EncryptedSecret,
			code:            validCode,
			wantErr:         nil,
		},
		{
			name:            "invalid code",
			encryptedSecret: result.EncryptedSecret,
			code:            "000000",
			wantErr:         ErrTOTPInvalidCode,
		},
		{
			name:            "wrong format code",
			encryptedSecret: result.EncryptedSecret,
			code:            "abc123",
			wantErr:         ErrTOTPInvalidCode,
		},
		{
			name:            "empty code",
			encryptedSecret: result.EncryptedSecret,
			code:            "",
			wantErr:         ErrTOTPInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := service.ValidateCode(tt.encryptedSecret, tt.code)
			if tt.wantErr != nil {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err != tt.wantErr && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTOTPService_ValidateCode_InvalidEncryptedSecret(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, _ := NewTOTPService(DefaultTOTPConfig(), encryptor)

	// Try to validate with invalid encrypted secret
	err := service.ValidateCode([]byte("invalid-encrypted-secret"), "123456")
	if err == nil {
		t.Error("expected error for invalid encrypted secret")
	}

	if !strings.Contains(err.Error(), "decrypt") {
		t.Errorf("expected decryption error, got: %v", err)
	}
}

func TestTOTPService_GenerateCurrentCode(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, _ := NewTOTPService(DefaultTOTPConfig(), encryptor)

	// Generate a secret
	result, err := service.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Generate code from encrypted secret
	code, err := service.GenerateCurrentCode(result.EncryptedSecret)
	if err != nil {
		t.Fatalf("failed to generate current code: %v", err)
	}

	// Code should be 6 digits
	if len(code) != 6 {
		t.Errorf("code length: got %d, want 6", len(code))
	}

	// Code should validate successfully
	err = service.ValidateCode(result.EncryptedSecret, code)
	if err != nil {
		t.Errorf("generated code should be valid: %v", err)
	}
}

func TestTOTPService_ValidateCodeWithSkew(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)
	service, _ := NewTOTPService(DefaultTOTPConfig(), encryptor)

	result, err := service.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Generate code for current time
	currentCode, _ := service.GenerateCurrentCode(result.EncryptedSecret)

	// Should validate with skew=0 (exact time)
	err = service.ValidateCodeWithSkew(result.EncryptedSecret, currentCode, 0)
	if err != nil {
		t.Errorf("code should validate with skew=0: %v", err)
	}

	// Should validate with larger skew
	err = service.ValidateCodeWithSkew(result.EncryptedSecret, currentCode, 2)
	if err != nil {
		t.Errorf("code should validate with skew=2: %v", err)
	}
}

func TestTOTPService_CustomConfig(t *testing.T) {
	t.Parallel()

	encryptor := createTestEncryptor(t)

	customConfig := TOTPConfig{
		Issuer:     "myapp",
		Period:     30,
		Digits:     otp.DigitsEight,
		Algorithm:  otp.AlgorithmSHA256,
		SecretSize: 32, // 256 bits
	}

	service, err := NewTOTPService(customConfig, encryptor)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	result, err := service.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Verify custom issuer
	if result.Issuer != "myapp" {
		t.Errorf("issuer: got %q, want %q", result.Issuer, "myapp")
	}

	if !strings.Contains(result.ProvisioningURI, "issuer=myapp") {
		t.Errorf("provisioning URI should contain custom issuer: %s", result.ProvisioningURI)
	}

	// Generate and validate code
	code, err := service.GenerateCurrentCode(result.EncryptedSecret)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// 8-digit code
	if len(code) != 8 {
		t.Errorf("code length: got %d, want 8", len(code))
	}

	err = service.ValidateCode(result.EncryptedSecret, code)
	if err != nil {
		t.Errorf("code should validate: %v", err)
	}
}

func TestDefaultTOTPConfig(t *testing.T) {
	t.Parallel()

	config := DefaultTOTPConfig()

	if config.Issuer != "goimg" {
		t.Errorf("issuer: got %q, want %q", config.Issuer, "goimg")
	}

	if config.Period != 30 {
		t.Errorf("period: got %d, want 30", config.Period)
	}

	if config.Digits != otp.DigitsSix {
		t.Errorf("digits: got %v, want DigitsSix", config.Digits)
	}

	if config.Algorithm != otp.AlgorithmSHA1 {
		t.Errorf("algorithm: got %v, want AlgorithmSHA1", config.Algorithm)
	}

	if config.SecretSize != 20 {
		t.Errorf("secret size: got %d, want 20", config.SecretSize)
	}
}
