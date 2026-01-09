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

// SQL queries for variant config operations.
const (
	sqlInsertVariantConfig = `
		INSERT INTO variant_configs (
			id, user_id, name, max_width, max_height, format, quality,
			crop_mode, is_preset, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	sqlUpdateVariantConfig = `
		UPDATE variant_configs
		SET max_width = $2,
		    max_height = $3,
		    format = $4,
		    quality = $5,
		    crop_mode = $6,
		    updated_at = $7
		WHERE id = $1
	`

	sqlSelectVariantConfigByID = `
		SELECT id, user_id, name, max_width, max_height, format, quality,
		       crop_mode, is_preset, created_at, updated_at
		FROM variant_configs
		WHERE id = $1
	`

	sqlSelectVariantConfigsByUser = `
		SELECT id, user_id, name, max_width, max_height, format, quality,
		       crop_mode, is_preset, created_at, updated_at
		FROM variant_configs
		WHERE user_id = $1
		ORDER BY name ASC
	`

	sqlSelectVariantConfigByUserAndName = `
		SELECT id, user_id, name, max_width, max_height, format, quality,
		       crop_mode, is_preset, created_at, updated_at
		FROM variant_configs
		WHERE user_id = $1 AND name = $2
	`

	sqlSelectVariantConfigPresets = `
		SELECT id, user_id, name, max_width, max_height, format, quality,
		       crop_mode, is_preset, created_at, updated_at
		FROM variant_configs
		WHERE is_preset = true
		ORDER BY name ASC
	`

	sqlDeleteVariantConfig = `
		DELETE FROM variant_configs WHERE id = $1
	`

	sqlExistsVariantConfig = `
		SELECT EXISTS(SELECT 1 FROM variant_configs WHERE id = $1)
	`
)

// variantConfigRow represents a variant config row in the database.
type variantConfigRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Name      string    `db:"name"`
	MaxWidth  int       `db:"max_width"`
	MaxHeight int       `db:"max_height"`
	Format    string    `db:"format"`
	Quality   int       `db:"quality"`
	CropMode  string    `db:"crop_mode"`
	IsPreset  bool      `db:"is_preset"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// VariantConfigRepository implements the gallery.VariantConfigRepository interface for PostgreSQL.
type VariantConfigRepository struct {
	db *sqlx.DB
}

// NewVariantConfigRepository creates a new VariantConfigRepository with the given database connection.
func NewVariantConfigRepository(db *sqlx.DB) *VariantConfigRepository {
	return &VariantConfigRepository{db: db}
}

// NextID generates the next available VariantConfigID.
func (r *VariantConfigRepository) NextID() gallery.VariantConfigID {
	return gallery.NewVariantConfigID()
}

// FindByID retrieves a variant config by its unique ID.
func (r *VariantConfigRepository) FindByID(ctx context.Context, id gallery.VariantConfigID) (*gallery.VariantConfig, error) {
	var row variantConfigRow
	if err := r.db.GetContext(ctx, &row, sqlSelectVariantConfigByID, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrVariantConfigNotFound
		}
		return nil, fmt.Errorf("failed to find variant config by id: %w", err)
	}

	config, err := rowToVariantConfig(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to variant config: %w", err)
	}

	return config, nil
}

