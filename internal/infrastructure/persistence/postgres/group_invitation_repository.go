package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for group invitation operations.
const (
	sqlInsertGroupInvitation = `
		INSERT INTO group_invitations (
			id, group_id, invited_by, email, user_id, token, expires_at, used_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	sqlUpdateGroupInvitation = `
		UPDATE group_invitations
		SET used_at = $2
		WHERE id = $1
	`

	sqlSelectInvitationByID = `
		SELECT id, group_id, invited_by, email, user_id, token, expires_at, used_at, created_at
		FROM group_invitations
		WHERE id = $1
	`

	sqlSelectInvitationByToken = `
		SELECT id, group_id, invited_by, email, user_id, token, expires_at, used_at, created_at
		FROM group_invitations
		WHERE token = $1
	` // #nosec G101

	sqlSelectPendingInvitationsByGroup = `
		SELECT id, group_id, invited_by, email, user_id, token, expires_at, used_at, created_at
		FROM group_invitations
		WHERE group_id = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	sqlSelectPendingInvitationsByUser = `
		SELECT id, group_id, invited_by, email, user_id, token, expires_at, used_at, created_at
		FROM group_invitations
		WHERE user_id = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	sqlDeleteInvitation = `
		DELETE FROM group_invitations WHERE id = $1
	`
)

// invitationRow represents a group invitation row in the database.
type invitationRow struct {
	ID        string         `db:"id"`
	GroupID   string         `db:"group_id"`
	InvitedBy string         `db:"invited_by"`
	Email     sql.NullString `db:"email"`
	UserID    sql.NullString `db:"user_id"`
	Token     string         `db:"token"`
	ExpiresAt sql.NullTime   `db:"expires_at"`
	UsedAt    sql.NullTime   `db:"used_at"`
	CreatedAt sql.NullTime   `db:"created_at"`
}

// GroupInvitationRepository implements the community.GroupInvitationRepository interface for PostgreSQL.
type GroupInvitationRepository struct {
	db *sqlx.DB
}

// NewGroupInvitationRepository creates a new GroupInvitationRepository with the given database connection.
func NewGroupInvitationRepository(db *sqlx.DB) *GroupInvitationRepository {
	return &GroupInvitationRepository{db: db}
}

// Save persists the invitation to storage.
// This handles both creation and updates (e.g., marking as used).
func (r *GroupInvitationRepository) Save(ctx context.Context, invitation *community.GroupInvitation) error {
	// Check if invitation already exists by trying to find it
	_, err := r.FindByID(ctx, invitation.ID())
	if err != nil {
		if errors.Is(err, community.ErrInvitationNotFound) {
			// Create new invitation
			return r.insert(ctx, invitation)
		}
		return fmt.Errorf("check invitation existence: %w", err)
	}

	// Update existing invitation (typically to mark as used)
	return r.update(ctx, invitation)
}

// insert creates a new invitation record.
func (r *GroupInvitationRepository) insert(ctx context.Context, invitation *community.GroupInvitation) error {
	var email, userID interface{}
	if invitation.Email() != nil {
		email = *invitation.Email()
	}
	if invitation.UserID() != nil {
		userID = invitation.UserID().String()
	}

	var usedAt interface{}
	if invitation.UsedAt() != nil {
		usedAt = *invitation.UsedAt()
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertGroupInvitation,
		invitation.ID().String(),
		invitation.GroupID().String(),
		invitation.InvitedBy().String(),
		email,
		userID,
		invitation.Token().String(),
		invitation.ExpiresAt(),
		usedAt,
		invitation.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("insert group invitation: %w", err)
	}

	return nil
}

// update updates an existing invitation record (typically to mark as used).
func (r *GroupInvitationRepository) update(ctx context.Context, invitation *community.GroupInvitation) error {
	var usedAt interface{}
	if invitation.UsedAt() != nil {
		usedAt = *invitation.UsedAt()
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlUpdateGroupInvitation,
		invitation.ID().String(),
		usedAt,
	)
	if err != nil {
		return fmt.Errorf("update group invitation: %w", err)
	}

	return nil
}

// FindByID retrieves an invitation by its ID.
// Returns ErrInvitationNotFound if the invitation doesn't exist.
func (r *GroupInvitationRepository) FindByID(ctx context.Context, id community.InvitationID) (*community.GroupInvitation, error) {
	var row invitationRow
	if err := r.db.GetContext(ctx, &row, sqlSelectInvitationByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("find invitation by id: %w", err)
	}

	invitation, err := rowToInvitation(row)
	if err != nil {
		return nil, fmt.Errorf("convert row to invitation: %w", err)
	}

	return invitation, nil
}

// FindByToken retrieves an invitation by its secure token.
// Returns ErrInvitationNotFound if the invitation doesn't exist.
func (r *GroupInvitationRepository) FindByToken(ctx context.Context, token community.InvitationToken) (*community.GroupInvitation, error) {
	var row invitationRow
	if err := r.db.GetContext(ctx, &row, sqlSelectInvitationByToken, token.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("find invitation by token: %w", err)
	}

	invitation, err := rowToInvitation(row)
	if err != nil {
		return nil, fmt.Errorf("convert row to invitation: %w", err)
	}

	return invitation, nil
}

// FindPendingByGroup retrieves all pending (unused, non-expired) invitations for a group.
func (r *GroupInvitationRepository) FindPendingByGroup(ctx context.Context, groupID community.GroupID) ([]*community.GroupInvitation, error) {
	var rows []invitationRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectPendingInvitationsByGroup, groupID.String())
	if err != nil {
		return nil, fmt.Errorf("find pending invitations by group: %w", err)
	}

	invitations, err := rowsToInvitations(rows)
	if err != nil {
		return nil, fmt.Errorf("convert rows to invitations: %w", err)
	}

	return invitations, nil
}

// FindPendingByUser retrieves all pending invitations sent to a specific user.
func (r *GroupInvitationRepository) FindPendingByUser(ctx context.Context, userID identity.UserID) ([]*community.GroupInvitation, error) {
	var rows []invitationRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectPendingInvitationsByUser, userID.String())
	if err != nil {
		return nil, fmt.Errorf("find pending invitations by user: %w", err)
	}

	invitations, err := rowsToInvitations(rows)
	if err != nil {
		return nil, fmt.Errorf("convert rows to invitations: %w", err)
	}

	return invitations, nil
}

// Delete removes the invitation from storage.
func (r *GroupInvitationRepository) Delete(ctx context.Context, id community.InvitationID) error {
	_, err := r.db.ExecContext(ctx, sqlDeleteInvitation, id.String())
	if err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	return nil
}

// rowToInvitation converts a database row to a GroupInvitation domain entity.
func rowToInvitation(row invitationRow) (*community.GroupInvitation, error) {
	id, err := community.ParseInvitationID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse invitation id: %w", err)
	}

	groupID, err := community.ParseGroupID(row.GroupID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}

	invitedBy, err := identity.ParseUserID(row.InvitedBy)
	if err != nil {
		return nil, fmt.Errorf("parse invited by user id: %w", err)
	}

	token, err := community.ParseInvitationToken(row.Token)
	if err != nil {
		return nil, fmt.Errorf("parse invitation token: %w", err)
	}

	var email *string
	if row.Email.Valid {
		email = &row.Email.String
	}

	var userID *identity.UserID
	if row.UserID.Valid {
		uid, err := identity.ParseUserID(row.UserID.String)
		if err != nil {
			return nil, fmt.Errorf("parse user id: %w", err)
		}
		userID = &uid
	}

	var usedAt *sql.NullTime
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt
	}

	var usedAtPtr *time.Time
	if usedAt != nil && usedAt.Valid {
		usedAtPtr = &usedAt.Time
	}

	return community.ReconstructGroupInvitation(
		id,
		groupID,
		invitedBy,
		email,
		userID,
		token,
		row.ExpiresAt.Time,
		usedAtPtr,
		row.CreatedAt.Time,
	), nil
}

// rowsToInvitations converts multiple database rows to GroupInvitation domain entities.
func rowsToInvitations(rows []invitationRow) ([]*community.GroupInvitation, error) {
	invitations := make([]*community.GroupInvitation, 0, len(rows))
	for _, row := range rows {
		invitation, err := rowToInvitation(row)
		if err != nil {
			return nil, fmt.Errorf("convert row to invitation: %w", err)
		}
		invitations = append(invitations, invitation)
	}
	return invitations, nil
}
