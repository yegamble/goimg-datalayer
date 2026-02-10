package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// SecretEncryptor errors.
var (
	ErrInvalidKeyLength   = errors.New("encryption key must be 32 bytes (256 bits)")
	ErrInvalidCiphertext  = errors.New("ciphertext is too short or invalid")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrKeyNotConfigured   = errors.New("encryption key not configured")
	ErrNonceGenerationErr = errors.New("failed to generate random nonce")
)

const (
	// aesGCMNonceSize is the standard nonce size for AES-GCM (12 bytes / 96 bits).
	aesGCMNonceSize = 12

	// aesKeySize is the required key size for AES-256 (32 bytes / 256 bits).
	aesKeySize = 32
)

// SecretEncryptor provides AES-256-GCM encryption for sensitive data like TOTP secrets.
// It uses authenticated encryption to ensure both confidentiality and integrity.
//
// Security properties:
//   - AES-256 for encryption (256-bit security)
//   - GCM mode for authenticated encryption (detects tampering)
//   - Random 12-byte nonce per encryption (prepended to ciphertext)
//   - Key should be cryptographically random 32 bytes
//
// Ciphertext format: [12-byte nonce][ciphertext][16-byte GCM tag]
type SecretEncryptor struct {
	gcm cipher.AEAD
}

// NewSecretEncryptor creates a new AES-256-GCM encryptor with the provided key.
// The key must be exactly 32 bytes (256 bits).
//
// Example key generation:
//
//	key := make([]byte, 32)
//	if _, err := rand.Read(key); err != nil {
//	    return err
//	}
//	keyBase64 := base64.StdEncoding.EncodeToString(key)
func NewSecretEncryptor(key []byte) (*SecretEncryptor, error) {
	if len(key) != aesKeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeyLength, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &SecretEncryptor{
		gcm: gcm,
	}, nil
}

// NewSecretEncryptorFromBase64 creates a new encryptor from a base64-encoded key.
// This is useful for loading keys from environment variables.
func NewSecretEncryptorFromBase64(keyBase64 string) (*SecretEncryptor, error) {
	if keyBase64 == "" {
		return nil, ErrKeyNotConfigured
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}

	return NewSecretEncryptor(key)
}

// Encrypt encrypts plaintext using AES-256-GCM and returns the ciphertext.
// The ciphertext includes the nonce prepended for use during decryption.
//
// Returns: [12-byte nonce][ciphertext][16-byte GCM tag]
func (e *SecretEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e.gcm == nil {
		return nil, ErrKeyNotConfigured
	}

	// Generate a random nonce
	nonce := make([]byte, aesGCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNonceGenerationErr, err)
	}

	// Encrypt and append ciphertext to nonce
	// Seal appends the ciphertext and GCM tag to the nonce slice
	// #nosec G407 // The nonce is randomly generated above using crypto/rand
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (e *SecretEncryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// Expects format: [12-byte nonce][ciphertext][16-byte GCM tag]
func (e *SecretEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e.gcm == nil {
		return nil, ErrKeyNotConfigured
	}

	// Minimum length: nonce (12) + GCM tag (16). Empty plaintext is valid for GCM.
	minLength := aesGCMNonceSize + e.gcm.Overhead()
	if len(ciphertext) < minLength {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and actual ciphertext
	nonce := ciphertext[:aesGCMNonceSize]
	encryptedData := ciphertext[aesGCMNonceSize:]

	// Decrypt and verify
	plaintext, err := e.gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		// Don't expose specific decryption errors (could leak info)
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// DecryptString decrypts base64-encoded ciphertext and returns the plaintext string.
func (e *SecretEncryptor) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GenerateKey generates a new cryptographically secure 256-bit key.
// Returns the key as bytes and its base64 encoding.
func GenerateKey() (key []byte, keyBase64 string, err error) {
	key = make([]byte, aesKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return key, base64.StdEncoding.EncodeToString(key), nil
}
