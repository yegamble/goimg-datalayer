package jwt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// refreshTokenKeyPrefix is the Redis key prefix for refresh token metadata.
	//nolint:gosec // G101: This is a Redis key prefix, not credentials
	refreshTokenKeyPrefix = "goimg:refresh:"
	// tokenFamilyKeyPrefix is the Redis key prefix for token family tracking.
	//nolint:gosec // G101: This is a Redis key prefix, not credentials
	tokenFamilyKeyPrefix = "goimg:token_family:"
	// refreshTokenLength is the length of the cryptographically secure refresh token in bytes.
	refreshTokenLength = 32
)

// RefreshTokenMetadata stores metadata about a refresh token.
type RefreshTokenMetadata struct {
	TokenHash  string    `json:"token_hash"`  // SHA-256 hash of the refresh token
	UserID     string    `json:"user_id"`     // User UUID
	SessionID  string    `json:"session_id"`  // Session UUID
	FamilyID   string    `json:"family_id"`   // Token family ID for rotation tracking
	IssuedAt   time.Time `json:"issued_at"`   // Token issue timestamp
	ExpiresAt  time.Time `json:"expires_at"`  // Token expiration timestamp
	IP         string    `json:"ip"`          // IP address when token was issued
	UserAgent  string    `json:"user_agent"`  // User agent when token was issued
	ParentHash string    `json:"parent_hash"` // Hash of parent token (for rotation chain)
	Used       bool      `json:"used"`        // Whether token has been used (for replay detection)
}

// RefreshTokenService manages refresh token generation, rotation, and replay detection.
type RefreshTokenService struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewRefreshTokenService creates a new refresh token service.
func NewRefreshTokenService(redisClient *redis.Client, ttl time.Duration) *RefreshTokenService {
	return &RefreshTokenService{
		redis: redisClient,
		ttl:   ttl,
	}
}

// GenerateToken generates a cryptographically secure refresh token.
// Returns the plaintext token and its metadata.
func (s *RefreshTokenService) GenerateToken(
	ctx context.Context, userID, sessionID, familyID, parentHash, ip, userAgent string,
) (string, *RefreshTokenMetadata, error) {
	if userID == "" {
		return "", nil, fmt.Errorf("user id cannot be empty")
	}

	if sessionID == "" {
		return "", nil, fmt.Errorf("session id cannot be empty")
	}

	// Generate cryptographically secure random token
	tokenBytes := make([]byte, refreshTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Encode token as base64 for transport
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token for storage (never store plaintext)
	tokenHash := hashToken(token)

	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	metadata := &RefreshTokenMetadata{
		TokenHash:  tokenHash,
		UserID:     userID,
		SessionID:  sessionID,
		FamilyID:   familyID,
		IssuedAt:   now,
		ExpiresAt:  expiresAt,
		IP:         ip,
		UserAgent:  userAgent,
		ParentHash: parentHash,
		Used:       false,
	}

	// Store metadata in Redis
	key := refreshTokenKeyPrefix + tokenHash
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", nil, fmt.Errorf("failed to serialize token metadata: %w", err)
	}

	err = s.redis.Set(ctx, key, data, s.ttl).Err()
	if err != nil {
		return "", nil, fmt.Errorf("failed to store token metadata: %w", err)
	}

	// Add token to family tracking
	if familyID != "" {
		familyKey := tokenFamilyKeyPrefix + familyID
		err = s.redis.SAdd(ctx, familyKey, tokenHash).Err()
		if err != nil {
			return "", nil, fmt.Errorf("failed to add token to family: %w", err)
		}

		// Set expiration on family set
		err = s.redis.Expire(ctx, familyKey, s.ttl).Err()
		if err != nil {
			return "", nil, fmt.Errorf("failed to set expiration on family: %w", err)
		}
	}

	return token, metadata, nil
}

