// Package containers provides testcontainer setup for integration tests.
package containers

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer represents a running PostgreSQL testcontainer.
type PostgresContainer struct {
	Container testcontainers.Container
	DB        *sqlx.DB
	ConnStr   string
}

// NewPostgresContainer creates and starts a PostgreSQL 16 testcontainer.
// It automatically runs migrations from the migrations directory.
func NewPostgresContainer(ctx context.Context, t *testing.T) (*PostgresContainer, error) {
	t.Helper()

	// Get project root (3 levels up from tests/integration/containers)
	projectRoot, err := filepath.Abs("../../..")
	require.NoError(t, err)
	migrationsPath := filepath.Join(projectRoot, "migrations")

	// Start PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Connect using sqlx
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		_ = postgresContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Run migrations using goose
	if err := runMigrations(db.DB, migrationsPath); err != nil {
		_ = db.Close()
		_ = postgresContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &PostgresContainer{
		Container: postgresContainer,
		DB:        db,
		ConnStr:   connStr,
	}, nil
}

// runMigrations runs goose migrations from the specified directory.
func runMigrations(db *sql.DB, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// Cleanup truncates all tables and resets sequences.
// This is faster than recreating the container for each test.
func (pc *PostgresContainer) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()

	// Truncate tables in reverse dependency order (respecting foreign keys)
	tables := []string{
		"image_tags",
		"tags",
		"album_images",
		"image_variants",
		"albums",
		"images",
		"sessions",
		"users",
	}
	for _, table := range tables {
		_, err := pc.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "failed to truncate table %s", table)
	}
}

// Terminate stops and removes the container.
func (pc *PostgresContainer) Terminate(ctx context.Context) error {
	if pc.DB != nil {
		_ = pc.DB.Close()
	}
	if pc.Container != nil {
		if err := pc.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("terminate postgres container: %w", err)
		}
	}
	return nil
}

// WaitForHealthy waits for the database to be ready.
func (pc *PostgresContainer) WaitForHealthy(ctx context.Context, t *testing.T) {
	t.Helper()

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if err := pc.DB.PingContext(ctx); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.Fail(t, "postgres did not become healthy within timeout")
}
