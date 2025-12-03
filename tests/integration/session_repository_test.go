// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

// TestSessionRepository_Create tests creating a new session in the database.
func TestSessionRepository_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances once infrastructure layer is implemented
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create a user first
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create a session for the user
	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())

	// Act
	// err = sessionRepo.Save(ctx, sessionFixture)

	// Assert
	// require.NoError(t, err)
	// assert.NotEqual(t, uuid.Nil, sessionFixture.ID)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_GetByID tests retrieving a session by ID.
func TestSessionRepository_GetByID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create user and session
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())
	// err = sessionRepo.Save(ctx, sessionFixture)
	// require.NoError(t, err)

	// Act
	// found, err := sessionRepo.FindByID(ctx, sessionFixture.ID)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, sessionFixture.ID, found.ID)
	// assert.Equal(t, sessionFixture.UserID, found.UserID)
	// assert.Equal(t, sessionFixture.RefreshTokenHash, found.RefreshTokenHash)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_GetByUserID tests retrieving all sessions for a user.
func TestSessionRepository_GetByUserID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange - create user with multiple sessions
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create 3 sessions for the user
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())
	session3 := fixtures.UniqueSession(t, user.ID().UUID())

	// err = sessionRepo.Save(ctx, session1)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, session2)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, session3)
	// require.NoError(t, err)

	// Act
	// sessions, err := sessionRepo.FindByUserID(ctx, user.ID())

	// Assert
	// require.NoError(t, err)
	// assert.Len(t, sessions, 3)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_Revoke tests revoking a session.
func TestSessionRepository_Revoke(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	sessionFixture := fixtures.ValidSession(t, user.ID().UUID())
	// err = sessionRepo.Save(ctx, sessionFixture)
	// require.NoError(t, err)

	// Act
	// err = sessionRepo.Revoke(ctx, sessionFixture.ID)

	// Assert
	// require.NoError(t, err)

	// Verify session is revoked
	// found, err := sessionRepo.FindByID(ctx, sessionFixture.ID)
	// require.NoError(t, err)
	// assert.NotNil(t, found.RevokedAt)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_DeleteExpired tests deleting expired sessions.
func TestSessionRepository_DeleteExpired(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create expired and valid sessions
	expiredSession := fixtures.ExpiredSession(t, user.ID().UUID())
	validSession := fixtures.ValidSession(t, user.ID().UUID())

	// err = sessionRepo.Save(ctx, expiredSession)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, validSession)
	// require.NoError(t, err)

	// Act
	// count, err := sessionRepo.DeleteExpired(ctx)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, 1, count) // Only expired session deleted

	// Verify expired session is gone
	// _, err = sessionRepo.FindByID(ctx, expiredSession.ID)
	// require.Error(t, err) // Should not be found

	// Verify valid session still exists
	// _, err = sessionRepo.FindByID(ctx, validSession.ID)
	// require.NoError(t, err)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_CascadeDelete tests that sessions are deleted when user is deleted.
func TestSessionRepository_CascadeDelete(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create sessions for the user
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())

	// err = sessionRepo.Save(ctx, session1)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, session2)
	// require.NoError(t, err)

	// Act - delete the user
	// err = userRepo.Delete(ctx, user.ID())
	// require.NoError(t, err)

	// Assert - sessions should be cascade deleted
	// sessions, err := sessionRepo.FindByUserID(ctx, user.ID())
	// require.NoError(t, err)
	// assert.Empty(t, sessions)

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_RevokeAllForUser tests revoking all sessions for a user.
func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create multiple sessions
	session1 := fixtures.UniqueSession(t, user.ID().UUID())
	session2 := fixtures.UniqueSession(t, user.ID().UUID())
	session3 := fixtures.UniqueSession(t, user.ID().UUID())

	// err = sessionRepo.Save(ctx, session1)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, session2)
	// require.NoError(t, err)
	// err = sessionRepo.Save(ctx, session3)
	// require.NoError(t, err)

	// Act
	// count, err := sessionRepo.RevokeAllForUser(ctx, user.ID())

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, 3, count)

	// Verify all sessions are revoked
	// sessions, err := sessionRepo.FindByUserID(ctx, user.ID())
	// require.NoError(t, err)
	// for _, session := range sessions {
	//     assert.NotNil(t, session.RevokedAt)
	// }

	t.Skip("Skipping until SessionRepository implementation is available")
}

// TestSessionRepository_FindActiveByRefreshTokenHash tests finding an active session by refresh token hash.
func TestSessionRepository_FindActiveByRefreshTokenHash(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instances
	// userRepo := postgres.NewUserRepository(suite.DB)
	// sessionRepo := postgres.NewSessionRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := userRepo.Save(ctx, user)
	// require.NoError(t, err)

	// Create active session
	activeSession := fixtures.ValidSession(t, user.ID().UUID())
	// err = sessionRepo.Save(ctx, activeSession)
	// require.NoError(t, err)

	// Create revoked session with different hash
	revokedSession := fixtures.RevokedSession(t, user.ID().UUID())
	// err = sessionRepo.Save(ctx, revokedSession)
	// require.NoError(t, err)

	// Act - find by active session's token hash
	// found, err := sessionRepo.FindActiveByRefreshTokenHash(ctx, activeSession.RefreshTokenHash)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, activeSession.ID, found.ID)
	// assert.Nil(t, found.RevokedAt)

	t.Skip("Skipping until SessionRepository implementation is available")
}
