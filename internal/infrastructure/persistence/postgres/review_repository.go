package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SQL queries for review operations.
const (
	sqlInsertReview = `
		INSERT INTO reviews (id, report_id, reviewer_id, action, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	sqlSelectReviewByID = `
		SELECT id, report_id, reviewer_id, action, notes, created_at
		FROM reviews
		WHERE id = $1
	`

	sqlSelectReviewsByReport = `
		SELECT id, report_id, reviewer_id, action, notes, created_at
		FROM reviews
		WHERE report_id = $1
		ORDER BY created_at ASC
	`

	sqlSelectReviewsByReviewer = `
		SELECT id, report_id, reviewer_id, action, notes, created_at
		FROM reviews
		WHERE reviewer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlCountReviewsByReviewer = `
		SELECT COUNT(*)
		FROM reviews
		WHERE reviewer_id = $1
	`
)

// reviewRow represents a review row in the database.
type reviewRow struct {
	ID         string    `db:"id"`
	ReportID   string    `db:"report_id"`
	ReviewerID string    `db:"reviewer_id"`
	Action     string    `db:"action"`
	Notes      string    `db:"notes"`
	CreatedAt  time.Time `db:"created_at"`
}

// ReviewRepository implements the moderation.ReviewRepository interface for PostgreSQL.
type ReviewRepository struct {
	db *sqlx.DB
}

// NewReviewRepository creates a new ReviewRepository with the given database connection.
func NewReviewRepository(db *sqlx.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// NextID generates the next available ReviewID.
func (r *ReviewRepository) NextID() moderation.ReviewID {
	return moderation.NewReviewID()
}

// FindByID retrieves a review by its unique ID.
func (r *ReviewRepository) FindByID(ctx context.Context, id moderation.ReviewID) (*moderation.Review, error) {
	var row reviewRow
	if err := r.db.GetContext(ctx, &row, sqlSelectReviewByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, moderation.ErrReviewNotFound
		}
		return nil, fmt.Errorf("failed to find review by id: %w", err)
	}

	review, err := rowToReview(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to review: %w", err)
	}

	return review, nil
}

// FindByReport retrieves all reviews for a specific report.
func (r *ReviewRepository) FindByReport(
	ctx context.Context,
	reportID moderation.ReportID,
) ([]*moderation.Review, error) {
	var rows []reviewRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectReviewsByReport, reportID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find reviews by report: %w", err)
	}

	// Convert rows to domain entities
	reviews := make([]*moderation.Review, 0, len(rows))
	for _, row := range rows {
		review, err := rowToReview(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to review: %w", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}

// FindByReviewer retrieves all reviews performed by a specific moderator.
//
//nolint:dupl // Similar pattern to FindByReporter but operates on different types
func (r *ReviewRepository) FindByReviewer(
	ctx context.Context,
	reviewerID identity.UserID,
	pagination shared.Pagination,
) ([]*moderation.Review, int64, error) {
	// Get paginated reviews
	var rows []reviewRow
	err := r.db.SelectContext(
		ctx,
		&rows,
		sqlSelectReviewsByReviewer,
		reviewerID.String(),
		pagination.Limit(),
		pagination.Offset(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reviews by reviewer: %w", err)
	}

	// Get total count
	var total int64
	err = r.db.GetContext(ctx, &total, sqlCountReviewsByReviewer, reviewerID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews by reviewer: %w", err)
	}

	// Convert rows to domain entities
	reviews := make([]*moderation.Review, 0, len(rows))
	for _, row := range rows {
		review, err := rowToReview(row)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert row to review: %w", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, total, nil
}

// Save persists a review to the repository.
// Reviews are immutable, so this only handles creation.
func (r *ReviewRepository) Save(ctx context.Context, review *moderation.Review) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertReview,
		review.ID().String(),
		review.ReportID().String(),
		review.ReviewerID().String(),
		review.Action().String(),
		review.Notes(),
		review.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert review: %w", err)
	}

	return nil
}

// rowToReview converts a database row to a domain Review entity.
func rowToReview(row reviewRow) (*moderation.Review, error) {
	// Parse IDs
	reviewID, err := moderation.ParseReviewID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid review id: %w", err)
	}

	reportID, err := moderation.ParseReportID(row.ReportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	reviewerID, err := identity.ParseUserID(row.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("invalid reviewer id: %w", err)
	}

	// Parse enum
	action, err := moderation.ParseReviewAction(row.Action)
	if err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}

	// Reconstitute review without validation
	review := moderation.ReconstructReview(
		reviewID,
		reportID,
		reviewerID,
		action,
		row.Notes,
		row.CreatedAt,
	)

	return review, nil
}
