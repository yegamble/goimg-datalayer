package identity_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"testing"
	"time"
)

func TestPasswordResetToken_IsExpired(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	expiredToken := identity.PasswordResetToken{ExpiresAt: now.Add(-1 * time.Hour)}
	validToken := identity.PasswordResetToken{ExpiresAt: now.Add(1 * time.Hour)}

	assert.True(t, expiredToken.IsExpired())
	assert.False(t, validToken.IsExpired())
}

func TestPasswordResetToken_IsUsed(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	usedToken := identity.PasswordResetToken{UsedAt: &now}
	unusedToken := identity.PasswordResetToken{UsedAt: nil}

	assert.True(t, usedToken.IsUsed())
	assert.False(t, unusedToken.IsUsed())
}

func TestEmailVerificationToken_IsExpired(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	expiredToken := identity.EmailVerificationToken{ExpiresAt: now.Add(-1 * time.Hour)}
	validToken := identity.EmailVerificationToken{ExpiresAt: now.Add(1 * time.Hour)}

	assert.True(t, expiredToken.IsExpired())
	assert.False(t, validToken.IsExpired())
}

func TestEmailVerificationToken_IsUsed(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	usedToken := identity.EmailVerificationToken{UsedAt: &now}
	unusedToken := identity.EmailVerificationToken{UsedAt: nil}

	assert.True(t, usedToken.IsUsed())
	assert.False(t, unusedToken.IsUsed())
}
