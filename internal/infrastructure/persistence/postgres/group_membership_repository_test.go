package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

// createTestMembership inserts a test membership into the database.
func createTestMembership(
	t *testing.T,
	db *sqlx.DB,
	membershipID community.MembershipID,
	groupID community.GroupID,
	userID identity.UserID,
	role string,
	status string,
) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO group_memberships (
			id, group_id, user_id, role, status, joined_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		membershipID.String(),
		groupID.String(),
		userID.String(),
		role,
		status,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func TestGroupMembershipRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	memberID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test users and group
	createTestUser(t, db, ownerID)
	createTestUser(t, db, memberID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	t.Run("saves new membership successfully", func(t *testing.T) {
		membership, err := community.NewGroupMembership(groupID, memberID, community.GroupRoleMember)
		require.NoError(t, err)

		err = repo.Save(ctx, membership)
		require.NoError(t, err)

		// Verify membership exists
		found, err := repo.FindByID(ctx, membership.ID())
		require.NoError(t, err)
		assert.Equal(t, membership.ID(), found.ID())
		assert.Equal(t, groupID, found.GroupID())
		assert.Equal(t, memberID, found.UserID())
		assert.Equal(t, community.GroupRoleMember, found.Role())
		assert.Equal(t, community.MemberStatusActive, found.Status())
	})

	t.Run("updates existing membership successfully", func(t *testing.T) {
		member2ID := identity.NewUserID()
		createTestUser(t, db, member2ID)

		membership, err := community.NewGroupMembership(groupID, member2ID, community.GroupRoleMember)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, membership)
		require.NoError(t, err)

		// Promote to admin
		err = membership.PromoteToAdmin()
		require.NoError(t, err)

		// Save again
		err = repo.Save(ctx, membership)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, membership.ID())
		require.NoError(t, err)
		assert.Equal(t, community.GroupRoleAdmin, found.Role())
	})

	t.Run("fails when group does not exist", func(t *testing.T) {
		invalidGroupID := community.NewGroupID()
		memberID := identity.NewUserID()
		createTestUser(t, db, memberID)

		membership, err := community.NewGroupMembership(invalidGroupID, memberID, community.GroupRoleMember)
		require.NoError(t, err)

		err = repo.Save(ctx, membership)
		assert.Error(t, err) // Foreign key constraint violation
	})

	t.Run("fails when user does not exist", func(t *testing.T) {
		invalidUserID := identity.NewUserID()

		membership, err := community.NewGroupMembership(groupID, invalidUserID, community.GroupRoleMember)
		require.NoError(t, err)

		err = repo.Save(ctx, membership)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestGroupMembershipRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	memberID := identity.NewUserID()
	groupID := community.NewGroupID()
	membershipID := community.NewMembershipID()

	// Setup: Create test users, group, and membership
	createTestUser(t, db, ownerID)
	createTestUser(t, db, memberID)
	createTestGroup(t, db, groupID, ownerID, "test-group")
	createTestMembership(t, db, membershipID, groupID, memberID, "member", "active")

	t.Run("finds existing membership by ID", func(t *testing.T) {
		membership, err := repo.FindByID(ctx, membershipID)
		require.NoError(t, err)
		assert.Equal(t, membershipID, membership.ID())
		assert.Equal(t, groupID, membership.GroupID())
		assert.Equal(t, memberID, membership.UserID())
	})

	t.Run("returns ErrMembershipNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewMembershipID()
		_, err := repo.FindByID(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrMembershipNotFound)
	})
}

func TestGroupMembershipRepository_FindByGroupAndUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	memberID := identity.NewUserID()
	groupID := community.NewGroupID()
	membershipID := community.NewMembershipID()

	// Setup: Create test users, group, and membership
	createTestUser(t, db, ownerID)
	createTestUser(t, db, memberID)
	createTestGroup(t, db, groupID, ownerID, "test-group")
	createTestMembership(t, db, membershipID, groupID, memberID, "member", "active")

	t.Run("finds membership by group and user", func(t *testing.T) {
		membership, err := repo.FindByGroupAndUser(ctx, groupID, memberID)
		require.NoError(t, err)
		assert.Equal(t, groupID, membership.GroupID())
		assert.Equal(t, memberID, membership.UserID())
	})

	t.Run("returns ErrMembershipNotFound when not found", func(t *testing.T) {
		nonMemberID := identity.NewUserID()
		createTestUser(t, db, nonMemberID)

		_, err := repo.FindByGroupAndUser(ctx, groupID, nonMemberID)
		require.ErrorIs(t, err, community.ErrMembershipNotFound)
	})
}

