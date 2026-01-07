package security

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestNewSecretEncryptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 32-byte key",
			keyLen:  32,
			wantErr: false,
		},
		{
			name:    "key too short (16 bytes)",
			keyLen:  16,
			wantErr: true,
			errMsg:  "encryption key must be 32 bytes",
		},
		{
			name:    "key too long (64 bytes)",
			keyLen:  64,
			wantErr: true,
			errMsg:  "encryption key must be 32 bytes",
		},
		{
			name:    "empty key",
			keyLen:  0,
			wantErr: true,
			errMsg:  "encryption key must be 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := make([]byte, tt.keyLen)
			if tt.keyLen > 0 {
				if _, err := rand.Read(key); err != nil {
					t.Fatalf("failed to generate test key: %v", err)
				}
			}

			encryptor, err := NewSecretEncryptor(key)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if encryptor == nil {
				t.Error("expected encryptor but got nil")
			}
		})
	}
}

func TestNewSecretEncryptorFromBase64(t *testing.T) {
	t.Parallel()

	// Generate a valid key
	validKey := make([]byte, 32)
	if _, err := rand.Read(validKey); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	validKeyBase64 := base64.StdEncoding.EncodeToString(validKey)

	tests := []struct {
		name      string
		keyBase64 string
		wantErr   bool
	}{
		{
			name:      "valid base64 key",
			keyBase64: validKeyBase64,
			wantErr:   false,
		},
		{
			name:      "empty key",
			keyBase64: "",
			wantErr:   true,
		},
		{
			name:      "invalid base64",
			keyBase64: "not-valid-base64!!!",
			wantErr:   true,
		},
		{
			name:      "valid base64 but wrong length",
			keyBase64: base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encryptor, err := NewSecretEncryptorFromBase64(tt.keyBase64)
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

			if encryptor == nil {
				t.Error("expected encryptor but got nil")
			}
		})
	}
}

func TestSecretEncryptor_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	// Create encryptor with random key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "short text",
			plaintext: []byte("secret"),
		},
		{
			name:      "TOTP secret (20 bytes)",
			plaintext: make([]byte, 20), // Standard TOTP secret length
		},
		{
			name:      "long text",
			plaintext: bytes.Repeat([]byte("a"), 1000),
		},
		{
			name:      "empty",
			plaintext: []byte{},
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0xFF, 0x01, 0xFE, 0x02, 0xFD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Ciphertext should be longer than plaintext (nonce + tag overhead)
			expectedMinLen := len(tt.plaintext) + aesGCMNonceSize + 16 // 16 = GCM tag
			if len(ciphertext) < expectedMinLen {
				t.Errorf("ciphertext too short: got %d, want at least %d", len(ciphertext), expectedMinLen)
			}

			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("decrypted text doesn't match original: got %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestSecretEncryptor_EncryptDecryptString(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	testCases := []string{
		"JBSWY3DPEHPK3PXP", // Example TOTP secret
		"",                 // Empty string
		"Hello, World! 👋",  // Unicode
	}

	for _, plaintext := range testCases {
		t.Run(plaintext, func(t *testing.T) {
			t.Parallel()

			ciphertext, err := encryptor.EncryptString(plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Should be valid base64
			if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
				t.Errorf("ciphertext is not valid base64: %v", err)
			}

			decrypted, err := encryptor.DecryptString(ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("decrypted text doesn't match: got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestSecretEncryptor_UniqueNonces(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := []byte("test secret")

	// Encrypt the same plaintext multiple times
	ciphertexts := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		ct, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}
		ciphertexts[i] = ct
	}

	// All ciphertexts should be different (due to random nonce)
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			if bytes.Equal(ciphertexts[i], ciphertexts[j]) {
				t.Errorf("ciphertexts %d and %d are identical (nonce reuse detected)", i, j)
			}
		}
	}
}

func TestSecretEncryptor_TamperDetection(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := []byte("secret data")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with ciphertext by flipping a bit
	tamperedCiphertext := make([]byte, len(ciphertext))
	copy(tamperedCiphertext, ciphertext)
	tamperedCiphertext[len(tamperedCiphertext)-5] ^= 0x01 // Flip a bit in the GCM tag area

	// Decryption should fail
	_, err = encryptor.Decrypt(tamperedCiphertext)
	if err == nil {
		t.Error("expected decryption to fail with tampered ciphertext")
	}
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestSecretEncryptor_WrongKey(t *testing.T) {
	t.Parallel()

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatalf("failed to generate test key 1: %v", err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatalf("failed to generate test key 2: %v", err)
	}

	encryptor1, _ := NewSecretEncryptor(key1)
	encryptor2, _ := NewSecretEncryptor(key2)

	plaintext := []byte("secret data")
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Decryption with wrong key should fail
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("expected decryption to fail with wrong key")
	}
}

func TestSecretEncryptor_InvalidCiphertext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	encryptor, _ := NewSecretEncryptor(key)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
		},
		{
			name:       "too short (only nonce)",
			ciphertext: make([]byte, 12),
		},
		{
			name:       "just under minimum length",
			ciphertext: make([]byte, 28), // nonce(12) + 1 + tag(16) - 1 = 28
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := encryptor.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestGenerateKey(t *testing.T) {
	t.Parallel()

	key, keyBase64, err := GenerateKey()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	// Key should be 32 bytes
	if len(key) != 32 {
		t.Errorf("key length: got %d, want 32", len(key))
	}

	// Base64 should decode back to the same key
	decoded, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		t.Fatalf("failed to decode base64 key: %v", err)
	}

	if !bytes.Equal(decoded, key) {
		t.Error("decoded key doesn't match original")
	}

	// Generate another key - should be different
	key2, _, err := GenerateKey()
	if err != nil {
		t.Fatalf("second key generation failed: %v", err)
	}

	if bytes.Equal(key, key2) {
		t.Error("two generated keys are identical (extremely unlikely)")
	}
}
