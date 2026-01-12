package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group operations.
const (
	sqlInsertGroup = `
		INSERT INTO groups (
			id, name, slug, description, group_type, owner_id,
			require_approval, allow_member_invites, allow_member_albums, max_members,
			member_count, image_count, album_count, cover_image_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	sqlUpdateGroup = `
		UPDATE groups
		SET name = $2,
		    slug = $3,
		    description = $4,
		    group_type = $5,
		    owner_id = $6,
		    require_approval = $7,
		    allow_member_invites = $8,
		    allow_member_albums = $9,
		    max_members = $10,
		    member_count = $11,
		    image_count = $12,
		    album_count = $13,
		    cover_image_id = $14,
		    updated_at = $15
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectGroupByID = `
		SELECT id, name, slug, description, group_type, owner_id,
		       require_approval, allow_member_invites, allow_member_albums, max_members,
		       member_count, image_count, album_count, cover_image_id,
		       created_at, updated_at
		FROM groups
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlSelectGroupBySlug = `
		SELECT id, name, slug, description, group_type, owner_id,
		       require_approval, allow_member_invites, allow_member_albums, max_members,
		       member_count, image_count, album_count, cover_image_id,
		       created_at, updated_at
		FROM groups
		WHERE slug = $1 AND deleted_at IS NULL
	`

	sqlSoftDeleteGroup = `
		UPDATE groups
		SET deleted_at = $2,
		    updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	sqlExistsGroupWithSlug = `
		SELECT EXISTS(SELECT 1 FROM groups WHERE slug = $1 AND deleted_at IS NULL)
	`
)

// groupRow represents a group row in the database.
type groupRow struct {
	ID                 string         `db:"id"`
	Name               string         `db:"name"`
	Slug               string         `db:"slug"`
	Description        string         `db:"description"`
	GroupType          string         `db:"group_type"`
	OwnerID            string         `db:"owner_id"`
	RequireApproval    bool           `db:"require_approval"`
	AllowMemberInvites bool           `db:"allow_member_invites"`
	AllowMemberAlbums  bool           `db:"allow_member_albums"`
	MaxMembers         int            `db:"max_members"`
	MemberCount        int            `db:"member_count"`
	ImageCount         int            `db:"image_count"`
	AlbumCount         int            `db:"album_count"`
	CoverImageID       sql.NullString `db:"cover_image_id"`
	CreatedAt          sql.NullTime   `db:"created_at"`
	UpdatedAt          sql.NullTime   `db:"updated_at"`
}

// GroupRepository implements the community.GroupRepository interface for PostgreSQL.
type GroupRepository struct {
	db *sqlx.DB
}

// NewGroupRepository creates a new GroupRepository with the given database connection.
func NewGroupRepository(db *sqlx.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// FindByID retrieves a group by its ID.
// Returns ErrGroupNotFound if the group doesn't exist.
func (r *GroupRepository) FindByID(ctx context.Context, id community.GroupID) (*community.Group, error) {
	var row groupRow
	if err := r.db.GetContext(ctx, &row, sqlSelectGroupByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to find group by id: %w", err)
	}

	group, err := rowToGroup(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to group: %w", err)
	}

	return group, nil
}

// FindBySlug retrieves a group by its slug.
// Returns ErrGroupNotFound if the group doesn't exist.
func (r *GroupRepository) FindBySlug(ctx context.Context, slug community.GroupSlug) (*community.Group, error) {
	var row groupRow
	if err := r.db.GetContext(ctx, &row, sqlSelectGroupBySlug, slug.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to find group by slug: %w", err)
	}

	group, err := rowToGroup(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to group: %w", err)
	}

	return group, nil
}

// FindPublicGroups retrieves a paginated list of public and invite-only groups.
// Private groups are excluded from this query.
// Returns the groups, total count, and error.
func (r *GroupRepository) FindPublicGroups(
	ctx context.Context,
	filter community.GroupFilter,
	pagination shared.Pagination,
) ([]*community.Group, int, error) {
	// Build dynamic query
	query, countQuery, args := r.buildPublicGroupsQuery(filter, pagination)

	// Execute main query
	var rows []groupRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find public groups: %w", err)
	}

	// Execute count query
	var total int
	countArgs := args[:len(args)-2] // Exclude LIMIT and OFFSET
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count public groups: %w", err)
	}

	// Convert rows to domain entities
	groups, err := rowsToGroups(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to groups: %w", err)
	}

	return groups, total, nil
}

