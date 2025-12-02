package moderation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewBan(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Repeated violations of community guidelines"

	t.Run("permanent ban", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)

		require.NoError(t, err)
		assert.NotNil(t, ban)
		assert.False(t, ban.ID().IsZero())
		assert.Equal(t, userID, ban.UserID())
		assert.Equal(t, bannedBy, ban.BannedBy())
		assert.Equal(t, reason, ban.Reason())
		assert.Nil(t, ban.ExpiresAt())
		assert.False(t, ban.CreatedAt().IsZero())
		assert.Nil(t, ban.RevokedAt())
		assert.Nil(t, ban.RevokedBy())
		assert.True(t, ban.IsActive())
		assert.True(t, ban.IsPermanent())
		assert.False(t, ban.IsExpired())

		// Should emit UserBanned event with isPermanent = true
		events := ban.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.user.banned", events[0].EventType())
	})

	t.Run("temporary ban", func(t *testing.T) {
		t.Parallel()

		duration := 7 * 24 * time.Hour // 7 days
		beforeBan := time.Now().UTC()

		ban, err := moderation.NewBan(userID, bannedBy, reason, &duration)

		require.NoError(t, err)
		assert.NotNil(t, ban)
		assert.NotNil(t, ban.ExpiresAt())
		assert.True(t, ban.ExpiresAt().After(beforeBan.Add(duration).Add(-1*time.Second)))
		assert.True(t, ban.IsActive())
		assert.False(t, ban.IsPermanent())
		assert.False(t, ban.IsExpired())

		// Should emit UserBanned event with isPermanent = false
		events := ban.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.user.banned", events[0].EventType())
	})

	t.Run("empty user id", func(t *testing.T) {
		t.Parallel()

		var emptyUserID identity.UserID
		_, err := moderation.NewBan(emptyUserID, bannedBy, reason, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user id is required")
	})

	t.Run("empty banned by id", func(t *testing.T) {
		t.Parallel()

		var emptyBannedBy identity.UserID
		_, err := moderation.NewBan(userID, emptyBannedBy, reason, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "banned by user id is required")
	})

	t.Run("empty reason", func(t *testing.T) {
		t.Parallel()

		_, err := moderation.NewBan(userID, bannedBy, "", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReasonRequired)
	})

	t.Run("reason too long", func(t *testing.T) {
		t.Parallel()

		longReason := strings.Repeat("a", 501)
		_, err := moderation.NewBan(userID, bannedBy, longReason, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrReasonTooLong)
	})

	t.Run("reason at max length", func(t *testing.T) {
		t.Parallel()

		maxReason := strings.Repeat("a", 500)
		ban, err := moderation.NewBan(userID, bannedBy, maxReason, nil)

		require.NoError(t, err)
		assert.NotNil(t, ban)
		assert.Equal(t, maxReason, ban.Reason())
	})
}

func TestReconstructBan(t *testing.T) {
	t.Parallel()

	id := moderation.NewBanID()
	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test reason"
	expiresAt := time.Now().Add(24 * time.Hour).UTC()
	createdAt := time.Now().Add(-1 * time.Hour).UTC()
	revokedAt := time.Now().UTC()
	revokedBy := identity.NewUserID()

	ban := moderation.ReconstructBan(
		id,
		userID,
		bannedBy,
		reason,
		&expiresAt,
		createdAt,
		&revokedAt,
		&revokedBy,
	)

	assert.NotNil(t, ban)
	assert.Equal(t, id, ban.ID())
	assert.Equal(t, userID, ban.UserID())
	assert.Equal(t, bannedBy, ban.BannedBy())
	assert.Equal(t, reason, ban.Reason())
	assert.Equal(t, &expiresAt, ban.ExpiresAt())
	assert.Equal(t, createdAt, ban.CreatedAt())
	assert.Equal(t, &revokedAt, ban.RevokedAt())
	assert.Equal(t, &revokedBy, ban.RevokedBy())

	// Reconstruction should not emit events
	assert.Empty(t, ban.Events())
}

func TestBan_IsActive(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test"

	t.Run("permanent ban is active", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		assert.True(t, ban.IsActive())
	})

	t.Run("temporary ban not expired is active", func(t *testing.T) {
		t.Parallel()

		duration := 24 * time.Hour
		ban, err := moderation.NewBan(userID, bannedBy, reason, &duration)
		require.NoError(t, err)

		assert.True(t, ban.IsActive())
	})

	t.Run("expired ban is not active", func(t *testing.T) {
		t.Parallel()

		// Create ban with past expiry
		pastExpiry := time.Now().Add(-1 * time.Hour).UTC()
		ban := moderation.ReconstructBan(
			moderation.NewBanID(),
			userID,
			bannedBy,
			reason,
			&pastExpiry,
			time.Now().Add(-2*time.Hour).UTC(),
			nil,
			nil,
		)

		assert.False(t, ban.IsActive())
	})

	t.Run("revoked ban is not active", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		revokedBy := identity.NewUserID()
		err = ban.Revoke(revokedBy)
		require.NoError(t, err)

		assert.False(t, ban.IsActive())
	})
}

