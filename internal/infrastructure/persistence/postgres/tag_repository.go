package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// SQL queries for tag operations.
const (
	sqlFindTagBySlug = `
		SELECT id, name, slug, usage_count, created_at
		FROM tags
		WHERE slug = $1
	`

	sqlFindTagByName = `
		SELECT id, name, slug, usage_count, created_at
		FROM tags
		WHERE LOWER(name) = LOWER($1)
	`

	sqlFindPopularTags = `
		SELECT id, name, slug, usage_count, created_at
		FROM tags
		WHERE usage_count > 0
		ORDER BY usage_count DESC
		LIMIT $1
	`

	// Popular tags by period - counts tags from image_tags in time window
	sqlFindPopularTagsByPeriod = `
		SELECT t.id, t.name, t.slug, t.usage_count, t.created_at,
		       COUNT(it.tag_id) as period_count
		FROM tags t
		INNER JOIN image_tags it ON t.id = it.tag_id
		WHERE it.tagged_at >= $2
		GROUP BY t.id, t.name, t.slug, t.usage_count, t.created_at
		ORDER BY period_count DESC
		LIMIT $1
	`

	// Trending tags with time-weighted popularity
	// Formula: (recent_count / total_count) * time_decay_factor
	sqlFindTrendingTags = `
		WITH tag_stats AS (
			SELECT t.id, t.name, t.slug, t.usage_count, t.created_at,
			       COUNT(it.tag_id) as recent_count
			FROM tags t
			LEFT JOIN image_tags it ON t.id = it.tag_id AND it.tagged_at >= $2
			WHERE t.usage_count > 0
			GROUP BY t.id, t.name, t.slug, t.usage_count, t.created_at
		)
		SELECT id, name, slug, usage_count, created_at,
		       CASE WHEN usage_count > 0
		            THEN (recent_count::float / usage_count::float) * 100
		            ELSE 0
		       END as trend_score
		FROM tag_stats
		WHERE recent_count > 0
		ORDER BY trend_score DESC, recent_count DESC
		LIMIT $1
	`

	sqlSearchTagsByPrefix = `
		SELECT id, name, slug, usage_count, created_at
		FROM tags
		WHERE slug LIKE $1 || '%' OR LOWER(name) LIKE LOWER($1) || '%'
		ORDER BY usage_count DESC, name ASC
		LIMIT $2
	`

	sqlUpsertTag = `
		INSERT INTO tags (id, name, slug, usage_count, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			usage_count = EXCLUDED.usage_count
		RETURNING id
	`

	sqlIncrementTagUsage = `
		UPDATE tags
		SET usage_count = usage_count + 1
		WHERE slug = $1
	`

	sqlDecrementTagUsage = `
		UPDATE tags
		SET usage_count = GREATEST(0, usage_count - 1)
		WHERE slug = $1
	`

	sqlCheckTagExistsBySlug = `
		SELECT EXISTS(SELECT 1 FROM tags WHERE slug = $1)
	`
)

// tagRow represents a row from the tags table.
type tagRow struct {
	ID         uuid.UUID `db:"id"`
	Name       string    `db:"name"`
	Slug       string    `db:"slug"`
	UsageCount int64     `db:"usage_count"`
	CreatedAt  time.Time `db:"created_at"`
}

// tagWithScoreRow represents a tag row with trending score.
type tagWithScoreRow struct {
	tagRow
	TrendScore   float64 `db:"trend_score"`
	PeriodCount  int64   `db:"period_count"`
}

// TagRepository implements tag operations for PostgreSQL.
type TagRepository struct {
	db *sqlx.DB
}

// NewTagRepository creates a new TagRepository with the given database connection.
func NewTagRepository(db *sqlx.DB) *TagRepository {
	return &TagRepository{db: db}
}

// FindBySlug retrieves a tag by its URL-friendly slug.
func (r *TagRepository) FindBySlug(ctx context.Context, slug string) (*gallery.Tag, error) {
	var row tagRow
	err := r.db.GetContext(ctx, &row, sqlFindTagBySlug, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find tag by slug %s: %w", slug, gallery.ErrTagNotFound)
		}
		return nil, fmt.Errorf("find tag by slug: %w", err)
	}

	tag, err := gallery.NewTag(row.Name)
	if err != nil {
		return nil, fmt.Errorf("reconstruct tag: %w", err)
	}

	return &tag, nil
}

// FindByName retrieves a tag by its display name.
func (r *TagRepository) FindByName(ctx context.Context, name string) (*gallery.Tag, error) {
	var row tagRow
	err := r.db.GetContext(ctx, &row, sqlFindTagByName, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find tag by name %s: %w", name, gallery.ErrTagNotFound)
		}
		return nil, fmt.Errorf("find tag by name: %w", err)
	}

	tag, err := gallery.NewTag(row.Name)
	if err != nil {
		return nil, fmt.Errorf("reconstruct tag: %w", err)
	}

	return &tag, nil
}

// FindPopular retrieves the most popular tags by usage count.
func (r *TagRepository) FindPopular(ctx context.Context, limit int, period gallery.TagPeriod) ([]gallery.TagWithUsage, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var rows []tagRow
	var err error

	if period == gallery.TagPeriodAll {
		// For "all time", use simple query sorted by usage_count
		err = r.db.SelectContext(ctx, &rows, sqlFindPopularTags, limit)
	} else {
		// For specific period, count tags from that time window
		since := r.periodToTime(period)
		err = r.db.SelectContext(ctx, &rows, sqlFindPopularTagsByPeriod, limit, since)
	}

	if err != nil {
		return nil, fmt.Errorf("find popular tags: %w", err)
	}

	return r.rowsToTagsWithUsage(rows)
}

