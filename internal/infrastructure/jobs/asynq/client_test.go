package asynq_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/jobs/asynq"
)

func setupTestRedis(t *testing.T) (string, func()) {
	t.Helper()

	// Create mini-redis server
	mr := miniredis.RunT(t)

	cleanup := func() {
		mr.Close()
	}

	return mr.Addr(), cleanup
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  asynq.ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: asynq.ClientConfig{
				RedisAddr: "localhost:6379",
				Logger:    zerolog.Nop(),
			},
			wantErr: false,
		},
		{
			name: "missing redis address",
			config: asynq.ClientConfig{
				Logger: zerolog.Nop(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := asynq.NewClient(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					_ = client.Close()
				}
			}
		})
	}
}

func TestClient_EnqueueTask(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynq.ClientConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	client, err := asynq.NewClient(cfg)
	require.NoError(t, err)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()

	t.Run("enqueue simple task", func(t *testing.T) {
		payload := map[string]string{"key": "value"}

		err := client.EnqueueTask(ctx, "test:task", payload)
		assert.NoError(t, err)
	})

	t.Run("enqueue task with invalid payload", func(t *testing.T) {
		// Channel cannot be marshaled to JSON
		payload := make(chan int)

		err := client.EnqueueTask(ctx, "test:task", payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "marshal task payload")
	})
}

func TestClient_EnqueueTaskWithDelay(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynq.ClientConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	client, err := asynq.NewClient(cfg)
	require.NoError(t, err)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	payload := map[string]string{"delayed": "true"}
	delay := 5 * time.Minute

	err = client.EnqueueTaskWithDelay(ctx, "test:delayed", payload, delay)
	assert.NoError(t, err)
}

func TestClient_EnqueueTaskAt(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynq.ClientConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	client, err := asynq.NewClient(cfg)
	require.NoError(t, err)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	payload := map[string]string{"scheduled": "true"}
	processAt := time.Now().Add(1 * time.Hour)

	err = client.EnqueueTaskAt(ctx, "test:scheduled", payload, processAt)
	assert.NoError(t, err)
}

func TestClient_Close(t *testing.T) {
	redisAddr, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zerolog.Nop()
	cfg := asynq.ClientConfig{
		RedisAddr: redisAddr,
		Logger:    logger,
	}

	client, err := asynq.NewClient(cfg)
	require.NoError(t, err)

	// Close should succeed
	err = client.Close()
	assert.NoError(t, err)

	// Double close should not panic
	err = client.Close()
	assert.Error(t, err) // Asynq returns error on double close
}
