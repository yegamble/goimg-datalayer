package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	sqlInsertFeaturedPick = `
		INSERT INTO featured_picks (
			id, image_id, featured_by, reason, display_order,
			featured_from, featured_until, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	sqlUpdateFeaturedPick = `
		UPDATE featured_picks
		SET display_order = $2,
			featured_from = $3,
			featured_until = $4,
			updated_at = $5
		WHERE id = $1`

	sqlFindFeaturedPickByID = `
		SELECT id, image_id, featured_by, reason, display_order,
			   featured_from, featured_until, created_at, updated_at
		FROM featured_picks
		WHERE id = $1`

	sqlFindFeaturedPickByImageID = `
		SELECT id, image_id, featured_by, reason, display_order,
			   featured_from, featured_until, created_at, updated_at
		FROM featured_picks
		WHERE image_id = $1
		  AND featured_from <= NOW()
		  AND (featured_until IS NULL OR featured_until > NOW())`

	sqlListActiveFeaturedPicks = `
		SELECT id, image_id, featured_by, reason, display_order,
			   featured_from, featured_until, created_at, updated_at
		FROM featured_picks
		WHERE featured_from <= NOW()
		  AND (featured_until IS NULL OR featured_until > NOW())
		ORDER BY display_order ASC, featured_from DESC
		LIMIT $1`

	sqlListAllFeaturedPicks = `
		SELECT id, image_id, featured_by, reason, display_order,
			   featured_from, featured_until, created_at, updated_at
		FROM featured_picks
		WHERE ($1 = true OR featured_until IS NULL OR featured_until > NOW())
		ORDER BY featured_from DESC
		OFFSET $2 LIMIT $3`

	sqlCountAllFeaturedPicks = `
		SELECT COUNT(*)
		FROM featured_picks
		WHERE ($1 = true OR featured_until IS NULL OR featured_until > NOW())`

	sqlDeleteFeaturedPick = `
		DELETE FROM featured_picks
		WHERE id = $1`

	sqlExistsFeaturedPickByImageID = `
		SELECT EXISTS(
			SELECT 1
			FROM featured_picks
			WHERE image_id = $1
			  AND featured_from <= NOW()
			  AND (featured_until IS NULL OR featured_until > NOW())
		)`
)

type featuredPickRow struct {
	ID            string       `db:"id"`
	ImageID       string       `db:"image_id"`
	FeaturedBy    string       `db:"featured_by"`
	Reason        string       `db:"reason"`
	DisplayOrder  int          `db:"display_order"`
	FeaturedFrom  time.Time    `db:"featured_from"`
	FeaturedUntil sql.NullTime `db:"featured_until"`
	CreatedAt     time.Time    `db:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at"`
}

// FeaturedPickRepository implements gallery.FeaturedPickRepository using PostgreSQL.
type FeaturedPickRepository struct {
	db *sqlx.DB
}

// NewFeaturedPickRepository creates a new PostgreSQL-backed FeaturedPickRepository.
func NewFeaturedPickRepository(db *sqlx.DB) *FeaturedPickRepository {
	return &FeaturedPickRepository{db: db}
}

// FindByID retrieves a featured pick by its ID.
func (r *FeaturedPickRepository) FindByID(ctx context.Context, id gallery.FeaturedPickID) (*gallery.FeaturedPick, error) {
	var row featuredPickRow
	err := r.db.GetContext(ctx, &row, sqlFindFeaturedPickByID, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrFeaturedPickNotFound
		}
		return nil, fmt.Errorf("query featured pick by id: %w", err)
	}

	return r.rowToDomain(row)
}

// FindByImageID retrieves the active featured pick for an image.
func (r *FeaturedPickRepository) FindByImageID(ctx context.Context, imageID gallery.ImageID) (*gallery.FeaturedPick, error) {
	var row featuredPickRow
	err := r.db.GetContext(ctx, &row, sqlFindFeaturedPickByImageID, imageID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrFeaturedPickNotFound
		}
		return nil, fmt.Errorf("query featured pick by image id: %w", err)
	}

	return r.rowToDomain(row)
}