// FindTrending retrieves trending tags with time-weighted popularity.
func (r *TagRepository) FindTrending(ctx context.Context, limit int, period gallery.TagPeriod) ([]gallery.TagWithUsage, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	since := r.periodToTime(period)

	var rows []tagWithScoreRow
	err := r.db.SelectContext(ctx, &rows, sqlFindTrendingTags, limit, since)
	if err != nil {
		return nil, fmt.Errorf("find trending tags: %w", err)
	}

	return r.scoreRowsToTagsWithUsage(rows)
}

// SearchByPrefix finds tags matching a name prefix for autocomplete.
func (r *TagRepository) SearchByPrefix(ctx context.Context, query string, limit int) ([]gallery.TagWithUsage, error) {
	if len(query) < 2 {
		return []gallery.TagWithUsage{}, nil
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	var rows []tagRow
	err := r.db.SelectContext(ctx, &rows, sqlSearchTagsByPrefix, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search tags by prefix: %w", err)
	}

	return r.rowsToTagsWithUsage(rows)
}

// Save persists a tag (insert or update).
func (r *TagRepository) Save(ctx context.Context, tag gallery.Tag) error {
	id := uuid.New()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, sqlUpsertTag, id, tag.Name(), tag.Slug(), 0, now)
	if err != nil {
		return fmt.Errorf("save tag: %w", err)
	}

	return nil
}

// GetOrCreate retrieves a tag by name or creates it if it doesn't exist.
func (r *TagRepository) GetOrCreate(ctx context.Context, name string) (*gallery.Tag, error) {
	// First try to find existing tag
	tag, err := r.FindByName(ctx, name)
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, gallery.ErrTagNotFound) {
		return nil, fmt.Errorf("check existing tag: %w", err)
	}

	// Create new tag
	newTag, err := gallery.NewTag(name)
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	if err := r.Save(ctx, newTag); err != nil {
		return nil, fmt.Errorf("save new tag: %w", err)
	}

	return &newTag, nil
}

// IncrementUsage increments the usage count for a tag by 1.
func (r *TagRepository) IncrementUsage(ctx context.Context, tagSlug string) error {
	result, err := r.db.ExecContext(ctx, sqlIncrementTagUsage, tagSlug)
	if err != nil {
		return fmt.Errorf("increment tag usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("increment usage for tag %s: %w", tagSlug, gallery.ErrTagNotFound)
	}

	return nil
}

// DecrementUsage decrements the usage count for a tag by 1.
func (r *TagRepository) DecrementUsage(ctx context.Context, tagSlug string) error {
	result, err := r.db.ExecContext(ctx, sqlDecrementTagUsage, tagSlug)
	if err != nil {
		return fmt.Errorf("decrement tag usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("decrement usage for tag %s: %w", tagSlug, gallery.ErrTagNotFound)
	}

	return nil
}

// ExistsBySlug checks if a tag exists by slug.
func (r *TagRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckTagExistsBySlug, slug)
	if err != nil {
		return false, fmt.Errorf("check tag exists: %w", err)
	}
	return exists, nil
}

// periodToTime converts a TagPeriod to a time.Time for SQL queries.
func (r *TagRepository) periodToTime(period gallery.TagPeriod) time.Time {
	now := time.Now().UTC()
	switch period {
	case gallery.TagPeriodDay:
		return now.AddDate(0, 0, -1)
	case gallery.TagPeriodWeek:
		return now.AddDate(0, 0, -7)
	case gallery.TagPeriodMonth:
		return now.AddDate(0, -1, 0)
	default:
		// For "all" or unknown, use a very old date
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
}

// rowsToTagsWithUsage converts database rows to domain objects.
func (r *TagRepository) rowsToTagsWithUsage(rows []tagRow) ([]gallery.TagWithUsage, error) {
	result := make([]gallery.TagWithUsage, 0, len(rows))

	for _, row := range rows {
		tag, err := gallery.NewTag(row.Name)
		if err != nil {
			// Skip invalid tags (shouldn't happen but be defensive)
			continue
		}

		result = append(result, gallery.TagWithUsage{
			Tag:        tag,
			UsageCount: row.UsageCount,
			TrendScore: 0,
		})
	}

	return result, nil
}

// scoreRowsToTagsWithUsage converts database rows with scores to domain objects.
func (r *TagRepository) scoreRowsToTagsWithUsage(rows []tagWithScoreRow) ([]gallery.TagWithUsage, error) {
	result := make([]gallery.TagWithUsage, 0, len(rows))

	for _, row := range rows {
		tag, err := gallery.NewTag(row.Name)
		if err != nil {
			// Skip invalid tags (shouldn't happen but be defensive)
			continue
		}

		result = append(result, gallery.TagWithUsage{
			Tag:        tag,
			UsageCount: row.UsageCount,
			TrendScore: row.TrendScore,
		})
	}

	return result, nil
}

// Ensure TagRepository implements the gallery.TagRepository interface.
var _ gallery.TagRepository = (*TagRepository)(nil)