func TestGroupMembershipRepository_FindByGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create multiple memberships
	member1ID := identity.NewUserID()
	member2ID := identity.NewUserID()
	member3ID := identity.NewUserID()

	createTestUser(t, db, member1ID)
	createTestUser(t, db, member2ID)
	createTestUser(t, db, member3ID)

	membership1ID := community.NewMembershipID()
	membership2ID := community.NewMembershipID()
	membership3ID := community.NewMembershipID()

	createTestMembership(t, db, membership1ID, groupID, member1ID, "member", "active")
	createTestMembership(t, db, membership2ID, groupID, member2ID, "admin", "active")
	createTestMembership(t, db, membership3ID, groupID, member3ID, "member", "invited")

	t.Run("finds all memberships for group", func(t *testing.T) {
		filter := community.MemberFilter{}
		pagination := shared.NewPagination(0, 10)

		memberships, total, err := repo.FindByGroup(ctx, groupID, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, memberships, 3)
	})

	t.Run("filters by role", func(t *testing.T) {
		role := community.GroupRoleAdmin
		filter := community.MemberFilter{Role: &role}
		pagination := shared.NewPagination(0, 10)

		memberships, total, err := repo.FindByGroup(ctx, groupID, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, memberships, 1)
		assert.Equal(t, community.GroupRoleAdmin, memberships[0].Role())
	})

	t.Run("filters by status", func(t *testing.T) {
		status := community.MemberStatusActive
		filter := community.MemberFilter{Status: &status}
		pagination := shared.NewPagination(0, 10)

		memberships, total, err := repo.FindByGroup(ctx, groupID, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, memberships, 2)

		for _, membership := range memberships {
			assert.Equal(t, community.MemberStatusActive, membership.Status())
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		filter := community.MemberFilter{}
		pagination := shared.NewPagination(0, 2)

		memberships, total, err := repo.FindByGroup(ctx, groupID, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, memberships, 2)
	})
}

func TestGroupMembershipRepository_FindByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	userID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, ownerID)
	createTestUser(t, db, userID)

	// Create multiple groups and memberships for the user
	group1ID := community.NewGroupID()
	group2ID := community.NewGroupID()

	createTestGroup(t, db, group1ID, ownerID, "test-group-1")
	createTestGroup(t, db, group2ID, ownerID, "test-group-2")

	membership1ID := community.NewMembershipID()
	membership2ID := community.NewMembershipID()

	createTestMembership(t, db, membership1ID, group1ID, userID, "member", "active")
	createTestMembership(t, db, membership2ID, group2ID, userID, "admin", "active")

	t.Run("finds all memberships for user", func(t *testing.T) {
		pagination := shared.NewPagination(0, 10)

		memberships, total, err := repo.FindByUser(ctx, userID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, memberships, 2)

		for _, membership := range memberships {
			assert.Equal(t, userID, membership.UserID())
		}
	})

	t.Run("returns empty results for user with no memberships", func(t *testing.T) {
		noMembershipUserID := identity.NewUserID()
		createTestUser(t, db, noMembershipUserID)

		pagination := shared.NewPagination(0, 10)

		memberships, total, err := repo.FindByUser(ctx, noMembershipUserID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, memberships)
	})

	t.Run("respects pagination", func(t *testing.T) {
		pagination := shared.NewPagination(0, 1)

		memberships, total, err := repo.FindByUser(ctx, userID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, memberships, 1)
	})
}

func TestGroupMembershipRepository_FindActiveByGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create memberships with different statuses
	activeMember1ID := identity.NewUserID()
	activeMember2ID := identity.NewUserID()
	invitedMemberID := identity.NewUserID()

	createTestUser(t, db, activeMember1ID)
	createTestUser(t, db, activeMember2ID)
	createTestUser(t, db, invitedMemberID)

	membership1ID := community.NewMembershipID()
	membership2ID := community.NewMembershipID()
	membership3ID := community.NewMembershipID()

	createTestMembership(t, db, membership1ID, groupID, activeMember1ID, "member", "active")
	createTestMembership(t, db, membership2ID, groupID, activeMember2ID, "member", "active")
	createTestMembership(t, db, membership3ID, groupID, invitedMemberID, "member", "invited")

	t.Run("returns only active memberships", func(t *testing.T) {
		memberships, err := repo.FindActiveByGroup(ctx, groupID)
		require.NoError(t, err)
		assert.Len(t, memberships, 2)

		for _, membership := range memberships {
			assert.Equal(t, community.MemberStatusActive, membership.Status())
		}
	})
}

