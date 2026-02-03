//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

func toPostgresSession(t *testing.T, fixture *fixtures.SessionFixture) *postgres.Session {
	t.Helper()
	userID, err := identity.ParseUserID(fixture.UserID.String())
	require.NoError(t, err)

	return &postgres.Session{
		ID:               fixture.ID,
		UserID:           userID,
		RefreshTokenHash: fixture.RefreshTokenHash,
		IPAddress:        fixture.IPAddress,
		UserAgent:        fixture.UserAgent,
		ExpiresAt:        fixture.ExpiresAt,
		CreatedAt:        fixture.CreatedAt,
		RevokedAt:        fixture.RevokedAt,
	}
}

// TestSessionRepository_Create tests creating a new session in the database.
func TestSessionRepository_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create a user first
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create a session for the user
	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())

	// Act
	session := toPostgresSession(t, sessionFixture)
	err = sessionRepo.Create(ctx, session)

	// Assert
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID)
}

// TestSessionRepository_GetByID tests retrieving a session by ID.
func TestSessionRepository_GetByID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create user and session
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())
	session := toPostgresSession(t, sessionFixture)
	err = sessionRepo.Create(ctx, session)
	require.NoError(t, err)

	// Act
	found, err := sessionRepo.GetByID(ctx, sessionFixture.ID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, sessionFixture.ID, found.ID)
	assert.Equal(t, sessionFixture.UserID.String(), found.UserID.String())
	assert.Equal(t, sessionFixture.RefreshTokenHash, found.RefreshTokenHash)
	assert.Equal(t, sessionFixture.IPAddress, found.IPAddress)
	assert.Equal(t, sessionFixture.UserAgent, found.UserAgent)
}

// TestSessionRepository_GetByUserID tests retrieving all sessions for a user.
func TestSessionRepository_GetByUserID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create user with multiple sessions
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create 3 sessions for the user
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())
	session3 := fixtures.UniqueSession(t, user.ID().UUID())

	err = sessionRepo.Create(ctx, toPostgresSession(t, session1))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, session2))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, session3))
	require.NoError(t, err)

	// Act
	sessions, err := sessionRepo.GetByUserID(ctx, user.ID())

	// Assert
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

// TestSessionRepository_Revoke tests revoking a session.
func TestSessionRepository_Revoke(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())
	err = sessionRepo.Create(ctx, toPostgresSession(t, sessionFixture))
	require.NoError(t, err)

	// Act
	err = sessionRepo.Revoke(ctx, sessionFixture.ID)

	// Assert
	require.NoError(t, err)

	// Verify session is revoked
	found, err := sessionRepo.GetByID(ctx, sessionFixture.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.RevokedAt)
}

// TestSessionRepository_DeleteExpired tests deleting expired sessions.
func TestSessionRepository_DeleteExpired(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create expired and valid sessions
	expiredSession := fixtures.ExpiredSession(t, user.ID().UUID())
	validSession := fixtures.ValidSession(t, user.ID().UUID())

	err = sessionRepo.Create(ctx, toPostgresSession(t, expiredSession))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, validSession))
	require.NoError(t, err)

	// Act
	count, err := sessionRepo.DeleteExpired(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), count) // Only expired session deleted

	// Verify expired session is gone
	_, err = sessionRepo.GetByID(ctx, expiredSession.ID)
	require.Error(t, err) // Should not be found

	// Verify valid session still exists
	_, err = sessionRepo.GetByID(ctx, validSession.ID)
	require.NoError(t, err)
}

// TestSessionRepository_CascadeDelete tests that sessions are deleted when user is deleted.
func TestSessionRepository_CascadeDelete(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create sessions for the user
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())

	err = sessionRepo.Create(ctx, toPostgresSession(t, session1))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, session2))
	require.NoError(t, err)

	// Act - delete the user (Hard Delete to trigger DB CASCADE)
	// Note: userRepo.Delete() performs a soft delete which doesn't trigger foreign key cascades.
	// For this test, we want to verify the database constraint configuration.
	_, err = suite.DB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID().String())
	require.NoError(t, err)

	// Assert - sessions should be cascade deleted
	sessions, err := sessionRepo.GetByUserID(ctx, user.ID())
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

// TestSessionRepository_RevokeAllForUser tests revoking all sessions for a user.
func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create multiple sessions
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())
	session3 := fixtures.UniqueSession(t, user.ID().UUID())

	err = sessionRepo.Create(ctx, toPostgresSession(t, session1))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, session2))
	require.NoError(t, err)
	err = sessionRepo.Create(ctx, toPostgresSession(t, session3))
	require.NoError(t, err)

	// Act
	count, err := sessionRepo.RevokeAllForUser(ctx, user.ID())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify all sessions are revoked
	// GetByUserID filters out revoked sessions, so it should return empty
	sessions, err := sessionRepo.GetByUserID(ctx, user.ID())
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Verify individual sessions are marked as revoked
	s1, err := sessionRepo.GetByID(ctx, session1.ID)
	require.NoError(t, err)
	assert.NotNil(t, s1.RevokedAt)
}

// TestSessionRepository_GetActiveByRefreshTokenHash tests finding an active session by refresh token hash.
func TestSessionRepository_GetActiveByRefreshTokenHash(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	userRepo := postgres.NewUserRepository(suite.DB)
	sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := userRepo.Save(ctx, user)
	require.NoError(t, err)

	// Create active session
	activeSession := fixtures.ValidSession(t, user.ID().UUID())
	err = sessionRepo.Create(ctx, toPostgresSession(t, activeSession))
	require.NoError(t, err)

	// Create revoked session with different hash
	revokedSession := fixtures.RevokedSession(t, user.ID().UUID())
	err = sessionRepo.Create(ctx, toPostgresSession(t, revokedSession))
	require.NoError(t, err)

	// Act - find by active session's token hash
	found, err := sessionRepo.GetActiveByRefreshTokenHash(ctx, activeSession.RefreshTokenHash)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, activeSession.ID, found.ID)
	assert.Nil(t, found.RevokedAt)
}