// ValidateToken validates a refresh token and returns its metadata.
// This performs constant-time comparison to prevent timing attacks.
func (s *RefreshTokenService) ValidateToken(ctx context.Context, token string) (*RefreshTokenMetadata, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Hash the provided token
	tokenHash := hashToken(token)

	// Retrieve metadata from Redis
	key := refreshTokenKeyPrefix + tokenHash
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("invalid or expired refresh token")
		}
		return nil, fmt.Errorf("failed to retrieve token metadata: %w", err)
	}

	var metadata RefreshTokenMetadata
	err = json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize token metadata: %w", err)
	}

	// Verify token hash using constant-time comparison
	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(metadata.TokenHash)) != 1 {
		return nil, fmt.Errorf("token hash mismatch")
	}

	// Check if token has been used (replay attack detection)
	if metadata.Used {
		// Token replay detected! Revoke entire token family
		if err := s.RevokeFamily(ctx, metadata.FamilyID); err != nil {
			return nil, fmt.Errorf("failed to revoke token family after replay detection: %w", err)
		}
		return nil, fmt.Errorf("refresh token has already been used (replay attack detected)")
	}

	// Check expiration
	now := time.Now().UTC()
	if now.After(metadata.ExpiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	return &metadata, nil
}

// MarkAsUsed marks a refresh token as used for replay detection.
// Once marked, the token cannot be used again.
func (s *RefreshTokenService) MarkAsUsed(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	tokenHash := hashToken(token)
	key := refreshTokenKeyPrefix + tokenHash

	// Retrieve current metadata
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("token not found")
		}
		return fmt.Errorf("failed to retrieve token metadata: %w", err)
	}

	var metadata RefreshTokenMetadata
	err = json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize token metadata: %w", err)
	}

	// Mark as used
	metadata.Used = true

	// Update metadata in Redis
	updatedData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize token metadata: %w", err)
	}

	// Calculate remaining TTL
	now := time.Now().UTC()
	ttl := metadata.ExpiresAt.Sub(now)
	if ttl <= 0 {
		return fmt.Errorf("token has expired")
	}

	err = s.redis.Set(ctx, key, updatedData, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to update token metadata: %w", err)
	}

	return nil
}

// RevokeToken revokes a single refresh token.
func (s *RefreshTokenService) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	tokenHash := hashToken(token)
	key := refreshTokenKeyPrefix + tokenHash

	// Retrieve metadata to get family ID
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil // Token already expired or revoked
		}
		return fmt.Errorf("failed to retrieve token metadata: %w", err)
	}

	var metadata RefreshTokenMetadata
	err = json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize token metadata: %w", err)
	}

	// Delete token
	err = s.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	// Remove from family tracking
	if metadata.FamilyID != "" {
		familyKey := tokenFamilyKeyPrefix + metadata.FamilyID
		err = s.redis.SRem(ctx, familyKey, tokenHash).Err()
		if err != nil {
			return fmt.Errorf("failed to remove token from family: %w", err)
		}
	}

	return nil
}

// RevokeFamily revokes all tokens in a token family.
// This is used when a replay attack is detected.
func (s *RefreshTokenService) RevokeFamily(ctx context.Context, familyID string) error {
	if familyID == "" {
		return fmt.Errorf("family id cannot be empty")
	}

	familyKey := tokenFamilyKeyPrefix + familyID

	// Get all token hashes in the family
	tokenHashes, err := s.redis.SMembers(ctx, familyKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get token family: %w", err)
	}

	// Delete each token
	for _, tokenHash := range tokenHashes {
		key := refreshTokenKeyPrefix + tokenHash
		err := s.redis.Del(ctx, key).Err()
		if err != nil {
			return fmt.Errorf("failed to delete token %s: %w", tokenHash, err)
		}
	}

	// Delete family set
	err = s.redis.Del(ctx, familyKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete token family: %w", err)
	}

	return nil
}

// DetectAnomalies checks for suspicious behavior in token usage.
// Returns true if anomalies are detected (e.g., IP or user agent change).
func (s *RefreshTokenService) DetectAnomalies(metadata *RefreshTokenMetadata, currentIP, currentUserAgent string) bool {
	if metadata == nil {
		return false
	}

	// Check for IP address change
	if metadata.IP != "" && currentIP != "" && metadata.IP != currentIP {
		return true
	}

	// Check for user agent change
	if metadata.UserAgent != "" && currentUserAgent != "" && metadata.UserAgent != currentUserAgent {
		return true
	}

	return false
}

// hashToken creates a SHA-256 hash of a token for secure storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}
