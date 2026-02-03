package postgres

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "goimg", cfg.Database)
	assert.Equal(t, "require", cfg.SSLMode)
	assert.Equal(t, 25, cfg.MaxOpenConns)
	assert.Equal(t, 25, cfg.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
}

func TestConfigFromEnv(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("DB_HOST")
	defer os.Setenv("DB_HOST", originalHost)
	originalMaxIdle := os.Getenv("DB_MAX_IDLE_CONNS")
	defer os.Setenv("DB_MAX_IDLE_CONNS", originalMaxIdle)

	// Set test env vars
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("DB_MAX_IDLE_CONNS", "50")

	cfg := ConfigFromEnv()

	assert.Equal(t, "test-host", cfg.Host)
	assert.Equal(t, 50, cfg.MaxIdleConns)
}
