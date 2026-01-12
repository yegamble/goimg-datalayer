package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group membership operations.
const (
	sqlInsertGroupMembership = `
		INSERT INTO group_memberships (
			id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	sqlUpdateGroupMembership = `
		UPDATE group_memberships
		SET role = $2,
		    status = $3,
		    updated_at = $4
		WHERE id = $1
	`

	sqlSelectMembershipByID = `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE id = $1
	`

	sqlSelectMembershipByGroupAndUser = `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE group_id = $1 AND user_id = $2
	`

	sqlDeleteMembership = `
		DELETE FROM group_memberships WHERE id = $1
	`

	sqlExistsMembershipByGroupAndUser = `
		SELECT EXISTS(
			SELECT 1 FROM group_memberships
			WHERE group_id = $1 AND user_id = $2 AND status = 'active'
		)
	`
)

// membershipRow represents a group membership row in the database.
type membershipRow struct {
	ID        string         `db:"id"`
	GroupID   string         `db:"group_id"`
	UserID    string         `db:"user_id"`
	Role      string         `db:"role"`
	Status    string         `db:"status"`
	InvitedBy sql.NullString `db:"invited_by"`
	JoinedAt  sql.NullTime   `db:"joined_at"`
	UpdatedAt sql.NullTime   `db:"updated_at"`
}

// GroupMembershipRepository implements the community.GroupMembershipRepository interface for PostgreSQL.
type GroupMembershipRepository struct {
	db *sqlx.DB
}

// NewGroupMembershipRepository creates a new GroupMembershipRepository with the given database connection.
func NewGroupMembershipRepository(db *sqlx.DB) *GroupMembershipRepository {
	return &GroupMembershipRepository{db: db}
}

// FindByID retrieves a membership by its ID.
// Returns ErrMembershipNotFound if the membership doesn't exist.
func (r *GroupMembershipRepository) FindByID(ctx context.Context, id community.MembershipID) (*community.GroupMembership, error) {
	var row membershipRow
	if err := r.db.GetContext(ctx, &row, sqlSelectMembershipByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrMembershipNotFound
		}
		return nil, fmt.Errorf("failed to find membership by id: %w", err)
	}

	membership, err := rowToMembership(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to membership: %w", err)
	}

	return membership, nil
}

// FindByGroupAndUser retrieves a membership for a specific user in a specific group.
// Returns ErrMembershipNotFound if the membership doesn't exist.
func (r *GroupMembershipRepository) FindByGroupAndUser(
	ctx context.Context,
	groupID community.GroupID,
	userID identity.UserID,
) (*community.GroupMembership, error) {
	var row membershipRow
	if err := r.db.GetContext(ctx, &row, sqlSelectMembershipByGroupAndUser, groupID.String(), userID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrMembershipNotFound
		}
		return nil, fmt.Errorf("failed to find membership by group and user: %w", err)
	}

	membership, err := rowToMembership(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to membership: %w", err)
	}

	return membership, nil
}

// FindByGroup retrieves all memberships for a specific group with filtering and pagination.
// Returns the memberships, total count, and error.
func (r *GroupMembershipRepository) FindByGroup(
	ctx context.Context,
	groupID community.GroupID,
	filter community.MemberFilter,
	pagination shared.Pagination,
) ([]*community.GroupMembership, int, error) {
	// Build dynamic query
	query, countQuery, args := r.buildFindByGroupQuery(groupID, filter, pagination)

	// Execute main query
	var rows []membershipRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find memberships by group: %w", err)
	}

	// Execute count query
	var total int
	countArgs := args[:len(args)-2] // Exclude LIMIT and OFFSET
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count memberships by group: %w", err)
	}

	// Convert rows to domain entities
	memberships, err := rowsToMemberships(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to memberships: %w", err)
	}

	return memberships, total, nil
}

// FindByUser retrieves all groups a user is a member of with pagination.
// Returns the memberships, total count, and error.
func (r *GroupMembershipRepository) FindByUser(
	ctx context.Context,
	userID identity.UserID,
	pagination shared.Pagination,
) ([]*community.GroupMembership, int, error) {
	query := `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE user_id = $1
		ORDER BY joined_at DESC
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM group_memberships
		WHERE user_id = $1
	`

	// Execute main query
	var rows []membershipRow
	err := r.db.SelectContext(ctx, &rows, query, userID.String(), pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find memberships by user: %w", err)
	}

	// Execute count query
	var total int
	err = r.db.GetContext(ctx, &total, countQuery, userID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count memberships by user: %w", err)
	}

	// Convert rows to domain entities
	memberships, err := rowsToMemberships(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to memberships: %w", err)
	}

	return memberships, total, nil
}

// FindActiveByGroup retrieves all active members of a group.
// This is a convenience method that filters by MemberStatusActive.
func (r *GroupMembershipRepository) FindActiveByGroup(
	ctx context.Context,
	groupID community.GroupID,
) ([]*community.GroupMembership, error) {
	query := `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE group_id = $1 AND status = 'active'
		ORDER BY joined_at ASC
	`

	var rows []membershipRow
	err := r.db.SelectContext(ctx, &rows, query, groupID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find active memberships: %w", err)
	}

	memberships, err := rowsToMemberships(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows to memberships: %w", err)
	}

	return memberships, nil
}

// FindPendingByGroup retrieves all pending members (invited or requested) of a group.
func (r *GroupMembershipRepository) FindPendingByGroup(
	ctx context.Context,
	groupID community.GroupID,
) ([]*community.GroupMembership, error) {
	query := `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE group_id = $1 AND status IN ('invited', 'requested')
		ORDER BY joined_at DESC
	`

	var rows []membershipRow
	err := r.db.SelectContext(ctx, &rows, query, groupID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find pending memberships: %w", err)
	}

	memberships, err := rowsToMemberships(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows to memberships: %w", err)
	}

	return memberships, nil
}

// CountByGroupAndStatus counts memberships by status for a specific group.
// Used for checking member limits and pending request counts.
func (r *GroupMembershipRepository) CountByGroupAndStatus(
	ctx context.Context,
	groupID community.GroupID,
	status community.MemberStatus,
) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM group_memberships
		WHERE group_id = $1 AND status = $2
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, groupID.String(), status.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count memberships by status: %w", err)
	}

	return count, nil
}

// Save persists the membership to storage.
// This handles both creation and updates.
func (r *GroupMembershipRepository) Save(ctx context.Context, membership *community.GroupMembership) error {
	// Check if membership exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM group_memberships WHERE id = $1)", membership.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check membership existence: %w", err)
	}

	if exists {
		return r.update(ctx, membership)
	}
	return r.insert(ctx, membership)
}

// Delete removes the membership from storage.
func (r *GroupMembershipRepository) Delete(ctx context.Context, id community.MembershipID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteMembership, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete membership: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrMembershipNotFound
	}

	return nil
}

// ExistsActiveByGroupAndUser checks if an active membership exists for the given group and user.
func (r *GroupMembershipRepository) ExistsActiveByGroupAndUser(
	ctx context.Context,
	groupID community.GroupID,
	userID identity.UserID,
) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsMembershipByGroupAndUser, groupID.String(), userID.String())
	if err != nil {
		return false, fmt.Errorf("failed to check active membership existence: %w", err)
	}
	return exists, nil
}

// insert creates a new membership in the database.
func (r *GroupMembershipRepository) insert(ctx context.Context, membership *community.GroupMembership) error {
	var invitedBy sql.NullString
	if membership.InvitedBy() != nil {
		invitedBy = sql.NullString{String: membership.InvitedBy().String(), Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertGroupMembership,
		membership.ID().String(),
		membership.GroupID().String(),
		membership.UserID().String(),
		membership.Role().String(),
		membership.Status().String(),
		invitedBy,
		membership.JoinedAt(),
		membership.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert membership: %w", err)
	}

	return nil
}

// update updates an existing membership in the database.
func (r *GroupMembershipRepository) update(ctx context.Context, membership *community.GroupMembership) error {
	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateGroupMembership,
		membership.ID().String(),
		membership.Role().String(),
		membership.Status().String(),
		membership.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update membership: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrMembershipNotFound
	}

	return nil
}

// buildFindByGroupQuery constructs a dynamic SQL query for finding memberships by group.
func (r *GroupMembershipRepository) buildFindByGroupQuery(
	groupID community.GroupID,
	filter community.MemberFilter,
	pagination shared.Pagination,
) (string, string, []interface{}) {
	baseQuery := `
		SELECT id, group_id, user_id, role, status, invited_by, joined_at, updated_at
		FROM group_memberships
		WHERE group_id = $1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM group_memberships
		WHERE group_id = $1
	`

	var conditions []string
	args := []interface{}{groupID.String()}
	paramIndex := 2

	// Filter by role
	if filter.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", paramIndex))
		args = append(args, filter.Role.String())
		paramIndex++
	}

	// Filter by status
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramIndex))
		args = append(args, filter.Status.String())
		paramIndex++
	}

	// Add conditions to queries
	if len(conditions) > 0 {
		conditionStr := " AND " + strings.Join(conditions, " AND ")
		baseQuery += conditionStr
		countQuery += conditionStr
	}

	// Add ORDER BY
	baseQuery += " ORDER BY joined_at DESC"

	// Add pagination
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	args = append(args, pagination.Limit(), pagination.Offset())

	return baseQuery, countQuery, args
}

// rowToMembership converts a database row to a domain GroupMembership entity.
func rowToMembership(row membershipRow) (*community.GroupMembership, error) {
	// Parse IDs
	membershipID, err := community.ParseMembershipID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid membership id: %w", err)
	}

	groupID, err := community.ParseGroupID(row.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Parse enums
	role, err := community.ParseGroupRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf("invalid role: %w", err)
	}

	status, err := community.ParseMemberStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Parse optional invited_by
	var invitedBy *identity.UserID
	if row.InvitedBy.Valid {
		inviterID, err := identity.ParseUserID(row.InvitedBy.String)
		if err != nil {
			return nil, fmt.Errorf("invalid inviter id: %w", err)
		}
		invitedBy = &inviterID
	}

	// Reconstitute membership without validation or events
	membership := community.ReconstructGroupMembership(
		membershipID,
		groupID,
		userID,
		role,
		status,
		invitedBy,
		row.JoinedAt.Time,
		row.UpdatedAt.Time,
	)

	return membership, nil
}

// rowsToMemberships converts multiple membership rows to domain entities.
func rowsToMemberships(rows []membershipRow) ([]*community.GroupMembership, error) {
	memberships := make([]*community.GroupMembership, 0, len(rows))
	for _, row := range rows {
		membership, err := rowToMembership(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row: %w", err)
		}
		memberships = append(memberships, membership)
	}
	return memberships, nil
}
