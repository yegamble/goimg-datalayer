package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshTokenService(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	ttl := 7 * 24 * time.Hour
	service := NewRefreshTokenService(client, ttl)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.redis)
	assert.Equal(t, ttl, service.ttl)
}

func TestRefreshTokenService_GenerateToken(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()
	ip := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	token, metadata, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", ip, userAgent)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, metadata)
	assert.Equal(t, userID, metadata.UserID)
	assert.Equal(t, sessionID, metadata.SessionID)
	assert.Equal(t, familyID, metadata.FamilyID)
	assert.Equal(t, ip, metadata.IP)
	assert.Equal(t, userAgent, metadata.UserAgent)
	assert.False(t, metadata.Used)
	assert.NotEmpty(t, metadata.TokenHash)
	assert.False(t, metadata.IssuedAt.IsZero())
	assert.False(t, metadata.ExpiresAt.IsZero())
	assert.True(t, metadata.ExpiresAt.After(metadata.IssuedAt))

	// Clean up
	service.RevokeToken(ctx, token)
	service.RevokeFamily(ctx, familyID)
}

func TestRefreshTokenService_GenerateToken_InvalidInputs(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		sessionID string
		wantError string
	}{
		{
			name:      "empty user id",
			userID:    "",
			sessionID: "session-123",
			wantError: "user id cannot be empty",
		},
		{
			name:      "empty session id",
			userID:    "user-123",
			sessionID: "",
			wantError: "session id cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, metadata, err := service.GenerateToken(ctx, tt.userID, tt.sessionID, "", "", "", "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Empty(t, token)
			assert.Nil(t, metadata)
		})
	}
}

func TestRefreshTokenService_ValidateToken(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate token
	token, originalMetadata, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up
	defer service.RevokeToken(ctx, token)
	defer service.RevokeFamily(ctx, familyID)

	// Validate token
	metadata, err := service.ValidateToken(ctx, token)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, originalMetadata.UserID, metadata.UserID)
	assert.Equal(t, originalMetadata.SessionID, metadata.SessionID)
	assert.Equal(t, originalMetadata.FamilyID, metadata.FamilyID)
	assert.False(t, metadata.Used)
}

func TestRefreshTokenService_ValidateToken_InvalidToken(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

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
			name:      "nonexistent token",
			token:     "nonexistent-token-12345",
			wantError: "invalid or expired refresh token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := service.ValidateToken(ctx, tt.token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Nil(t, metadata)
		})
	}
}

func TestRefreshTokenService_MarkAsUsed(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate token
	token, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up
	defer service.RevokeToken(ctx, token)
	defer service.RevokeFamily(ctx, familyID)

	// Token should not be used initially
	metadata, err := service.ValidateToken(ctx, token)
	require.NoError(t, err)
	assert.False(t, metadata.Used)

	// Mark as used
	err = service.MarkAsUsed(ctx, token)
	require.NoError(t, err)

	// Token should be marked as used
	metadata, err = service.ValidateToken(ctx, token)
	require.NoError(t, err)
	assert.True(t, metadata.Used)
}

func TestRefreshTokenService_MarkAsUsed_EmptyToken(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	err := service.MarkAsUsed(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token cannot be empty")
}

func TestRefreshTokenService_ReplayDetection(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate token
	token, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up
	defer service.RevokeFamily(ctx, familyID)

	// Mark as used
	err = service.MarkAsUsed(ctx, token)
	require.NoError(t, err)

	// Try to validate the used token (replay attack)
	metadata, err := service.ValidateToken(ctx, token)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "replay attack detected")
	assert.Nil(t, metadata)

	// Entire token family should be revoked
	// Try to generate another token in the same family and validate
	token2, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// This token should not validate because the family was revoked
	metadata, err = service.ValidateToken(ctx, token2)
	require.Error(t, err)
}

