package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for group image operations.
const (
	sqlInsertGroupImage = `
		INSERT INTO group_images (
			id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	sqlUpdateGroupImage = `
		UPDATE group_images SET
			status = $2,
			reviewed_by = $3,
			reviewed_at = $4
		WHERE id = $1
	`

	sqlSelectGroupImageByID = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE id = $1
	`

	sqlSelectGroupImageByGroupAndImage = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE group_id = $1 AND image_id = $2
	`

	sqlSelectGroupImagesByGroup = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE group_id = $1
		ORDER BY shared_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlSelectGroupImagesByGroupAndStatus = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE group_id = $1 AND status = $2
		ORDER BY shared_at DESC
		LIMIT $3 OFFSET $4
	`

	sqlSelectPendingGroupImagesByGroup = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE group_id = $1 AND status = 'pending'
		ORDER BY shared_at ASC
		LIMIT $2 OFFSET $3
	`

	sqlSelectApprovedGroupImagesByGroup = `
		SELECT id, group_id, image_id, shared_by, status, reviewed_by, shared_at, reviewed_at
		FROM group_images
		WHERE group_id = $1 AND status = 'approved'
		ORDER BY shared_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountGroupImagesByGroup = `
		SELECT COUNT(*) FROM group_images WHERE group_id = $1
	`

	sqlCountGroupImagesByGroupAndStatus = `
		SELECT COUNT(*) FROM group_images WHERE group_id = $1 AND status = $2
	`

	sqlDeleteGroupImage = `
		DELETE FROM group_images WHERE id = $1
	`

	sqlExistsGroupImageByGroupAndImage = `
		SELECT EXISTS(SELECT 1 FROM group_images WHERE group_id = $1 AND image_id = $2)
	`
)

// groupImageRow represents a group image row in the database.
type groupImageRow struct {
	ID         string         `db:"id"`
	GroupID    string         `db:"group_id"`
	ImageID    string         `db:"image_id"`
	SharedBy   string         `db:"shared_by"`
	Status     string         `db:"status"`
	ReviewedBy sql.NullString `db:"reviewed_by"`
	SharedAt   time.Time      `db:"shared_at"`
	ReviewedAt sql.NullTime   `db:"reviewed_at"`
}

// GroupImageRepository implements the community.GroupImageRepository interface for PostgreSQL.
type GroupImageRepository struct {
	db *sqlx.DB
}

// NewGroupImageRepository creates a new GroupImageRepository with the given database connection.
func NewGroupImageRepository(db *sqlx.DB) *GroupImageRepository {
	return &GroupImageRepository{db: db}
}

// Save persists the group image to storage.
func (r *GroupImageRepository) Save(ctx context.Context, groupImage *community.GroupImage) error {
	// Check if this is an insert or update
	var exists bool
	if err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM group_images WHERE id = $1)", groupImage.ID().String()); err != nil {
		return fmt.Errorf("failed to check group image existence: %w", err)
	}

	var reviewedBy sql.NullString
	if groupImage.ReviewedBy() != nil {
		reviewedBy = sql.NullString{String: groupImage.ReviewedBy().String(), Valid: true}
	}

	var reviewedAt sql.NullTime
	if groupImage.ReviewedAt() != nil {
		reviewedAt = sql.NullTime{Time: *groupImage.ReviewedAt(), Valid: true}
	}

	if exists {
		// Update
		_, err := r.db.ExecContext(
			ctx,
			sqlUpdateGroupImage,
			groupImage.ID().String(),
			groupImage.Status().String(),
			reviewedBy,
			reviewedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to update group image: %w", err)
		}
	} else {
		// Insert
		_, err := r.db.ExecContext(
			ctx,
			sqlInsertGroupImage,
			groupImage.ID().String(),
			groupImage.GroupID().String(),
			groupImage.ImageID().String(),
			groupImage.SharedBy().String(),
			groupImage.Status().String(),
			reviewedBy,
			groupImage.SharedAt(),
			reviewedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert group image: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a group image by its ID.
func (r *GroupImageRepository) FindByID(ctx context.Context, id community.GroupImageID) (*community.GroupImage, error) {
	var row groupImageRow
	if err := r.db.GetContext(ctx, &row, sqlSelectGroupImageByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrGroupImageNotFound
		}
		return nil, fmt.Errorf("failed to find group image by id: %w", err)
	}

	return rowToGroupImage(row)
}

// FindByGroupAndImage retrieves a group image by group ID and image ID.
func (r *GroupImageRepository) FindByGroupAndImage(
	ctx context.Context,
	groupID community.GroupID,
	imageID gallery.ImageID,
) (*community.GroupImage, error) {
	var row groupImageRow
	if err := r.db.GetContext(ctx, &row, sqlSelectGroupImageByGroupAndImage, groupID.String(), imageID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, community.ErrGroupImageNotFound
		}
		return nil, fmt.Errorf("failed to find group image by group and image: %w", err)
	}

	return rowToGroupImage(row)
}