// ListActive retrieves currently active featured picks.
func (r *FeaturedPickRepository) ListActive(ctx context.Context, limit int) ([]*gallery.FeaturedPick, error) {
	var rows []featuredPickRow
	err := r.db.SelectContext(ctx, &rows, sqlListActiveFeaturedPicks, limit)
	if err != nil {
		return nil, fmt.Errorf("query active featured picks: %w", err)
	}

	picks := make([]*gallery.FeaturedPick, 0, len(rows))
	for _, row := range rows {
		pick, err := r.rowToDomain(row)
		if err != nil {
			return nil, err
		}
		picks = append(picks, pick)
	}

	return picks, nil
}

// ListAll retrieves all featured picks with pagination.
func (r *FeaturedPickRepository) ListAll(
	ctx context.Context,
	includeExpired bool,
	offset, limit int,
) ([]*gallery.FeaturedPick, int, error) {
	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, sqlCountAllFeaturedPicks, includeExpired)
	if err != nil {
		return nil, 0, fmt.Errorf("count featured picks: %w", err)
	}

	// Get paginated results
	var rows []featuredPickRow
	err = r.db.SelectContext(ctx, &rows, sqlListAllFeaturedPicks, includeExpired, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query all featured picks: %w", err)
	}

	picks := make([]*gallery.FeaturedPick, 0, len(rows))
	for _, row := range rows {
		pick, err := r.rowToDomain(row)
		if err != nil {
			return nil, 0, err
		}
		picks = append(picks, pick)
	}

	return picks, total, nil
}

// Save persists a featured pick (insert or update).
func (r *FeaturedPickRepository) Save(ctx context.Context, pick *gallery.FeaturedPick) error {
	// Check if exists
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM featured_picks WHERE id = $1)",
		pick.ID().String())
	if err != nil {
		return fmt.Errorf("check featured pick existence: %w", err)
	}

	var featuredUntil sql.NullTime
	if pick.FeaturedUntil() != nil {
		featuredUntil = sql.NullTime{Time: *pick.FeaturedUntil(), Valid: true}
	}

	if !exists {
		// Insert
		_, err = r.db.ExecContext(ctx, sqlInsertFeaturedPick,
			pick.ID().String(),
			pick.ImageID().String(),
			pick.FeaturedBy().String(),
			pick.Reason(),
			pick.DisplayOrder(),
			pick.FeaturedFrom(),
			featuredUntil,
			pick.CreatedAt(),
			pick.UpdatedAt(),
		)
		if err != nil {
			return fmt.Errorf("insert featured pick: %w", err)
		}
	} else {
		// Update
		_, err = r.db.ExecContext(ctx, sqlUpdateFeaturedPick,
			pick.ID().String(),
			pick.DisplayOrder(),
			pick.FeaturedFrom(),
			featuredUntil,
			pick.UpdatedAt(),
		)
		if err != nil {
			return fmt.Errorf("update featured pick: %w", err)
		}
	}

	return nil
}

// Delete removes a featured pick.
func (r *FeaturedPickRepository) Delete(ctx context.Context, id gallery.FeaturedPickID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteFeaturedPick, id.String())
	if err != nil {
		return fmt.Errorf("delete featured pick: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}

	if rows == 0 {
		return gallery.ErrFeaturedPickNotFound
	}

	return nil
}

// ExistsByImageID checks if an image is currently featured.
func (r *FeaturedPickRepository) ExistsByImageID(ctx context.Context, imageID gallery.ImageID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsFeaturedPickByImageID, imageID.String())
	if err != nil {
		return false, fmt.Errorf("check if image is featured: %w", err)
	}
	return exists, nil
}

// rowToDomain converts a database row to a domain FeaturedPick.
func (r *FeaturedPickRepository) rowToDomain(row featuredPickRow) (*gallery.FeaturedPick, error) {
	id, err := gallery.ParseFeaturedPickID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse featured pick id: %w", err)
	}

	imageID, err := gallery.ParseImageID(row.ImageID)
	if err != nil {
		return nil, fmt.Errorf("parse image id: %w", err)
	}

	featuredBy, err := identity.ParseUserID(row.FeaturedBy)
	if err != nil {
		return nil, fmt.Errorf("parse featured_by user id: %w", err)
	}

	var featuredUntil *time.Time
	if row.FeaturedUntil.Valid {
		featuredUntil = &row.FeaturedUntil.Time
	}

	return gallery.ReconstructFeaturedPick(
		id,
		imageID,
		featuredBy,
		row.Reason,
		row.DisplayOrder,
		row.FeaturedFrom,
		featuredUntil,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}
