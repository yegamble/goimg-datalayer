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

// createTestGroup inserts a test group into the database.
func createTestGroup(t *testing.T, db *sqlx.DB, groupID community.GroupID, ownerID identity.UserID, slug string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO groups (
			id, name, slug, description, group_type, owner_id,
			require_approval, allow_member_invites, allow_member_albums, max_members,
			member_count, image_count, album_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		groupID.String(),
		"Test Group",
		slug,
		"Test description",
		"public",
		ownerID.String(),
		false,
		true,
		true,
		0,
		1, // member_count (owner)
		0, // image_count
		0, // album_count
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func TestGroupRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()

	// Setup: Create test user
	createTestUser(t, db, ownerID)

	t.Run("saves new group successfully", func(t *testing.T) {
		name, err := community.NewGroupName("Photography Lovers")
		require.NoError(t, err)

		slug, err := community.NewGroupSlug("photography-lovers")
		require.NoError(t, err)

		group, err := community.NewGroup(ownerID, name, slug, community.GroupTypePublic)
		require.NoError(t, err)

		err = repo.Save(ctx, group)
		require.NoError(t, err)

		// Verify group exists
		found, err := repo.FindByID(ctx, group.ID())
		require.NoError(t, err)
		assert.Equal(t, group.ID(), found.ID())
		assert.Equal(t, name.String(), found.Name().String())
		assert.Equal(t, slug.String(), found.Slug().String())
	})

	t.Run("updates existing group successfully", func(t *testing.T) {
		name, err := community.NewGroupName("Nature Photography")
		require.NoError(t, err)

		slug, err := community.NewGroupSlug("nature-photography")
		require.NoError(t, err)

		group, err := community.NewGroup(ownerID, name, slug, community.GroupTypePublic)
		require.NoError(t, err)

		// Save first time
		err = repo.Save(ctx, group)
		require.NoError(t, err)

		// Update description
		err = group.UpdateDescription("Updated description about nature photography")
		require.NoError(t, err)

		// Save again
		err = repo.Save(ctx, group)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, group.ID())
		require.NoError(t, err)
		assert.Equal(t, "Updated description about nature photography", found.Description())
	})

	t.Run("fails when owner does not exist", func(t *testing.T) {
		invalidOwnerID := identity.NewUserID()

		name, err := community.NewGroupName("Invalid Group")
		require.NoError(t, err)

		slug, err := community.NewGroupSlug("invalid-group")
		require.NoError(t, err)

		group, err := community.NewGroup(invalidOwnerID, name, slug, community.GroupTypePublic)
		require.NoError(t, err)

		err = repo.Save(ctx, group)
		assert.Error(t, err) // Foreign key constraint violation
	})
}

func TestGroupRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "test-group")

	t.Run("finds existing group by ID", func(t *testing.T) {
		group, err := repo.FindByID(ctx, groupID)
		require.NoError(t, err)
		assert.Equal(t, groupID, group.ID())
		assert.Equal(t, "Test Group", group.Name().String())
		assert.Equal(t, "test-group", group.Slug().String())
	})

	t.Run("returns ErrGroupNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewGroupID()
		_, err := repo.FindByID(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrGroupNotFound)
	})
}

func TestGroupRepository_FindBySlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "unique-slug")

	t.Run("finds existing group by slug", func(t *testing.T) {
		slug, err := community.NewGroupSlug("unique-slug")
		require.NoError(t, err)

		group, err := repo.FindBySlug(ctx, slug)
		require.NoError(t, err)
		assert.Equal(t, groupID, group.ID())
		assert.Equal(t, "unique-slug", group.Slug().String())
	})

	t.Run("returns ErrGroupNotFound for non-existent slug", func(t *testing.T) {
		slug, err := community.NewGroupSlug("non-existent-slug")
		require.NoError(t, err)

		_, err = repo.FindBySlug(ctx, slug)
		require.ErrorIs(t, err, community.ErrGroupNotFound)
	})
}