// SearchGroups searches for groups by name or description.
// Only returns public and invite-only groups (private groups are not searchable).
// Returns the matching groups, total count, and error.
func (r *GroupRepository) SearchGroups(
	ctx context.Context,
	query string,
	pagination shared.Pagination,
) ([]*community.Group, int, error) {
	// Build search query with trigram similarity
	sqlQuery := `
		SELECT id, name, slug, description, group_type, owner_id,
		       require_approval, allow_member_invites, allow_member_albums, max_members,
		       member_count, image_count, album_count, cover_image_id,
		       created_at, updated_at
		FROM groups
		WHERE deleted_at IS NULL
		  AND group_type IN ('public', 'invite_only')
		  AND (
		      name ILIKE $1
		      OR description ILIKE $1
		      OR similarity(name, $2) > 0.3
		  )
		ORDER BY similarity(name, $2) DESC, member_count DESC
		LIMIT $3 OFFSET $4
	`

	countQuery := `
		SELECT COUNT(*)
		FROM groups
		WHERE deleted_at IS NULL
		  AND group_type IN ('public', 'invite_only')
		  AND (
		      name ILIKE $1
		      OR description ILIKE $1
		      OR similarity(name, $2) > 0.3
		  )
	`

	searchPattern := "%" + query + "%"

	// Execute main query
	var rows []groupRow
	err := r.db.SelectContext(ctx, &rows, sqlQuery, searchPattern, query, pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search groups: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, countQuery, searchPattern, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Convert rows to domain entities
	groups, err := rowsToGroups(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to groups: %w", err)
	}

	return groups, total, nil
}

// FindByOwner retrieves all groups owned by a specific user.
func (r *GroupRepository) FindByOwner(
	ctx context.Context,
	ownerID identity.UserID,
	pagination shared.Pagination,
) ([]*community.Group, int, error) {
	query := `
		SELECT id, name, slug, description, group_type, owner_id,
		       require_approval, allow_member_invites, allow_member_albums, max_members,
		       member_count, image_count, album_count, cover_image_id,
		       created_at, updated_at
		FROM groups
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM groups
		WHERE owner_id = $1 AND deleted_at IS NULL
	`

	// Execute main query
	var rows []groupRow
	err := r.db.SelectContext(ctx, &rows, query, ownerID.String(), pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find groups by owner: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, countQuery, ownerID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count groups by owner: %w", err)
	}

	// Convert rows to domain entities
	groups, err := rowsToGroups(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to groups: %w", err)
	}

	return groups, total, nil
}

// Save persists the group to storage.
// This handles both creation and updates.
func (r *GroupRepository) Save(ctx context.Context, group *community.Group) error {
	// Check if group exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", group.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	if exists {
		return r.update(ctx, group)
	}
	return r.insert(ctx, group)
}

// Delete removes the group from storage.
// This is typically a soft delete.
func (r *GroupRepository) Delete(ctx context.Context, id community.GroupID) error {
	now := sql.NullTime{Time: shared.Now(), Valid: true}
	result, err := r.db.ExecContext(ctx, sqlSoftDeleteGroup, id.String(), now)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrGroupNotFound
	}

	return nil
}

// ExistsWithSlug checks if a group with the given slug exists.
// Used for slug uniqueness validation.
func (r *GroupRepository) ExistsWithSlug(ctx context.Context, slug community.GroupSlug) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsGroupWithSlug, slug.String())
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}
	return exists, nil
}

