package unit_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// generateTestKeys creates RSA key pair for testing
func generateTestKeys(t *testing.T, bits int) (privateKeyPath, publicKeyPath string) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	// Create temp directory for keys
	tempDir := t.TempDir()

	// Save private key
	privateKeyPath = filepath.Join(tempDir, "private.pem")
	privateKeyFile, err := os.Create(privateKeyPath)
	require.NoError(t, err)
	defer func() {
		if err := privateKeyFile.Close(); err != nil {
			t.Logf("failed to close private key file: %v", err)
		}
	}()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(privateKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	require.NoError(t, err)

	// Save public key
	publicKeyPath = filepath.Join(tempDir, "public.pem")
	publicKeyFile, err := os.Create(publicKeyPath)
	require.NoError(t, err)
	defer func() {
		if err := publicKeyFile.Close(); err != nil {
			t.Logf("failed to close public key file: %v", err)
		}
	}()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	err = pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	require.NoError(t, err)

	return privateKeyPath, publicKeyPath
}

func getTestService(t *testing.T) *jwt.Service {
	t.Helper()

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	cfg := jwt.Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}

	svc, err := jwt.NewService(cfg)
	require.NoError(t, err)

	return svc
}

// TestJWTService_GenerateAccessToken tests generating an access token.
func TestJWTService_GenerateAccessToken(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange
	userID := uuid.New().String()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New().String()

	// Act
	token, err := jwtService.GenerateAccessToken(userID, email, role, sessionID)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, len(token), 100, "JWT should be reasonably long")
}

// TestJWTService_GenerateRefreshToken tests generating a refresh token.
func TestJWTService_GenerateRefreshToken(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange
	userID := uuid.New().String()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New().String()

	// Act
	token, err := jwtService.GenerateRefreshToken(userID, email, role, sessionID)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, len(token), 100, "JWT should be reasonably long")
}

// TestJWTService_ValidateAccessToken tests validating a valid access token.
func TestJWTService_ValidateAccessToken(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange - generate token
	userID := uuid.New().String()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New().String()

	token, err := jwtService.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	// Act - validate token
	claims, err := jwtService.ValidateToken(token)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, jwt.TokenTypeAccess, claims.TokenType)
}

// TestJWTService_ValidateRefreshToken tests validating a valid refresh token.
func TestJWTService_ValidateRefreshToken(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange - generate refresh token
	userID := uuid.New().String()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New().String()

	token, err := jwtService.GenerateRefreshToken(userID, email, role, sessionID)
	require.NoError(t, err)

	// Act - validate token
	claims, err := jwtService.ValidateToken(token)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, jwt.TokenTypeRefresh, claims.TokenType)
}

// TestJWTService_ExpiredToken tests that expired tokens are rejected.
func TestJWTService_ExpiredToken(t *testing.T) {
	t.Parallel()
	// This test requires mocking time or waiting.
	// Since we can't mock time easily without DI of a clock, and waiting is flaky/slow,
	// we will create a service with very short TTL and sleep.

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	cfg := jwt.Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      1 * time.Second, // 1 second TTL
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}

	jwtService, err := jwt.NewService(cfg)
	require.NoError(t, err)

	// Arrange - generate token
	userID := uuid.New().String()
	token, err := jwtService.GenerateAccessToken(userID, "test@example.com", "user", uuid.New().String())
	require.NoError(t, err)

	// Wait for token to expire + buffer
	time.Sleep(2 * time.Second)

	// Act - validate expired token
	_, err = jwtService.ValidateToken(token)

	// Assert - should return error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestJWTService_InvalidSignature tests that tokens with invalid signatures are rejected.
func TestJWTService_InvalidSignature(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange - create a token with wrong key
	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)
	differentKeyCfg := jwt.Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}
	differentKeyService, err := jwt.NewService(differentKeyCfg)
	require.NoError(t, err)

	userID := uuid.New().String()
	token, err := differentKeyService.GenerateAccessToken(userID, "test@example.com", "user", uuid.New().String())
	require.NoError(t, err)

	// Act - validate with original service (different keys)
	_, err = jwtService.ValidateToken(token)

	// Assert - should fail signature verification
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")
}

// TestJWTService_TokenClaims tests that all expected claims are present.
func TestJWTService_TokenClaims(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange
	userID := uuid.New().String()
	email := "test@example.com"
	role := "admin"
	sessionID := uuid.New().String()

	// Act
	token, err := jwtService.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	claims, err := jwtService.ValidateToken(token)
	require.NoError(t, err)

	// Assert - verify all standard claims
	assert.NotEmpty(t, claims.ID, "JTI (token ID) should be set")
	assert.Equal(t, "goimg-api", claims.Issuer, "Issuer should match")
	// Audience is not set in current implementation, skipping check
	assert.NotZero(t, claims.IssuedAt, "IssuedAt should be set")
	assert.NotZero(t, claims.ExpiresAt, "ExpiresAt should be set")
	assert.Greater(t, claims.ExpiresAt.Time.Unix(), claims.IssuedAt.Time.Unix(), "ExpiresAt should be after IssuedAt")

	// Assert - verify custom claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
}

// TestJWTService_MalformedToken tests handling of malformed tokens.
func TestJWTService_MalformedToken(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"random string", "not-a-jwt-token"},
		{"incomplete JWT", "header.payload"},
		{"too many parts", "header.payload.signature.extra"},
		{"invalid base64", "!!!.!!!.!!!"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			_, err := jwtService.ValidateToken(tt.token)

			// Assert
			require.Error(t, err, "should reject malformed token")
		})
	}
}

// TestJWTService_WrongIssuer tests that tokens with wrong issuer are rejected.
func TestJWTService_WrongIssuer(t *testing.T) {
	t.Parallel()

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	cfg1 := jwt.Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "issuer1",
	}
	service1, err := jwt.NewService(cfg1)
	require.NoError(t, err)

	cfg2 := jwt.Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "issuer2",
	}
	service2, err := jwt.NewService(cfg2)
	require.NoError(t, err)

	// Arrange - generate token with issuer1
	userID := uuid.New().String()
	token, err := service1.GenerateAccessToken(userID, "test@example.com", "user", uuid.New().String())
	require.NoError(t, err)

	// Act - validate with service2 (different issuer)
	_, err = service2.ValidateToken(token)

	// Assert - should fail issuer check
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issuer")
}

// TestJWTService_UniqueJTI tests that each token has a unique JTI.
func TestJWTService_UniqueJTI(t *testing.T) {
	t.Parallel()

	jwtService := getTestService(t)

	// Arrange
	userID := uuid.New().String()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New().String()

	// Act - generate multiple tokens
	token1, err := jwtService.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)
	token2, err := jwtService.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	claims1, err := jwtService.ValidateToken(token1)
	require.NoError(t, err)
	claims2, err := jwtService.ValidateToken(token2)
	require.NoError(t, err)

	// Assert - JTIs should be different
	assert.NotEqual(t, claims1.ID, claims2.ID, "each token should have unique JTI")
}
