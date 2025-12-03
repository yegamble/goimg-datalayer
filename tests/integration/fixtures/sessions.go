package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// SessionFixture provides test session data.
type SessionFixture struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshToken     string
	RefreshTokenHash string
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	RevokedAt        *time.Time
}

// ValidSession returns a valid session fixture.
func ValidSession(t *testing.T, userID uuid.UUID) *SessionFixture {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 days from now

	return &SessionFixture{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshToken:     "refresh-token-" + uuid.New().String(),
		RefreshTokenHash: "hashed-refresh-token",
		IPAddress:        "192.168.1.100",
		UserAgent:        "Mozilla/5.0 (Test Browser)",
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		RevokedAt:        nil,
	}
}

// ExpiredSession returns an expired session fixture.
func ExpiredSession(t *testing.T, userID uuid.UUID) *SessionFixture {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(-24 * time.Hour) // Expired 24 hours ago

	return &SessionFixture{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshToken:     "expired-refresh-token-" + uuid.New().String(),
		RefreshTokenHash: "hashed-expired-token",
		IPAddress:        "192.168.1.100",
		UserAgent:        "Mozilla/5.0 (Test Browser)",
		ExpiresAt:        expiresAt,
		CreatedAt:        now.Add(-8 * 24 * time.Hour),
		RevokedAt:        nil,
	}
}

// RevokedSession returns a revoked session fixture.
func RevokedSession(t *testing.T, userID uuid.UUID) *SessionFixture {
	t.Helper()

	now := time.Now().UTC()
	revokedAt := now.Add(-1 * time.Hour)
	expiresAt := now.Add(6 * 24 * time.Hour)

	return &SessionFixture{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshToken:     "revoked-refresh-token-" + uuid.New().String(),
		RefreshTokenHash: "hashed-revoked-token",
		IPAddress:        "192.168.1.100",
		UserAgent:        "Mozilla/5.0 (Test Browser)",
		ExpiresAt:        expiresAt,
		CreatedAt:        now.Add(-24 * time.Hour),
		RevokedAt:        &revokedAt,
	}
}

// WithIPAddress returns a copy of the fixture with custom IP address.
func (f *SessionFixture) WithIPAddress(ip string) *SessionFixture {
	copy := *f
	copy.IPAddress = ip
	return &copy
}

// WithUserAgent returns a copy of the fixture with custom user agent.
func (f *SessionFixture) WithUserAgent(ua string) *SessionFixture {
	copy := *f
	copy.UserAgent = ua
	return &copy
}

// WithExpiresAt returns a copy of the fixture with custom expiration.
func (f *SessionFixture) WithExpiresAt(expiresAt time.Time) *SessionFixture {
	copy := *f
	copy.ExpiresAt = expiresAt
	return &copy
}

// UniqueSession generates a session fixture with unique ID and token.
func UniqueSession(t *testing.T, userID uuid.UUID) *SessionFixture {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour)

	return &SessionFixture{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshToken:     "unique-token-" + uuid.New().String(),
		RefreshTokenHash: "hashed-unique-token-" + uuid.New().String(),
		IPAddress:        "192.168.1.100",
		UserAgent:        "Mozilla/5.0 (Test Browser)",
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		RevokedAt:        nil,
	}
}