// insert creates a new group in the database.
func (r *GroupRepository) insert(ctx context.Context, group *community.Group) error {
	settings := group.Settings()
	var coverImageID sql.NullString
	if group.CoverImageID() != nil {
		coverImageID = sql.NullString{String: group.CoverImageID().String(), Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertGroup,
		group.ID().String(),
		group.Name().String(),
		group.Slug().String(),
		group.Description(),
		group.GroupType().String(),
		group.OwnerID().String(),
		settings.RequireApproval(),
		settings.AllowMemberInvites(),
		settings.AllowMemberAlbums(),
		settings.MaxMembers(),
		group.MemberCount(),
		group.ImageCount(),
		group.AlbumCount(),
		coverImageID,
		group.CreatedAt(),
		group.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert group: %w", err)
	}

	return nil
}

// update updates an existing group in the database.
func (r *GroupRepository) update(ctx context.Context, group *community.Group) error {
	settings := group.Settings()
	var coverImageID sql.NullString
	if group.CoverImageID() != nil {
		coverImageID = sql.NullString{String: group.CoverImageID().String(), Valid: true}
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateGroup,
		group.ID().String(),
		group.Name().String(),
		group.Slug().String(),
		group.Description(),
		group.GroupType().String(),
		group.OwnerID().String(),
		settings.RequireApproval(),
		settings.AllowMemberInvites(),
		settings.AllowMemberAlbums(),
		settings.MaxMembers(),
		group.MemberCount(),
		group.ImageCount(),
		group.AlbumCount(),
		coverImageID,
		group.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrGroupNotFound
	}

	return nil
}

// buildPublicGroupsQuery constructs a dynamic SQL query for finding public groups.
func (r *GroupRepository) buildPublicGroupsQuery(
	filter community.GroupFilter,
	pagination shared.Pagination,
) (string, string, []interface{}) {
	baseQuery := `
		SELECT id, name, slug, description, group_type, owner_id,
		       require_approval, allow_member_invites, allow_member_albums, max_members,
		       member_count, image_count, album_count, cover_image_id,
		       created_at, updated_at
		FROM groups
		WHERE deleted_at IS NULL
		  AND group_type IN ('public', 'invite_only')
	`

	countQuery := `
		SELECT COUNT(*)
		FROM groups
		WHERE deleted_at IS NULL
		  AND group_type IN ('public', 'invite_only')
	`

	var conditions []string
	var args []interface{}
	paramIndex := 1

	// Filter by group type
	if filter.GroupType != nil {
		conditions = append(conditions, fmt.Sprintf("group_type = $%d", paramIndex))
		args = append(args, filter.GroupType.String())
		paramIndex++
	}

	// Filter by owner
	if filter.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", paramIndex))
		args = append(args, filter.OwnerID.String())
		paramIndex++
	}

	// Add conditions to queries
	if len(conditions) > 0 {
		conditionStr := " AND " + strings.Join(conditions, " AND ")
		baseQuery += conditionStr
		countQuery += conditionStr
	}

	// Add ORDER BY based on sort
	switch filter.SortBy {
	case community.GroupSortByRecent:
		baseQuery += " ORDER BY created_at DESC"
	case community.GroupSortByPopular:
		baseQuery += " ORDER BY member_count DESC, created_at DESC"
	case community.GroupSortByName:
		baseQuery += " ORDER BY name ASC"
	case community.GroupSortByActivity:
		baseQuery += " ORDER BY updated_at DESC"
	default:
		baseQuery += " ORDER BY created_at DESC"
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	args = append(args, pagination.Limit(), pagination.Offset())

	return baseQuery, countQuery, args
}

// rowToGroup converts a database row to a domain Group entity.
func rowToGroup(row groupRow) (*community.Group, error) {
	// Parse IDs
	groupID, err := community.ParseGroupID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}

	ownerID, err := identity.ParseUserID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	// Parse value objects
	name, err := community.NewGroupName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid group name: %w", err)
	}

	slug, err := community.NewGroupSlug(row.Slug)
	if err != nil {
		return nil, fmt.Errorf("invalid group slug: %w", err)
	}

	groupType, err := community.ParseGroupType(row.GroupType)
	if err != nil {
		return nil, fmt.Errorf("invalid group type: %w", err)
	}

	// Parse settings
	settings, err := community.NewGroupSettings(
		row.RequireApproval,
		row.AllowMemberInvites,
		row.AllowMemberAlbums,
		row.MaxMembers,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid group settings: %w", err)
	}

	// Parse optional cover image ID
	var coverImageID *gallery.ImageID
	if row.CoverImageID.Valid {
		imageID, err := gallery.ParseImageID(row.CoverImageID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid cover image id: %w", err)
		}
		coverImageID = &imageID
	}

	// Reconstitute group without validation or events
	group := community.ReconstructGroup(
		groupID,
		name,
		slug,
		row.Description,
		groupType,
		ownerID,
		settings,
		row.MemberCount,
		row.ImageCount,
		row.AlbumCount,
		coverImageID,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)

	return group, nil
}

// rowsToGroups converts multiple group rows to domain entities.
func rowsToGroups(rows []groupRow) ([]*community.Group, error) {
	groups := make([]*community.Group, 0, len(rows))
	for _, row := range rows {
		group, err := rowToGroup(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row: %w", err)
		}
		groups = append(groups, group)
	}
	return groups, nil
}
