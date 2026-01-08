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
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for report operations.
const (
	sqlInsertReport = `
		INSERT INTO reports (
			id, reporter_id, image_id, reason, description, status,
			resolved_by, resolved_at, resolution, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	sqlUpdateReport = `
		UPDATE reports
		SET status = $2,
		    resolved_by = $3,
		    resolved_at = $4,
		    resolution = $5
		WHERE id = $1
	`

	sqlSelectReportByID = `
		SELECT id, reporter_id, image_id, reason, description, status, resolved_by, resolved_at, resolution, created_at
		FROM reports
		WHERE id = $1
	`

	sqlSelectPendingReports = `
		SELECT id, reporter_id, image_id, reason, description, status, resolved_by, resolved_at, resolution, created_at
		FROM reports
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	sqlCountPendingReports = `
		SELECT COUNT(*)
		FROM reports
		WHERE status = 'pending'
	`

	sqlSelectReportsByImage = `
		SELECT id, reporter_id, image_id, reason, description, status, resolved_by, resolved_at, resolution, created_at
		FROM reports
		WHERE image_id = $1
		ORDER BY created_at DESC
	`

	sqlSelectReportsByReporter = `
		SELECT id, reporter_id, image_id, reason, description, status, resolved_by, resolved_at, resolution, created_at
		FROM reports
		WHERE reporter_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountReportsByReporter = `
		SELECT COUNT(*)
		FROM reports
		WHERE reporter_id = $1
	`
)

// reportRow represents a report row in the database.
type reportRow struct {
	ID          string         `db:"id"`
	ReporterID  string         `db:"reporter_id"`
	ImageID     string         `db:"image_id"`
	Reason      string         `db:"reason"`
	Description string         `db:"description"`
	Status      string         `db:"status"`
	ResolvedBy  sql.NullString `db:"resolved_by"`
	ResolvedAt  sql.NullTime   `db:"resolved_at"`
	Resolution  string         `db:"resolution"`
	CreatedAt   time.Time      `db:"created_at"`
}

// ReportRepository implements the moderation.ReportRepository interface for PostgreSQL.
type ReportRepository struct {
	db *sqlx.DB
}

// NewReportRepository creates a new ReportRepository with the given database connection.
func NewReportRepository(db *sqlx.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// NextID generates the next available ReportID.
func (r *ReportRepository) NextID() moderation.ReportID {
	return moderation.NewReportID()
}

// FindByID retrieves a report by its unique ID.
func (r *ReportRepository) FindByID(ctx context.Context, id moderation.ReportID) (*moderation.Report, error) {
	var row reportRow
	if err := r.db.GetContext(ctx, &row, sqlSelectReportByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, moderation.ErrReportNotFound
		}
		return nil, fmt.Errorf("failed to find report by id: %w", err)
	}

	report, err := rowToReport(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to report: %w", err)
	}

	return report, nil
}

// FindPending retrieves all reports with pending status.
//
//nolint:dupl // Standard repository pagination pattern
func (r *ReportRepository) FindPending(
	ctx context.Context,
	pagination shared.Pagination,
) ([]*moderation.Report, int64, error) {
	// Get paginated reports
	var rows []reportRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectPendingReports,
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pending reports: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountPendingReports)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pending reports: %w", err)
	}

	// Convert rows to domain entities
	reports := make([]*moderation.Report, 0, len(rows))
	for _, row := range rows {
		report, err := rowToReport(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to report: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, total, nil
}

// FindByImage retrieves all reports for a specific image.
func (r *ReportRepository) FindByImage(
	ctx context.Context,
	imageID gallery.ImageID,
) ([]*moderation.Report, error) {
	var rows []reportRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectReportsByImage, imageID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find reports by image: %w", err)
	}

	// Convert rows to domain entities
	reports := make([]*moderation.Report, 0, len(rows))
	for _, row := range rows {
		report, err := rowToReport(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to report: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// FindByReporter retrieves all reports submitted by a specific user.
//
//nolint:dupl // Similar pattern to FindByReviewer but operates on different types
func (r *ReportRepository) FindByReporter(
	ctx context.Context,
	reporterID identity.UserID,
	pagination shared.Pagination,
) ([]*moderation.Report, int64, error) {
	// Get paginated reports
	var rows []reportRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectReportsByReporter,
		reporterID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reports by reporter: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountReportsByReporter, reporterID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reports by reporter: %w", err)
	}

	// Convert rows to domain entities
	reports := make([]*moderation.Report, 0, len(rows))
	for _, row := range rows {
		report, err := rowToReport(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to report: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, total, nil
}

// Save persists a report to the repository.
// If the report already exists, it is updated; otherwise, it is created.
func (r *ReportRepository) Save(ctx context.Context, report *moderation.Report) error {
	// Check if report exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM reports WHERE id = $1)", report.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check report existence: %w", err)
	}

	if exists {
		return r.update(ctx, report)
	}
	return r.insert(ctx, report)
}

// insert creates a new report in the database.
func (r *ReportRepository) insert(ctx context.Context, report *moderation.Report) error {
	var resolvedBy *string
	if report.ResolvedBy() != nil {
		val := report.ResolvedBy().String()
		resolvedBy = &val
	}

	_, err := r.db.ExecContext(
		ctx,
		sqlInsertReport,
		report.ID().String(),
		report.ReporterID().String(),
		report.ImageID().String(),
		report.Reason().String(),
		report.Description(),
		report.Status().String(),
		resolvedBy,
		report.ResolvedAt(),
		report.Resolution(),
		report.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert report: %w", err)
	}

	return nil
}

// update updates an existing report in the database.
func (r *ReportRepository) update(ctx context.Context, report *moderation.Report) error {
	var resolvedBy *string
	if report.ResolvedBy() != nil {
		val := report.ResolvedBy().String()
		resolvedBy = &val
	}

	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateReport,
		report.ID().String(),
		report.Status().String(),
		resolvedBy,
		report.ResolvedAt(),
		report.Resolution(),
	)
	if err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return moderation.ErrReportNotFound
	}

	return nil
}

// rowToReport converts a database row to a domain Report entity.
func rowToReport(row reportRow) (*moderation.Report, error) {
	// Parse IDs
	reportID, err := moderation.ParseReportID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	reporterID, err := identity.ParseUserID(row.ReporterID)
	if err != nil {
		return nil, fmt.Errorf("invalid reporter id: %w", err)
	}

	imageID, err := gallery.ParseImageID(row.ImageID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// Parse enums
	reason, err := moderation.ParseReportReason(row.Reason)
	if err != nil {
		return nil, fmt.Errorf("invalid reason: %w", err)
	}

	status, err := moderation.ParseReportStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Parse nullable fields
	var resolvedBy *identity.UserID
	if row.ResolvedBy.Valid {
		id, err := identity.ParseUserID(row.ResolvedBy.String)
		if err != nil {
			return nil, fmt.Errorf("invalid resolved by id: %w", err)
		}
		resolvedBy = &id
	}

	var resolvedAt *time.Time
	if row.ResolvedAt.Valid {
		resolvedAt = &row.ResolvedAt.Time
	}

	// Reconstitute report without validation or events
	report := moderation.ReconstructReport(
		reportID,
		reporterID,
		imageID,
		reason,
		row.Description,
		status,
		resolvedBy,
		resolvedAt,
		row.Resolution,
		row.CreatedAt,
	)

	return report, nil
}