func TestBan_IsPermanent(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test"

	t.Run("ban with no expiry is permanent", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		assert.True(t, ban.IsPermanent())
	})

	t.Run("ban with expiry is not permanent", func(t *testing.T) {
		t.Parallel()

		duration := 24 * time.Hour
		ban, err := moderation.NewBan(userID, bannedBy, reason, &duration)
		require.NoError(t, err)

		assert.False(t, ban.IsPermanent())
	})
}

func TestBan_IsExpired(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test"

	t.Run("permanent ban is never expired", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		assert.False(t, ban.IsExpired())
	})

	t.Run("temporary ban not past expiry is not expired", func(t *testing.T) {
		t.Parallel()

		duration := 24 * time.Hour
		ban, err := moderation.NewBan(userID, bannedBy, reason, &duration)
		require.NoError(t, err)

		assert.False(t, ban.IsExpired())
	})

	t.Run("temporary ban past expiry is expired", func(t *testing.T) {
		t.Parallel()

		// Create ban with past expiry
		pastExpiry := time.Now().Add(-1 * time.Hour).UTC()
		ban := moderation.ReconstructBan(
			moderation.NewBanID(),
			userID,
			bannedBy,
			reason,
			&pastExpiry,
			time.Now().Add(-2*time.Hour).UTC(),
			nil,
			nil,
		)

		assert.True(t, ban.IsExpired())
	})

	t.Run("revoked ban is not considered expired", func(t *testing.T) {
		t.Parallel()

		// Create ban with future expiry but revoked
		futureExpiry := time.Now().Add(24 * time.Hour).UTC()
		now := time.Now().UTC()
		revokedBy := identity.NewUserID()

		ban := moderation.ReconstructBan(
			moderation.NewBanID(),
			userID,
			bannedBy,
			reason,
			&futureExpiry,
			time.Now().Add(-1*time.Hour).UTC(),
			&now,
			&revokedBy,
		)

		assert.False(t, ban.IsExpired())
	})
}

func TestBan_Revoke(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test"

	t.Run("revoke permanent ban", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)
		ban.ClearEvents()

		revokedBy := identity.NewUserID()
		beforeRevoke := time.Now().UTC()

		err = ban.Revoke(revokedBy)

		require.NoError(t, err)
		assert.NotNil(t, ban.RevokedAt())
		assert.True(t, ban.RevokedAt().After(beforeRevoke) || ban.RevokedAt().Equal(beforeRevoke))
		assert.NotNil(t, ban.RevokedBy())
		assert.Equal(t, revokedBy, *ban.RevokedBy())
		assert.False(t, ban.IsActive())

		// Should emit BanRevoked event
		events := ban.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "moderation.ban.revoked", events[0].EventType())
	})

	t.Run("revoke temporary ban", func(t *testing.T) {
		t.Parallel()

		duration := 24 * time.Hour
		ban, err := moderation.NewBan(userID, bannedBy, reason, &duration)
		require.NoError(t, err)
		ban.ClearEvents()

		revokedBy := identity.NewUserID()

		err = ban.Revoke(revokedBy)

		require.NoError(t, err)
		assert.NotNil(t, ban.RevokedAt())
		assert.NotNil(t, ban.RevokedBy())
		assert.False(t, ban.IsActive())
	})

	t.Run("empty revoked by id", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		var emptyRevokedBy identity.UserID
		err = ban.Revoke(emptyRevokedBy)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoked by user id is required")
		assert.Nil(t, ban.RevokedAt())
	})

	t.Run("cannot revoke already revoked ban", func(t *testing.T) {
		t.Parallel()

		ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
		require.NoError(t, err)

		revokedBy := identity.NewUserID()
		require.NoError(t, ban.Revoke(revokedBy))
		firstRevokedAt := ban.RevokedAt()
		ban.ClearEvents()

		err = ban.Revoke(revokedBy)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrBanAlreadyRevoked)
		assert.Equal(t, firstRevokedAt, ban.RevokedAt()) // Should not change
	})

	t.Run("cannot revoke expired ban", func(t *testing.T) {
		t.Parallel()

		// Create ban with past expiry
		pastExpiry := time.Now().Add(-1 * time.Hour).UTC()
		ban := moderation.ReconstructBan(
			moderation.NewBanID(),
			userID,
			bannedBy,
			reason,
			&pastExpiry,
			time.Now().Add(-2*time.Hour).UTC(),
			nil,
			nil,
		)

		revokedBy := identity.NewUserID()
		err := ban.Revoke(revokedBy)

		require.Error(t, err)
		assert.ErrorIs(t, err, moderation.ErrBanExpired)
		assert.Nil(t, ban.RevokedAt())
	})
}

func TestBan_ClearEvents(t *testing.T) {
	t.Parallel()

	userID := identity.NewUserID()
	bannedBy := identity.NewUserID()
	reason := "Test"

	ban, err := moderation.NewBan(userID, bannedBy, reason, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, ban.Events())

	ban.ClearEvents()

	assert.Empty(t, ban.Events())
}