// FindByUser retrieves all variant configs for a user.
func (r *VariantConfigRepository) FindByUser(ctx context.Context, userID identity.UserID) ([]*gallery.VariantConfig, error) {
	var rows []variantConfigRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectVariantConfigsByUser, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find variant configs by user: %w", err)
	}

	configs := make([]*gallery.VariantConfig, 0, len(rows))
	for _, row := range rows {
		config, err := rowToVariantConfig(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to variant config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// FindByUserAndName retrieves a specific variant config by user and name.
func (r *VariantConfigRepository) FindByUserAndName(
	ctx context.Context,
	userID identity.UserID,
	name string,
) (*gallery.VariantConfig, error) {
	var row variantConfigRow
	err := r.db.GetContext(ctx, &row, sqlSelectVariantConfigByUserAndName, userID.String(), name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gallery.ErrVariantConfigNotFound
		}
		return nil, fmt.Errorf("failed to find variant config by user and name: %w", err)
	}

	config, err := rowToVariantConfig(row)
	if err != nil {
		return nil, fmt.Errorf("failed to convert row to variant config: %w", err)
	}

	return config, nil
}

// FindPresets retrieves all system-defined variant presets.
func (r *VariantConfigRepository) FindPresets(ctx context.Context) ([]*gallery.VariantConfig, error) {
	var rows []variantConfigRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectVariantConfigPresets)
	if err != nil {
		return nil, fmt.Errorf("failed to find variant config presets: %w", err)
	}

	configs := make([]*gallery.VariantConfig, 0, len(rows))
	for _, row := range rows {
		config, err := rowToVariantConfig(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to variant config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// Save persists a variant config to the repository.
// If the config already exists, it is updated; otherwise, it is created.
func (r *VariantConfigRepository) Save(ctx context.Context, config *gallery.VariantConfig) error {
	// Check if config exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsVariantConfig, config.ID().String())
	if err != nil {
		return fmt.Errorf("failed to check variant config existence: %w", err)
	}

	if exists {
		return r.update(ctx, config)
	}
	return r.insert(ctx, config)
}

// Delete permanently removes a variant config from the repository.
func (r *VariantConfigRepository) Delete(ctx context.Context, id gallery.VariantConfigID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteVariantConfig, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete variant config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrVariantConfigNotFound
	}

	return nil
}

// ExistsByID checks if a variant config exists.
func (r *VariantConfigRepository) ExistsByID(ctx context.Context, id gallery.VariantConfigID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlExistsVariantConfig, id.String())
	if err != nil {
		return false, fmt.Errorf("failed to check variant config existence: %w", err)
	}
	return exists, nil
}

// insert creates a new variant config in the database.
func (r *VariantConfigRepository) insert(ctx context.Context, config *gallery.VariantConfig) error {
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertVariantConfig,
		config.ID().String(),
		config.UserID().String(),
		config.Name(),
		config.MaxWidth(),
		config.MaxHeight(),
		string(config.Format()),
		config.Quality(),
		string(config.CropMode()),
		config.IsPreset(),
		config.CreatedAt(),
		config.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert variant config: %w", err)
	}

	return nil
}

// update updates an existing variant config in the database.
func (r *VariantConfigRepository) update(ctx context.Context, config *gallery.VariantConfig) error {
	result, err := r.db.ExecContext(
		ctx,
		sqlUpdateVariantConfig,
		config.ID().String(),
		config.MaxWidth(),
		config.MaxHeight(),
		string(config.Format()),
		config.Quality(),
		string(config.CropMode()),
		config.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to update variant config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return gallery.ErrVariantConfigNotFound
	}

	return nil
}

// rowToVariantConfig converts a database row to a domain VariantConfig entity.
func rowToVariantConfig(row variantConfigRow) (*gallery.VariantConfig, error) {
	// Parse IDs
	configID, err := gallery.ParseVariantConfigID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid variant config id: %w", err)
	}

	userID, err := identity.ParseUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// Parse output format
	format := gallery.OutputFormat(row.Format)
	if !format.IsValid() {
		return nil, fmt.Errorf("invalid output format: %s", row.Format)
	}

	// Parse crop mode
	cropMode := gallery.CropMode(row.CropMode)
	if !cropMode.IsValid() {
		return nil, fmt.Errorf("invalid crop mode: %s", row.CropMode)
	}

	// Reconstitute variant config without validation or events
	config := gallery.ReconstructVariantConfig(
		configID,
		userID,
		row.Name,
		row.MaxWidth,
		row.MaxHeight,
		format,
		row.Quality,
		cropMode,
		row.IsPreset,
		row.CreatedAt,
		row.UpdatedAt,
	)

	return config, nil
}