func TestGroupMembershipRepository_FindPendingByGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create memberships with different statuses
	activeMemberID := identity.NewUserID()
	invitedMemberID := identity.NewUserID()
	requestedMemberID := identity.NewUserID()

	createTestUser(t, db, activeMemberID)
	createTestUser(t, db, invitedMemberID)
	createTestUser(t, db, requestedMemberID)

	membership1ID := community.NewMembershipID()
	membership2ID := community.NewMembershipID()
	membership3ID := community.NewMembershipID()

	createTestMembership(t, db, membership1ID, groupID, activeMemberID, "member", "active")
	createTestMembership(t, db, membership2ID, groupID, invitedMemberID, "member", "invited")
	createTestMembership(t, db, membership3ID, groupID, requestedMemberID, "member", "requested")

	t.Run("returns only pending memberships", func(t *testing.T) {
		memberships, err := repo.FindPendingByGroup(ctx, groupID)
		require.NoError(t, err)
		assert.Len(t, memberships, 2)

		for _, membership := range memberships {
			assert.True(t, membership.Status().IsPending())
		}
	})
}

func TestGroupMembershipRepository_CountByGroupAndStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create memberships with different statuses
	member1ID := identity.NewUserID()
	member2ID := identity.NewUserID()
	member3ID := identity.NewUserID()

	createTestUser(t, db, member1ID)
	createTestUser(t, db, member2ID)
	createTestUser(t, db, member3ID)

	membership1ID := community.NewMembershipID()
	membership2ID := community.NewMembershipID()
	membership3ID := community.NewMembershipID()

	createTestMembership(t, db, membership1ID, groupID, member1ID, "member", "active")
	createTestMembership(t, db, membership2ID, groupID, member2ID, "member", "active")
	createTestMembership(t, db, membership3ID, groupID, member3ID, "member", "invited")

	t.Run("counts active memberships", func(t *testing.T) {
		count, err := repo.CountByGroupAndStatus(ctx, groupID, community.MemberStatusActive)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("counts invited memberships", func(t *testing.T) {
		count, err := repo.CountByGroupAndStatus(ctx, groupID, community.MemberStatusInvited)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("returns zero for status with no memberships", func(t *testing.T) {
		count, err := repo.CountByGroupAndStatus(ctx, groupID, community.MemberStatusBanned)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestGroupMembershipRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	memberID := identity.NewUserID()
	groupID := community.NewGroupID()
	membershipID := community.NewMembershipID()

	// Setup: Create test users, group, and membership
	createTestUser(t, db, ownerID)
	createTestUser(t, db, memberID)
	createTestGroup(t, db, groupID, ownerID, "test-group")
	createTestMembership(t, db, membershipID, groupID, memberID, "member", "active")

	t.Run("deletes membership successfully", func(t *testing.T) {
		err := repo.Delete(ctx, membershipID)
		require.NoError(t, err)

		// Verify membership is no longer findable
		_, err = repo.FindByID(ctx, membershipID)
		require.ErrorIs(t, err, community.ErrMembershipNotFound)
	})

	t.Run("returns ErrMembershipNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewMembershipID()
		err := repo.Delete(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrMembershipNotFound)
	})
}

func TestGroupMembershipRepository_ExistsActiveByGroupAndUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupMembershipRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	activeMemberID := identity.NewUserID()
	invitedMemberID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test users and group
	createTestUser(t, db, ownerID)
	createTestUser(t, db, activeMemberID)
	createTestUser(t, db, invitedMemberID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	// Create active and invited memberships
	activeMembershipID := community.NewMembershipID()
	invitedMembershipID := community.NewMembershipID()

	createTestMembership(t, db, activeMembershipID, groupID, activeMemberID, "member", "active")
	createTestMembership(t, db, invitedMembershipID, groupID, invitedMemberID, "member", "invited")

	t.Run("returns true for active membership", func(t *testing.T) {
		exists, err := repo.ExistsActiveByGroupAndUser(ctx, groupID, activeMemberID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for invited membership", func(t *testing.T) {
		exists, err := repo.ExistsActiveByGroupAndUser(ctx, groupID, invitedMemberID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns false for non-member", func(t *testing.T) {
		nonMemberID := identity.NewUserID()
		createTestUser(t, db, nonMemberID)

		exists, err := repo.ExistsActiveByGroupAndUser(ctx, groupID, nonMemberID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
