// Package postgres implements PostgreSQL persistence for the Identity bounded context.
// It provides repository implementations and database connection management using sqlx.
package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Default connection pool configuration constants.
const (
	defaultPort            = 5432
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 10 * time.Minute
	defaultPingTimeout     = 5 * time.Second
)

// Config holds the PostgreSQL connection configuration.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns a Config with secure defaults.
// NOTE: User and Password are intentionally left empty to enforce explicit configuration.
// In production, use environment variables or a secrets manager for credentials.
// SSLMode defaults to "require" for security - only disable for local development.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            defaultPort,
		User:            "", // Must be explicitly set
		Password:        "", // Must be explicitly set
		Database:        "goimg",
		SSLMode:         "require", // Secure default; use "disable" only for local dev
		MaxOpenConns:    defaultMaxOpenConns,
		MaxIdleConns:    defaultMaxIdleConns,
		ConnMaxLifetime: defaultConnMaxLifetime,
		ConnMaxIdleTime: defaultConnMaxIdleTime,
	}
}

// ConfigFromEnv creates a Config populated from environment variables.
// Environment variables:
//   - DB_HOST (default: localhost)
//   - DB_PORT (default: 5432)
//   - DB_USER (required)
//   - DB_PASSWORD (required)
//   - DB_NAME (default: goimg)
//   - DB_SSL_MODE (default: require)
//   - DB_MAX_OPEN_CONNS (default: 25)
//   - DB_MAX_IDLE_CONNS (default: 5)
//   - DB_CONN_MAX_LIFETIME_MINUTES (default: 30)
//   - DB_CONN_MAX_IDLE_TIME_MINUTES (default: 10)
func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if v := getEnv("DB_HOST", ""); v != "" {
		cfg.Host = v
	}
	if v := getEnvInt("DB_PORT", 0); v != 0 {
		cfg.Port = v
	}
	cfg.User = getEnv("DB_USER", cfg.User)
	cfg.Password = getEnv("DB_PASSWORD", cfg.Password)
	if v := getEnv("DB_NAME", ""); v != "" {
		cfg.Database = v
	}
	if v := getEnv("DB_SSL_MODE", ""); v != "" {
		cfg.SSLMode = v
	}
	if v := getEnvInt("DB_MAX_OPEN_CONNS", 0); v != 0 {
		cfg.MaxOpenConns = v
	}
	if v := getEnvInt("DB_MAX_IDLE_CONNS", 0); v != 0 {
		cfg.MaxIdleConns = v
	}
	if v := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 0); v != 0 {
		cfg.ConnMaxLifetime = time.Duration(v) * time.Minute
	}
	if v := getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 0); v != 0 {
		cfg.ConnMaxIdleTime = time.Duration(v) * time.Minute
	}
	return cfg
}

// NewDB creates a new PostgreSQL connection pool with the given configuration.
// It configures the pool settings and verifies connectivity.
func NewDB(cfg Config) (*sqlx.DB, error) {
	// Build PostgreSQL connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Open database connection
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// HealthCheck verifies the database connection is healthy.
// Returns an error if the database is unreachable or unhealthy.
func HealthCheck(ctx context.Context, db *sqlx.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Ping with context timeout
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Verify we can execute a simple query
	var result int
	if err := db.GetContext(ctx, &result, "SELECT 1"); err != nil {
		return fmt.Errorf("database query check failed: %w", err)
	}

	return nil
}

// Close gracefully closes the database connection pool.
func Close(db *sqlx.DB) error {
	if db == nil {
		return nil
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
