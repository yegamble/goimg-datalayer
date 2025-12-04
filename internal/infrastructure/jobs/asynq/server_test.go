package asynq_test

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	asynqpkg "github.com/yegamble/goimg-datalayer/internal/infrastructure/jobs/asynq"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	tests := []struct {
		name    string
		config  asynqpkg.ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: asynqpkg.ServerConfig{
				RedisAddr:       "localhost:6379",
				Concurrency:     5,
				Queues:          map[string]int{"default": 1},
				Logger:          logger,
				ShutdownTimeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing redis address",
			config: asynqpkg.ServerConfig{
				Logger: logger,
			},
			wantErr: true,
		},
		{
			name: "default config applied",
			config: asynqpkg.ServerConfig{
				RedisAddr: "localhost:6379",
				Logger:    logger,
				// No concurrency, queues, or timeout specified
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, err := asynqpkg.NewServer(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestDefaultServerConfig(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	cfg := asynqpkg.DefaultServerConfig("localhost:6379", logger)

	assert.Equal(t, "localhost:6379", cfg.RedisAddr)
	assert.Equal(t, 10, cfg.Concurrency)
	assert.Equal(t, map[string]int{"default": 1}, cfg.Queues)
	assert.False(t, cfg.StrictPriority)
	assert.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
}

func TestServer_RegisterHandler(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynqpkg.ServerConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	server, err := asynqpkg.NewServer(cfg)
	require.NoError(t, err)

	// Register a simple handler
	handlerCalled := false
	handler := asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		handlerCalled = true
		return nil
	})

	// Should not panic
	server.RegisterHandler("test:task", handler)
	assert.False(t, handlerCalled) // Not called until task is processed
}

func TestServer_RegisterHandlerFunc(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynqpkg.ServerConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	server, err := asynqpkg.NewServer(cfg)
	require.NoError(t, err)

	// Register a simple handler function
	handlerCalled := false
	handlerFunc := func(ctx context.Context, task *asynq.Task) error {
		handlerCalled = true
		return nil
	}

	// Should not panic
	server.RegisterHandlerFunc("test:task", handlerFunc)
	assert.False(t, handlerCalled) // Not called until task is processed
}

func TestServer_Shutdown(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynqpkg.ServerConfig{
		RedisAddr:       redisAddr,
		Logger:          logger,
		ShutdownTimeout: 1 * time.Second,
	}

	server, err := asynqpkg.NewServer(cfg)
	require.NoError(t, err)

	// Shutdown should not panic (even without starting)
	server.Shutdown()
}