// FindByGroup retrieves images for a specific group with optional status filtering.
func (r *GroupImageRepository) FindByGroup(
	ctx context.Context,
	groupID community.GroupID,
	status *community.GroupImageStatus,
	pagination shared.Pagination,
) ([]*community.GroupImage, int, error) {
	var rows []groupImageRow
	var total int

	if status != nil {
		if err := r.db.SelectContext(
			ctx,
			&rows,
			sqlSelectGroupImagesByGroupAndStatus,
			groupID.String(),
			status.String(),
			pagination.Limit,
			pagination.Offset,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to find group images by group and status: %w", err)
		}
		if err := r.db.GetContext(ctx, &total, sqlCountGroupImagesByGroupAndStatus, groupID.String(), status.String()); err != nil {
			return nil, 0, fmt.Errorf("failed to count group images by group and status: %w", err)
		}
	} else {
		if err := r.db.SelectContext(
			ctx,
			&rows,
			sqlSelectGroupImagesByGroup,
			groupID.String(),
			pagination.Limit,
			pagination.Offset,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to find group images by group: %w", err)
		}
		if err := r.db.GetContext(ctx, &total, sqlCountGroupImagesByGroup, groupID.String()); err != nil {
			return nil, 0, fmt.Errorf("failed to count group images by group: %w", err)
		}
	}

	images := make([]*community.GroupImage, 0, len(rows))
	for _, row := range rows {
		image, err := rowToGroupImage(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to group image: %w", err)
		}
		images = append(images, image)
	}

	return images, total, nil
}

// FindPendingByGroup retrieves all pending images for a group (moderation queue).
func (r *GroupImageRepository) FindPendingByGroup(
	ctx context.Context,
	groupID community.GroupID,
	pagination shared.Pagination,
) ([]*community.GroupImage, int, error) {
	var rows []groupImageRow
	if err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectPendingGroupImagesByGroup,
		groupID.String(),
		pagination.Limit,
		pagination.Offset,
	); err != nil {
		return nil, 0, fmt.Errorf("failed to find pending group images: %w", err)
	}

	images := make([]*community.GroupImage, 0, len(rows))
	for _, row := range rows {
		image, err := rowToGroupImage(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to group image: %w", err)
		}
		images = append(images, image)
	}

	var total int
	status := community.GroupImageStatusPending
	if err := r.db.GetContext(ctx, &total, sqlCountGroupImagesByGroupAndStatus, groupID.String(), status.String()); err != nil {
		return nil, 0, fmt.Errorf("failed to count pending group images: %w", err)
	}

	return images, total, nil
}

// FindApprovedByGroup retrieves all approved images for a group.
func (r *GroupImageRepository) FindApprovedByGroup(
	ctx context.Context,
	groupID community.GroupID,
	pagination shared.Pagination,
) ([]*community.GroupImage, int, error) {
	var rows []groupImageRow
	if err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectApprovedGroupImagesByGroup,
		groupID.String(),
		pagination.Limit,
		pagination.Offset,
	); err != nil {
		return nil, 0, fmt.Errorf("failed to find approved group images: %w", err)
	}

	images := make([]*community.GroupImage, 0, len(rows))
	for _, row := range rows {
		image, err := rowToGroupImage(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to group image: %w", err)
		}
		images = append(images, image)
	}

	var total int
	status := community.GroupImageStatusApproved
	if err := r.db.GetContext(ctx, &total, sqlCountGroupImagesByGroupAndStatus, groupID.String(), status.String()); err != nil {
		return nil, 0, fmt.Errorf("failed to count approved group images: %w", err)
	}

	return images, total, nil
}

// Delete removes the group image from storage.
func (r *GroupImageRepository) Delete(ctx context.Context, id community.GroupImageID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteGroupImage, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete group image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return community.ErrGroupImageNotFound
	}

	return nil
}

// ExistsByGroupAndImage checks if an image is already shared to the group.
func (r *GroupImageRepository) ExistsByGroupAndImage(
	ctx context.Context,
	groupID community.GroupID,
	imageID gallery.ImageID,
) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, sqlExistsGroupImageByGroupAndImage, groupID.String(), imageID.String()); err != nil {
		return false, fmt.Errorf("failed to check group image existence: %w", err)
	}
	return exists, nil
}

// rowToGroupImage converts a database row to a GroupImage domain entity.
func rowToGroupImage(row groupImageRow) (*community.GroupImage, error) {
	groupImageID, err := community.ParseGroupImageID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse group image id: %w", err)
	}

	groupID, err := community.ParseGroupID(row.GroupID)
	if err != nil {
		return nil, fmt.Errorf("parse group id: %w", err)
	}

	imageID, err := gallery.ParseImageID(row.ImageID)
	if err != nil {
		return nil, fmt.Errorf("parse image id: %w", err)
	}

	sharedBy, err := identity.ParseUserID(row.SharedBy)
	if err != nil {
		return nil, fmt.Errorf("parse shared by user id: %w", err)
	}

	status := community.GroupImageStatus(row.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid group image status: %s", row.Status)
	}

	var reviewedBy *identity.UserID
	if row.ReviewedBy.Valid {
		parsedID, err := identity.ParseUserID(row.ReviewedBy.String)
		if err != nil {
			return nil, fmt.Errorf("parse reviewed by user id: %w", err)
		}
		reviewedBy = &parsedID
	}

	var reviewedAt *time.Time
	if row.ReviewedAt.Valid {
		reviewedAt = &row.ReviewedAt.Time
	}

	return community.ReconstructGroupImage(
		groupImageID,
		groupID,
		imageID,
		sharedBy,
		status,
		reviewedBy,
		row.SharedAt,
		reviewedAt,
	), nil
}
