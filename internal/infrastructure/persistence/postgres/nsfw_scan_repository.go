package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	sqlInsertNSFWScan = `
		INSERT INTO nsfw_scans (
			id, image_id, provider, status, category, score,
			nudity_score, weapon_score, violence_score, offensive_score, drug_score,
			sub_categories, error_message, scanned_at, created_at, updated_at
		) VALUES (
			:id, :image_id, :provider, :status, :category, :score,
			:nudity_score, :weapon_score, :violence_score, :offensive_score, :drug_score,
			:sub_categories, :error_message, :scanned_at, :created_at, :updated_at
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			category = EXCLUDED.category,
			score = EXCLUDED.score,
			nudity_score = EXCLUDED.nudity_score,
			weapon_score = EXCLUDED.weapon_score,
			violence_score = EXCLUDED.violence_score,
			offensive_score = EXCLUDED.offensive_score,
			drug_score = EXCLUDED.drug_score,
			sub_categories = EXCLUDED.sub_categories,
			error_message = EXCLUDED.error_message,
			scanned_at = EXCLUDED.scanned_at,
			updated_at = EXCLUDED.updated_at
	`

	sqlSelectNSFWScanByID = `
		SELECT * FROM nsfw_scans WHERE id = $1
	`

	sqlSelectNSFWScanByImageID = `
		SELECT * FROM nsfw_scans WHERE image_id = $1 ORDER BY created_at DESC LIMIT 1
	`

	sqlSelectAllNSFWScansByImageID = `
		SELECT * FROM nsfw_scans WHERE image_id = $1 ORDER BY created_at DESC
	`

	sqlSelectPendingNSFWScans = `
		SELECT * FROM nsfw_scans
		WHERE status IN ('pending', 'scanning')
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`

	sqlCountPendingNSFWScans = `
		SELECT COUNT(*) FROM nsfw_scans WHERE status IN ('pending', 'scanning')
	`

	sqlSelectNSFWScansByStatus = `
		SELECT * FROM nsfw_scans
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	sqlCountNSFWScansByStatus = `
		SELECT COUNT(*) FROM nsfw_scans WHERE status = $1
	`

	sqlSelectNSFWImages = `
		SELECT * FROM nsfw_scans
		WHERE category IN ('nudity', 'explicit', 'violence')
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	sqlCountNSFWImages = `
		SELECT COUNT(*) FROM nsfw_scans
		WHERE category IN ('nudity', 'explicit', 'violence')
	`

	sqlHasActiveScan = `
		SELECT EXISTS(
			SELECT 1 FROM nsfw_scans
			WHERE image_id = $1 AND status IN ('pending', 'scanning')
		)
	`
)

