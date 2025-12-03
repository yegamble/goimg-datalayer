// +build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

// TestUserRepository_Create tests creating a new user in the database.
func TestUserRepository_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance once infrastructure layer is implemented
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)

	// Act
	// err := repo.Save(ctx, user)

	// Assert
	// require.NoError(t, err)
	// assert.NotEqual(t, uuid.Nil, user.ID().UUID())

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_GetByID tests retrieving a user by ID.
func TestUserRepository_GetByID(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create a user first
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := repo.Save(ctx, user)
	// require.NoError(t, err)

	// Act - retrieve the user
	// found, err := repo.FindByID(ctx, user.ID())

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, user.ID(), found.ID())
	// assert.Equal(t, user.Email().String(), found.Email().String())
	// assert.Equal(t, user.Username().String(), found.Username().String())

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_GetByEmail tests retrieving a user by email.
func TestUserRepository_GetByEmail(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := repo.Save(ctx, user)
	// require.NoError(t, err)

	// Act
	// email, _ := identity.NewEmail(userFixture.Email)
	// found, err := repo.FindByEmail(ctx, email)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, user.ID(), found.ID())
	// assert.Equal(t, user.Email(), found.Email())

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_GetByUsername tests retrieving a user by username.
func TestUserRepository_GetByUsername(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := repo.Save(ctx, user)
	// require.NoError(t, err)

	// Act
	// username, _ := identity.NewUsername(userFixture.Username)
	// found, err := repo.FindByUsername(ctx, username)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, user.ID(), found.ID())
	// assert.Equal(t, user.Username(), found.Username())

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_Update tests updating an existing user.
func TestUserRepository_Update(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create a user
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := repo.Save(ctx, user)
	// require.NoError(t, err)

	// Modify the user
	// newEmail, _ := identity.NewEmail("newemail@example.com")
	// err = user.ChangeEmail(newEmail)
	// require.NoError(t, err)

	// Act - update the user
	// err = repo.Save(ctx, user)

	// Assert
	// require.NoError(t, err)

	// Verify the update persisted
	// found, err := repo.FindByID(ctx, user.ID())
	// require.NoError(t, err)
	// assert.Equal(t, newEmail, found.Email())

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_Delete tests soft deleting a user.
func TestUserRepository_Delete(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	user := userFixture.ToEntity(t)
	// err := repo.Save(ctx, user)
	// require.NoError(t, err)

	// Act
	// err = repo.Delete(ctx, user.ID())

	// Assert
	// require.NoError(t, err)

	// Verify user is soft deleted (not found)
	// _, err = repo.FindByID(ctx, user.ID())
	// require.ErrorIs(t, err, identity.ErrUserNotFound)

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_DuplicateEmail tests that duplicate emails are rejected.
func TestUserRepository_DuplicateEmail(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create first user
	user1Fixture := fixtures.ValidUser(t)
	user1 := user1Fixture.ToEntity(t)
	// err := repo.Save(ctx, user1)
	// require.NoError(t, err)

	// Create second user with same email
	user2Fixture := fixtures.ValidUser(t).WithEmail(user1Fixture.Email).WithUsername("different")
	user2 := user2Fixture.ToEntity(t)

	// Act
	// err = repo.Save(ctx, user2)

	// Assert - should fail due to unique constraint
	// require.Error(t, err)
	// assert.Contains(t, err.Error(), "duplicate") // or check for specific postgres error

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_DuplicateUsername tests that duplicate usernames are rejected.
func TestUserRepository_DuplicateUsername(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	// Arrange - create first user
	user1Fixture := fixtures.ValidUser(t)
	user1 := user1Fixture.ToEntity(t)
	// err := repo.Save(ctx, user1)
	// require.NoError(t, err)

	// Create second user with same username
	user2Fixture := fixtures.ValidUser(t).WithEmail("different@example.com").WithUsername(user1Fixture.Username)
	user2 := user2Fixture.ToEntity(t)

	// Act
	// err = repo.Save(ctx, user2)

	// Assert - should fail due to unique constraint
	// require.Error(t, err)
	// assert.Contains(t, err.Error(), "duplicate")

	t.Skip("Skipping until UserRepository implementation is available")
}

// TestUserRepository_NotFound tests that ErrUserNotFound is returned for non-existent users.
func TestUserRepository_NotFound(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create repository instance
	// repo := postgres.NewUserRepository(suite.DB)

	t.Run("FindByID returns ErrUserNotFound", func(t *testing.T) {
		nonExistentID := identity.NewUserID()

		// Act
		// _, err := repo.FindByID(ctx, nonExistentID)

		// Assert
		// require.ErrorIs(t, err, identity.ErrUserNotFound)

		t.Skip("Skipping until UserRepository implementation is available")
	})

	t.Run("FindByEmail returns ErrUserNotFound", func(t *testing.T) {
		email, _ := identity.NewEmail("nonexistent@example.com")

		// Act
		// _, err := repo.FindByEmail(ctx, email)

		// Assert
		// require.ErrorIs(t, err, identity.ErrUserNotFound)

		t.Skip("Skipping until UserRepository implementation is available")
	})

	t.Run("FindByUsername returns ErrUserNotFound", func(t *testing.T) {
		username, _ := identity.NewUsername("nonexistent")

		// Act
		// _, err := repo.FindByUsername(ctx, username)

		// Assert
		// require.ErrorIs(t, err, identity.ErrUserNotFound)

		t.Skip("Skipping until UserRepository implementation is available")
	})
}
