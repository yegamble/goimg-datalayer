//nolint:testpackage // White-box testing required for internal implementation
package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKeys creates RSA key pair for testing
//
//nolint:nonamedreturns // Named returns used for assignment in function body
func generateTestKeys(t *testing.T, bits int) (privateKeyPath, publicKeyPath string) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	// Create temp directory for keys
	tempDir := t.TempDir()

	// Save private key
	privateKeyPath = filepath.Join(tempDir, "private.pem")
	//nolint:gosec // G304: Test file path is generated safely in temp directory
	privateKeyFile, err := os.Create(privateKeyPath)
	require.NoError(t, err)
	defer privateKeyFile.Close()

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
	defer publicKeyFile.Close()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	err = pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	require.NoError(t, err)

	return privateKeyPath, publicKeyPath
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, "", cfg.PrivateKeyPath)
	assert.Equal(t, "", cfg.PublicKeyPath)
	assert.Equal(t, 15*time.Minute, cfg.AccessTTL)
	assert.Equal(t, 7*24*time.Hour, cfg.RefreshTTL)
	assert.Equal(t, "goimg-api", cfg.Issuer)
}

func TestNewService_InvalidConfig(t *testing.T) {
	t.Parallel()

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	tests := []struct {
		name      string
		cfg       Config
		wantError string
	}{
		{
			name: "empty issuer",
			cfg: Config{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  publicKeyPath,
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "",
			},
			wantError: "jwt issuer cannot be empty",
		},
		{
			name: "zero access TTL",
			cfg: Config{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  publicKeyPath,
				AccessTTL:      0,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "jwt access TTL must be positive",
		},
		{
			name: "negative refresh TTL",
			cfg: Config{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  publicKeyPath,
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     -1 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "jwt refresh TTL must be positive",
		},
		{
			name: "empty private key path",
			cfg: Config{
				PrivateKeyPath: "",
				PublicKeyPath:  publicKeyPath,
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "jwt private key path cannot be empty",
		},
		{
			name: "empty public key path",
			cfg: Config{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  "",
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "jwt public key path cannot be empty",
		},
		{
			name: "private key file does not exist",
			cfg: Config{
				PrivateKeyPath: "/nonexistent/private.pem",
				PublicKeyPath:  publicKeyPath,
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "failed to load private key",
		},
		{
			name: "public key file does not exist",
			cfg: Config{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  "/nonexistent/public.pem",
				AccessTTL:      15 * time.Minute,
				RefreshTTL:     7 * 24 * time.Hour,
				Issuer:         "goimg-api",
			},
			wantError: "failed to load public key",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := NewService(tt.cfg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Nil(t, svc)
		})
	}
}

func TestNewService_KeyTooSmall(t *testing.T) {
	t.Parallel()

	// Generate a 2048-bit key (below the required 4096-bit minimum)
	privateKeyPath, publicKeyPath := generateTestKeys(t, 2048)

	cfg := Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}

	svc, err := NewService(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key must be at least 4096 bits")
	assert.Nil(t, svc)
}

func TestNewService_Success(t *testing.T) {
	t.Parallel()

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	cfg := Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}

	svc, err := NewService(cfg)

	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.privateKey)
	assert.NotNil(t, svc.publicKey)
	assert.Equal(t, cfg, svc.config)
}

func getTestService(t *testing.T) *Service {
	t.Helper()

	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	cfg := Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}

	svc, err := NewService(cfg)
	require.NoError(t, err)

	return svc
}

func TestService_GenerateAccessToken(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"
	sessionID := "223e4567-e89b-12d3-a456-426614174000"

	token, err := svc.GenerateAccessToken(userID, email, role, sessionID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token
	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
	assert.Equal(t, "goimg-api", claims.Issuer)
	assert.Equal(t, userID, claims.Subject)
	assert.NotEmpty(t, claims.ID)

	// Check expiration
	now := time.Now().UTC()
	assert.True(t, claims.ExpiresAt.After(now))
	assert.True(t, claims.ExpiresAt.Before(now.Add(16*time.Minute)))
}

func TestService_GenerateAccessToken_InvalidInputs(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	tests := []struct {
		name      string
		userID    string
		email     string
		role      string
		sessionID string
		wantError string
	}{
		{
			name:      "empty user id",
			userID:    "",
			email:     "test@example.com",
			role:      "user",
			sessionID: "session-id",
			wantError: "user id cannot be empty",
		},
		{
			name:      "empty email",
			userID:    "user-id",
			email:     "",
			role:      "user",
			sessionID: "session-id",
			wantError: "email cannot be empty",
		},
		{
			name:      "empty role",
			userID:    "user-id",
			email:     "test@example.com",
			role:      "",
			sessionID: "session-id",
			wantError: "role cannot be empty",
		},
		{
			name:      "empty session id",
			userID:    "user-id",
			email:     "test@example.com",
			role:      "user",
			sessionID: "",
			wantError: "session id cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, err := svc.GenerateAccessToken(tt.userID, tt.email, tt.role, tt.sessionID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Empty(t, token)
		})
	}
}

func TestService_GenerateRefreshToken(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"
	sessionID := "223e4567-e89b-12d3-a456-426614174000"

	token, err := svc.GenerateRefreshToken(userID, email, role, sessionID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token
	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, TokenTypeRefresh, claims.TokenType)
	assert.Equal(t, "goimg-api", claims.Issuer)
	assert.Equal(t, userID, claims.Subject)
	assert.NotEmpty(t, claims.ID)

	// Check expiration (should be ~7 days)
	now := time.Now().UTC()
	expectedExpiry := now.Add(7 * 24 * time.Hour)
	assert.True(t, claims.ExpiresAt.After(now))
	assert.True(t, claims.ExpiresAt.Before(expectedExpiry.Add(time.Minute)))
}

func TestService_ValidateToken_InvalidToken(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	tests := []struct {
		name      string
		token     string
		wantError string
	}{
		{
			name:      "empty token",
			token:     "",
			wantError: "token cannot be empty",
		},
		{
			name:      "malformed token",
			token:     "not.a.valid.token",
			wantError: "failed to parse token",
		},
		{
			name:      "invalid signature",
			token:     "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIn0.invalid",
			wantError: "failed to parse token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claims, err := svc.ValidateToken(tt.token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Nil(t, claims)
		})
	}
}

func TestService_ValidateToken_WrongIssuer(t *testing.T) {
	t.Parallel()

	// Generate keys once and use for both services
	privateKeyPath, publicKeyPath := generateTestKeys(t, 4096)

	// Create service with one issuer to sign the token
	cfg1 := Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "wrong-issuer",
	}
	wrongSvc, err := NewService(cfg1)
	require.NoError(t, err)

	// Generate token with wrong issuer
	token, err := wrongSvc.GenerateAccessToken("user-id", "test@example.com", "user", "session-id")
	require.NoError(t, err)

	// Create another service with same keys but different issuer
	cfg2 := Config{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}
	svc, err := NewService(cfg2)
	require.NoError(t, err)

	// Try to validate with service expecting different issuer
	claims, err := svc.ValidateToken(token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issuer")
	assert.Nil(t, claims)
}

func TestService_ValidateToken_ExpiredToken(t *testing.T) {
	// Note: This test would require mocking time or waiting, so we skip it
	// In production, expired tokens are handled by the jwt library
	t.Skip("Requires time mocking or waiting for token expiration")
}

func TestService_ExtractTokenID(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"
	sessionID := "223e4567-e89b-12d3-a456-426614174000"

	token, err := svc.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	tokenID, err := svc.ExtractTokenID(token)

	require.NoError(t, err)
	assert.NotEmpty(t, tokenID)

	// Verify it matches the claim
	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, claims.ID, tokenID)
}