type nsfwScanRow struct {
	ID             string         `db:"id"`
	ImageID        string         `db:"image_id"`
	Provider       string         `db:"provider"`
	Status         string         `db:"status"`
	Category       string         `db:"category"`
	Score          float64        `db:"score"`
	NudityScore    float64        `db:"nudity_score"`
	WeaponScore    float64        `db:"weapon_score"`
	ViolenceScore  float64        `db:"violence_score"`
	OffensiveScore float64        `db:"offensive_score"`
	DrugScore      float64        `db:"drug_score"`
	SubCategories  pq.StringArray `db:"sub_categories"`
	ErrorMessage   sql.NullString `db:"error_message"`
	ScannedAt      sql.NullTime   `db:"scanned_at"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

func (r *nsfwScanRow) toDomain() (*moderation.NSFWScan, error) {
	id, err := moderation.ParseNSFWScanID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}

	imageID, err := gallery.ParseImageID(r.ImageID)
	if err != nil {
		return nil, fmt.Errorf("parse image id: %w", err)
	}

	provider := moderation.NSFWProvider(r.Provider)
	status := moderation.NSFWScanStatus(r.Status)
	category := moderation.NSFWCategory(r.Category)

	details := moderation.NewNSFWDetails(
		r.NudityScore,
		r.WeaponScore,
		r.ViolenceScore,
		r.OffensiveScore,
		r.DrugScore,
	).WithSubCategories(r.SubCategories)

	return moderation.ReconstructNSFWScan(
		id,
		imageID,
		provider,
		status,
		category,
		r.Score,
		details,
		r.ErrorMessage.String,
		r.ScannedAt.Time,
		r.CreatedAt,
		r.UpdatedAt,
	), nil
}

func fromDomain(scan *moderation.NSFWScan) *nsfwScanRow {
	details := scan.Details()

	row := &nsfwScanRow{
		ID:             scan.ID().String(),
		ImageID:        scan.ImageID().String(),
		Provider:       string(scan.Provider()),
		Status:         string(scan.Status()),
		Category:       string(scan.Category()),
		Score:          scan.Score(),
		NudityScore:    details.NudityScore,
		WeaponScore:    details.WeaponScore,
		ViolenceScore:  details.ViolenceScore,
		OffensiveScore: details.OffensiveScore,
		DrugScore:      details.DrugScore,
		SubCategories:  pq.StringArray(details.SubCategories),
		CreatedAt:      scan.CreatedAt(),
		UpdatedAt:      scan.UpdatedAt(),
	}

	if scan.ErrorMessage() != "" {
		row.ErrorMessage = sql.NullString{String: scan.ErrorMessage(), Valid: true}
	}

	if !scan.ScannedAt().IsZero() {
		row.ScannedAt = sql.NullTime{Time: scan.ScannedAt(), Valid: true}
	}

	return row
}

type NSFWScanRepository struct {
	db *sqlx.DB
}

func NewNSFWScanRepository(db *sqlx.DB) *NSFWScanRepository {
	return &NSFWScanRepository{db: db}
}

func (r *NSFWScanRepository) NextID() moderation.NSFWScanID {
	return moderation.NewNSFWScanID()
}

func (r *NSFWScanRepository) FindByID(ctx context.Context, id moderation.NSFWScanID) (*moderation.NSFWScan, error) {
	var row nsfwScanRow
	err := r.db.GetContext(ctx, &row, sqlSelectNSFWScanByID, id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, moderation.ErrNSFWScanNotFound
		}
		return nil, fmt.Errorf("find nsfw scan by id: %w", err)
	}

	return row.toDomain()
}

func (r *NSFWScanRepository) FindByImageID(ctx context.Context, imageID gallery.ImageID) (*moderation.NSFWScan, error) {
	var row nsfwScanRow
	err := r.db.GetContext(ctx, &row, sqlSelectNSFWScanByImageID, imageID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, moderation.ErrNSFWScanNotFound
		}
		return nil, fmt.Errorf("find nsfw scan by image id: %w", err)
	}

	return row.toDomain()
}

func (r *NSFWScanRepository) FindByImageIDAll(ctx context.Context, imageID gallery.ImageID) ([]*moderation.NSFWScan, error) {
	var rows []nsfwScanRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectAllNSFWScansByImageID, imageID.String())
	if err != nil {
		return nil, fmt.Errorf("find all nsfw scans by image id: %w", err)
	}

	scans := make([]*moderation.NSFWScan, 0, len(rows))
	for _, row := range rows {
		scan, err := row.toDomain()
		if err != nil {
			return nil, fmt.Errorf("convert row to domain: %w", err)
		}
		scans = append(scans, scan)
	}

	return scans, nil
}

func (r *NSFWScanRepository) FindPending(ctx context.Context, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	var rows []nsfwScanRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectPendingNSFWScans, pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("find pending nsfw scans: %w", err)
	}

	scans := make([]*moderation.NSFWScan, 0, len(rows))
	for _, row := range rows {
		scan, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("convert row to domain: %w", err)
		}
		scans = append(scans, scan)
	}

	var count int64
	err = r.db.GetContext(ctx, &count, sqlCountPendingNSFWScans)
	if err != nil {
		return nil, 0, fmt.Errorf("count pending nsfw scans: %w", err)
	}

	return scans, count, nil
}

func (r *NSFWScanRepository) FindByStatus(ctx context.Context, status moderation.NSFWScanStatus, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	var rows []nsfwScanRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectNSFWScansByStatus, string(status), pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("find nsfw scans by status: %w", err)
	}

	scans := make([]*moderation.NSFWScan, 0, len(rows))
	for _, row := range rows {
		scan, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("convert row to domain: %w", err)
		}
		scans = append(scans, scan)
	}

	var count int64
	err = r.db.GetContext(ctx, &count, sqlCountNSFWScansByStatus, string(status))
	if err != nil {
		return nil, 0, fmt.Errorf("count nsfw scans by status: %w", err)
	}

	return scans, count, nil
}

func (r *NSFWScanRepository) FindNSFWImages(ctx context.Context, pagination shared.Pagination) ([]*moderation.NSFWScan, int64, error) {
	var rows []nsfwScanRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectNSFWImages, pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("find nsfw images: %w", err)
	}

	scans := make([]*moderation.NSFWScan, 0, len(rows))
	for _, row := range rows {
		scan, err := row.toDomain()
		if err != nil {
			return nil, 0, fmt.Errorf("convert row to domain: %w", err)
		}
		scans = append(scans, scan)
	}

	var count int64
	err = r.db.GetContext(ctx, &count, sqlCountNSFWImages)
	if err != nil {
		return nil, 0, fmt.Errorf("count nsfw images: %w", err)
	}

	return scans, count, nil
}

func (r *NSFWScanRepository) HasActiveScan(ctx context.Context, imageID gallery.ImageID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlHasActiveScan, imageID.String())
	if err != nil {
		return false, fmt.Errorf("check active scan: %w", err)
	}
	return exists, nil
}

func (r *NSFWScanRepository) Save(ctx context.Context, scan *moderation.NSFWScan) error {
	row := fromDomain(scan)

	_, err := r.db.NamedExecContext(ctx, sqlInsertNSFWScan, row)
	if err != nil {
		return fmt.Errorf("save nsfw scan: %w", err)
	}

	return nil
}
