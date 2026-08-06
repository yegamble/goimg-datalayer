//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

// TestUserRepository_Create tests creating a new user in the database.
func TestUserRepository_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)

	// Act
	err := repo.Save(ctx, user)

	// Assert
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID().UUID())
}

// TestUserRepository_GetByID tests retrieving a user by ID.
func TestUserRepository_GetByID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create a user first
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Act - retrieve the user
	found, err := repo.FindByID(ctx, user.ID())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.ID())
	assert.Equal(t, user.Email().String(), found.Email().String())
	assert.Equal(t, user.Username().String(), found.Username().String())
}

// TestUserRepository_GetByEmail tests retrieving a user by email.
func TestUserRepository_GetByEmail(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Act
	email, _ := identity.NewEmail(userFixture.Email)
	found, err := repo.FindByEmail(ctx, email)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.ID())
	assert.Equal(t, user.Email(), found.Email())
}

// TestUserRepository_GetByUsername tests retrieving a user by username.
func TestUserRepository_GetByUsername(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Act
	username, _ := identity.NewUsername(userFixture.Username)
	found, err := repo.FindByUsername(ctx, username)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.ID())
	assert.Equal(t, user.Username(), found.Username())
}

// TestUserRepository_Update tests updating an existing user.
func TestUserRepository_Update(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create a user
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Modify the user
	newEmail, _ := identity.NewEmail("newemail@example.com")
	err = user.ChangeEmail(newEmail)
	require.NoError(t, err)

	// Act - update the user
	err = repo.Save(ctx, user)

	// Assert
	require.NoError(t, err)

	// Verify the update persisted
	found, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, newEmail, found.Email())
}

// TestUserRepository_Delete tests soft deleting a user.
func TestUserRepository_Delete(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Act
	err = repo.Delete(ctx, user.ID())

	// Assert
	require.NoError(t, err)

	// Verify user is soft deleted (not found)
	_, err = repo.FindByID(ctx, user.ID())
	require.ErrorIs(t, err, identity.ErrUserNotFound)
}

// TestUserRepository_DuplicateEmail tests that duplicate emails are rejected.
func TestUserRepository_DuplicateEmail(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create first user
	user1Fixture := fixtures.ValidUser(t)
	user1 := user1Fixture.ToEntity(t)
	err := repo.Save(ctx, user1)
	require.NoError(t, err)

	// Create second user with same email
	user2Fixture := fixtures.ValidUser(t).WithEmail(user1Fixture.Email).WithUsername("different")
	user2 := user2Fixture.ToEntity(t)

	// Act
	err = repo.Save(ctx, user2)

	// Assert - should fail due to unique constraint
	require.Error(t, err)
	assert.ErrorIs(t, err, identity.ErrEmailExists)
}

// TestUserRepository_DuplicateUsername tests that duplicate usernames are rejected.
func TestUserRepository_DuplicateUsername(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create first user
	user1Fixture := fixtures.ValidUser(t)
	user1 := user1Fixture.ToEntity(t)
	err := repo.Save(ctx, user1)
	require.NoError(t, err)

	// Create second user with same username
	user2Fixture := fixtures.ValidUser(t).WithEmail("different@example.com").WithUsername(user1Fixture.Username)
	user2 := user2Fixture.ToEntity(t)

	// Act
	err = repo.Save(ctx, user2)

	// Assert - should fail due to unique constraint
	require.Error(t, err)
	assert.ErrorIs(t, err, identity.ErrUsernameExists)
}

// TestUserRepository_NotFound tests that ErrUserNotFound is returned for non-existent users.
func TestUserRepository_NotFound(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	t.Run("FindByID returns ErrUserNotFound", func(t *testing.T) {
		nonExistentID := identity.NewUserID()

		// Act
		_, err := repo.FindByID(ctx, nonExistentID)

		// Assert
		require.ErrorIs(t, err, identity.ErrUserNotFound)
	})

	t.Run("FindByEmail returns ErrUserNotFound", func(t *testing.T) {
		email, _ := identity.NewEmail("nonexistent@example.com")

		// Act
		_, err := repo.FindByEmail(ctx, email)

		// Assert
		require.ErrorIs(t, err, identity.ErrUserNotFound)
	})

	t.Run("FindByUsername returns ErrUserNotFound", func(t *testing.T) {
		username, _ := identity.NewUsername("nonexistent")

		// Act
		_, err := repo.FindByUsername(ctx, username)

		// Assert
		require.ErrorIs(t, err, identity.ErrUserNotFound)
	})
}

// TestUserRepository_FindExpiredGuests tests retrieving expired guest users.
func TestUserRepository_FindExpiredGuests(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// 1. Create an expired guest user
	expiredID := identity.NewUserID()
	expiredEmail, _ := identity.NewEmail(fmt.Sprintf("guest_%s@goimg.local", expiredID.String()))
	expiredUsername, _ := identity.NewUsername(fmt.Sprintf("guest_%s", expiredID.String()[:8]))
	expiredHash, err := identity.NewPasswordHash("dummy-password-123!")
	require.NoError(t, err)

	now := time.Now().UTC()
	// Ensure expired time is definitely in the past relative to the query time (25 hours ago)
	expiredTime := now.Add(-25 * time.Hour)
	ip := "127.0.0.1"

	expiredGuest := identity.ReconstructUser(
		expiredID,
		expiredEmail,
		expiredUsername,
		expiredHash,
		identity.RoleUser,
		identity.StatusActive,
		expiredUsername.String(),
		"",
		0,
		now.Add(-48*time.Hour), // Created 2 days ago
		now.Add(-48*time.Hour),
		identity.UserTypeGuest,
		&ip,
		&expiredTime,
		true,
		nil,
	)

	err = repo.Save(ctx, expiredGuest)
	require.NoError(t, err)

	// Verify persistence immediately to ensure data is correct before querying
	saved, err := repo.FindByID(ctx, expiredID)
	require.NoError(t, err)
	assert.Equal(t, identity.UserTypeGuest, saved.UserType())
	require.NotNil(t, saved.ExpiresAt())
	// Allow for small time difference due to round trip and DB precision
	assert.WithinDuration(t, expiredTime, *saved.ExpiresAt(), time.Second)

	// 2. Create an active guest user
	activeGuest, err := identity.NewGuestUser("127.0.0.2")
	require.NoError(t, err)
	err = repo.Save(ctx, activeGuest)
	require.NoError(t, err)

	// 3. Create a registered user
	registeredFixture := fixtures.ValidUser(t)
	registeredUser := registeredFixture.ToEntity(t)
	err = repo.Save(ctx, registeredUser)
	require.NoError(t, err)

	// Act
	expiredGuests, err := repo.FindExpiredGuests(ctx, now, 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, expiredGuests, 1)
	assert.Equal(t, expiredGuest.ID(), expiredGuests[0].ID())
}

// TestUserRepository_NextID tests generating a new UserID.
func TestUserRepository_NextID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)

	// Create repository instance
	repo := postgres.NewUserRepository(suite.DB)

	// Act
	id := repo.NextID()

	// Assert
	assert.NotEmpty(t, id.String())
	assert.NotEqual(t, uuid.Nil, id.UUID())
}