func TestRefreshTokenService_RevokeToken(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate token
	token, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up family
	defer service.RevokeFamily(ctx, familyID)

	// Token should be valid
	_, err = service.ValidateToken(ctx, token)
	require.NoError(t, err)

	// Revoke token
	err = service.RevokeToken(ctx, token)
	require.NoError(t, err)

	// Token should no longer be valid
	_, err = service.ValidateToken(ctx, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestRefreshTokenService_RevokeToken_EmptyToken(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	err := service.RevokeToken(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token cannot be empty")
}

func TestRefreshTokenService_RevokeFamily(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate multiple tokens in the same family
	tokens := make([]string, 3)
	for i := 0; i < 3; i++ {
		token, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
		require.NoError(t, err)
		tokens[i] = token
	}

	// All tokens should be valid
	for _, token := range tokens {
		_, err := service.ValidateToken(ctx, token)
		require.NoError(t, err)
	}

	// Revoke entire family
	err := service.RevokeFamily(ctx, familyID)
	require.NoError(t, err)

	// All tokens should be invalid
	for _, token := range tokens {
		_, err := service.ValidateToken(ctx, token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired")
	}
}

func TestRefreshTokenService_RevokeFamily_EmptyFamilyID(t *testing.T) {
	t.Parallel()

	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	err := service.RevokeFamily(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "family id cannot be empty")
}

func TestRefreshTokenService_DetectAnomalies(t *testing.T) {
	t.Parallel()

	service := NewRefreshTokenService(nil, 7*24*time.Hour)

	tests := []struct {
		name             string
		metadata         *RefreshTokenMetadata
		currentIP        string
		currentUserAgent string
		wantAnomaly      bool
	}{
		{
			name:             "nil metadata",
			metadata:         nil,
			currentIP:        "192.168.1.1",
			currentUserAgent: "Mozilla/5.0",
			wantAnomaly:      false,
		},
		{
			name: "same IP and user agent",
			metadata: &RefreshTokenMetadata{
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "192.168.1.1",
			currentUserAgent: "Mozilla/5.0",
			wantAnomaly:      false,
		},
		{
			name: "different IP",
			metadata: &RefreshTokenMetadata{
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "10.0.0.1",
			currentUserAgent: "Mozilla/5.0",
			wantAnomaly:      true,
		},
		{
			name: "different user agent",
			metadata: &RefreshTokenMetadata{
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "192.168.1.1",
			currentUserAgent: "Chrome/90.0",
			wantAnomaly:      true,
		},
		{
			name: "different IP and user agent",
			metadata: &RefreshTokenMetadata{
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "10.0.0.1",
			currentUserAgent: "Chrome/90.0",
			wantAnomaly:      true,
		},
		{
			name: "empty original IP",
			metadata: &RefreshTokenMetadata{
				IP:        "",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "192.168.1.1",
			currentUserAgent: "Mozilla/5.0",
			wantAnomaly:      false,
		},
		{
			name: "empty current IP",
			metadata: &RefreshTokenMetadata{
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			currentIP:        "",
			currentUserAgent: "Mozilla/5.0",
			wantAnomaly:      false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hasAnomaly := service.DetectAnomalies(tt.metadata, tt.currentIP, tt.currentUserAgent)

			assert.Equal(t, tt.wantAnomaly, hasAnomaly)
		})
	}
}

func TestRefreshTokenService_TokenRotation(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	service := NewRefreshTokenService(client, 7*24*time.Hour)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate initial token
	token1, metadata1, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up
	defer service.RevokeFamily(ctx, familyID)

	// Mark first token as used
	err = service.MarkAsUsed(ctx, token1)
	require.NoError(t, err)

	// Generate rotated token with parent reference
	token2, metadata2, err := service.GenerateToken(ctx, userID, sessionID, familyID, metadata1.TokenHash, "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Second token should have parent hash
	assert.Equal(t, metadata1.TokenHash, metadata2.ParentHash)

	// Second token should be valid
	_, err = service.ValidateToken(ctx, token2)
	require.NoError(t, err)

	// First token should fail validation (used)
	_, err = service.ValidateToken(ctx, token1)
	require.Error(t, err)
}

func TestRefreshTokenService_TokenExpiration(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	// Create service with short TTL for testing
	service := NewRefreshTokenService(client, 2*time.Second)
	ctx := context.Background()

	userID := uuid.New().String()
	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	// Generate token
	token, _, err := service.GenerateToken(ctx, userID, sessionID, familyID, "", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Clean up
	defer service.RevokeFamily(ctx, familyID)

	// Token should be valid initially
	_, err = service.ValidateToken(ctx, token)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(2500 * time.Millisecond)

	// Token should be expired
	_, err = service.ValidateToken(ctx, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestHashToken(t *testing.T) {
	t.Parallel()

	token := "test-token-12345"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	// Same token should produce same hash
	assert.Equal(t, hash1, hash2)

	// Different token should produce different hash
	hash3 := hashToken("different-token")
	assert.NotEqual(t, hash1, hash3)

	// Hash should not be empty
	assert.NotEmpty(t, hash1)
}