func TestGroupRepository_FindPublicGroups(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()

	// Setup: Create test user
	createTestUser(t, db, ownerID)

	// Create multiple groups with different types
	publicGroup1 := community.NewGroupID()
	publicGroup2 := community.NewGroupID()
	privateGroup := community.NewGroupID()

	createTestGroup(t, db, publicGroup1, ownerID, "public-group-1")
	createTestGroup(t, db, publicGroup2, ownerID, "public-group-2")

	// Create private group
	_, err := db.Exec(`
		INSERT INTO groups (
			id, name, slug, description, group_type, owner_id,
			require_approval, allow_member_invites, allow_member_albums, max_members,
			member_count, image_count, album_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		privateGroup.String(),
		"Private Group",
		"private-group",
		"Private description",
		"private",
		ownerID.String(),
		false,
		false,
		true,
		0,
		1,
		0,
		0,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)

	t.Run("returns only public and invite-only groups", func(t *testing.T) {
		filter := community.GroupFilter{
			SortBy: community.GroupSortByRecent,
		}
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.FindPublicGroups(ctx, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 2)

		// Verify no private groups in results
		for _, group := range groups {
			assert.NotEqual(t, community.GroupTypePrivate, group.GroupType())
		}
	})

	t.Run("filters by group type", func(t *testing.T) {
		groupType := community.GroupTypePublic
		filter := community.GroupFilter{
			GroupType: &groupType,
			SortBy:    community.GroupSortByRecent,
		}
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.FindPublicGroups(ctx, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 2)

		for _, group := range groups {
			assert.Equal(t, community.GroupTypePublic, group.GroupType())
		}
	})

	t.Run("filters by owner", func(t *testing.T) {
		filter := community.GroupFilter{
			OwnerID: &ownerID,
			SortBy:  community.GroupSortByRecent,
		}
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.FindPublicGroups(ctx, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 2)

		for _, group := range groups {
			assert.Equal(t, ownerID, group.OwnerID())
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		filter := community.GroupFilter{
			SortBy: community.GroupSortByRecent,
		}
		pagination := shared.NewPagination(0, 1)

		groups, total, err := repo.FindPublicGroups(ctx, filter, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 1)
	})
}

func TestGroupRepository_SearchGroups(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()

	// Setup: Create test user
	createTestUser(t, db, ownerID)

	// Create groups with different names
	photographyGroup := community.NewGroupID()
	natureGroup := community.NewGroupID()

	createTestGroup(t, db, photographyGroup, ownerID, "photography-lovers")

	_, err := db.Exec(`
		INSERT INTO groups (
			id, name, slug, description, group_type, owner_id,
			require_approval, allow_member_invites, allow_member_albums, max_members,
			member_count, image_count, album_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		natureGroup.String(),
		"Nature Photography",
		"nature-photography",
		"A group for nature photography enthusiasts",
		"public",
		ownerID.String(),
		false,
		true,
		true,
		0,
		1,
		0,
		0,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err)

	t.Run("searches by name substring", func(t *testing.T) {
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.SearchGroups(ctx, "Photography", pagination)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("searches by description substring", func(t *testing.T) {
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.SearchGroups(ctx, "nature", pagination)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("returns empty results for non-matching query", func(t *testing.T) {
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.SearchGroups(ctx, "NonExistentGroupNameXYZ", pagination)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, groups)
	})
}

func TestGroupRepository_FindByOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	owner1ID := identity.NewUserID()
	owner2ID := identity.NewUserID()

	// Setup: Create test users
	createTestUser(t, db, owner1ID)
	createTestUser(t, db, owner2ID)

	// Create groups for owner1
	group1 := community.NewGroupID()
	group2 := community.NewGroupID()
	createTestGroup(t, db, group1, owner1ID, "owner1-group-1")
	createTestGroup(t, db, group2, owner1ID, "owner1-group-2")

	// Create group for owner2
	group3 := community.NewGroupID()
	createTestGroup(t, db, group3, owner2ID, "owner2-group-1")

	t.Run("finds all groups by owner", func(t *testing.T) {
		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.FindByOwner(ctx, owner1ID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 2)

		for _, group := range groups {
			assert.Equal(t, owner1ID, group.OwnerID())
		}
	})

	t.Run("returns empty results for owner with no groups", func(t *testing.T) {
		noGroupsOwner := identity.NewUserID()
		createTestUser(t, db, noGroupsOwner)

		pagination := shared.NewPagination(0, 10)

		groups, total, err := repo.FindByOwner(ctx, noGroupsOwner, pagination)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, groups)
	})

	t.Run("respects pagination", func(t *testing.T) {
		pagination := shared.NewPagination(0, 1)

		groups, total, err := repo.FindByOwner(ctx, owner1ID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, groups, 1)
	})
}

func TestGroupRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "deletable-group")

	t.Run("soft deletes group successfully", func(t *testing.T) {
		err := repo.Delete(ctx, groupID)
		require.NoError(t, err)

		// Verify group is no longer findable
		_, err = repo.FindByID(ctx, groupID)
		require.ErrorIs(t, err, community.ErrGroupNotFound)
	})

	t.Run("returns ErrGroupNotFound for non-existent ID", func(t *testing.T) {
		nonExistentID := community.NewGroupID()
		err := repo.Delete(ctx, nonExistentID)
		require.ErrorIs(t, err, community.ErrGroupNotFound)
	})
}

func TestGroupRepository_ExistsWithSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	repo := postgres.NewGroupRepository(db)

	ctx := context.Background()
	ownerID := identity.NewUserID()
	groupID := community.NewGroupID()

	// Setup: Create test user and group
	createTestUser(t, db, ownerID)
	createTestGroup(t, db, groupID, ownerID, "existing-slug")

	t.Run("returns true for existing slug", func(t *testing.T) {
		slug, err := community.NewGroupSlug("existing-slug")
		require.NoError(t, err)

		exists, err := repo.ExistsWithSlug(ctx, slug)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent slug", func(t *testing.T) {
		slug, err := community.NewGroupSlug("non-existent-slug")
		require.NoError(t, err)

		exists, err := repo.ExistsWithSlug(ctx, slug)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