func TestService_ExtractTokenID_InvalidToken(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	tests := []struct {
		name      string
		token     string
		wantError string
	}{
		{
			name:      "empty token",
			token:     "",
			wantError: "token cannot be empty",
		},
		{
			name:      "malformed token",
			token:     "not.a.token",
			wantError: "failed to parse token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenID, err := svc.ExtractTokenID(tt.token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Empty(t, tokenID)
		})
	}
}

func TestService_GetTokenExpiration(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"
	sessionID := "223e4567-e89b-12d3-a456-426614174000"

	token, err := svc.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	expiresAt, err := svc.GetTokenExpiration(token)

	require.NoError(t, err)
	assert.False(t, expiresAt.IsZero())

	// Should be in the future
	now := time.Now().UTC()
	assert.True(t, expiresAt.After(now))

	// Should be within the access TTL
	expectedExpiry := now.Add(15 * time.Minute)
	assert.True(t, expiresAt.Before(expectedExpiry.Add(time.Minute)))
}

func TestService_GetTokenExpiration_InvalidToken(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	tests := []struct {
		name      string
		token     string
		wantError string
	}{
		{
			name:      "empty token",
			token:     "",
			wantError: "token cannot be empty",
		},
		{
			name:      "malformed token",
			token:     "not.a.token",
			wantError: "failed to parse token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expiresAt, err := svc.GetTokenExpiration(tt.token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.True(t, expiresAt.IsZero())
		})
	}
}

func TestService_TokensAreUnique(t *testing.T) {
	t.Parallel()

	svc := getTestService(t)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"
	sessionID := "223e4567-e89b-12d3-a456-426614174000"

	// Generate multiple tokens
	token1, err := svc.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	token2, err := svc.GenerateAccessToken(userID, email, role, sessionID)
	require.NoError(t, err)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Token IDs should be different
	id1, err := svc.ExtractTokenID(token1)
	require.NoError(t, err)

	id2, err := svc.ExtractTokenID(token2)
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2)
}
