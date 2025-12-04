# Background Jobs Infrastructure Guide

> Async task processing with Asynq (Redis-backed) for image processing and malware scanning.

## Overview

This package implements background job processing using [Asynq](https://github.com/hibiken/asynq), a distributed task queue backed by Redis. It provides reliable, asynchronous task execution with features like retries, scheduling, prioritization, and monitoring.

## Architecture

```
internal/infrastructure/jobs/
├── asynq/
│   ├── client.go       # Task enqueuing (producer)
│   └── server.go       # Task processing (worker)
├── tasks/
│   ├── image_process.go   # Image variant generation
│   └── image_scan.go      # Malware scanning with ClamAV
└── CLAUDE.md           # This guide
```

### Component Roles

| Component | Role | Used By |
|-----------|------|---------|
| `asynq.Client` | Enqueues tasks into Redis | HTTP handlers (upload endpoint) |
| `asynq.Server` | Processes tasks from Redis | Background worker process |
| `ImageProcessHandler` | Generates image variants via bimg | Asynq server |
| `ImageScanHandler` | Scans images via ClamAV | Asynq server |

## Task Types

### 1. Image Processing (`image:process`)

**Purpose**: Generate image variants (thumbnail, small, medium, large) using bimg/libvips.

**Payload**:
```go
type ImageProcessPayload struct {
    ImageID          string    `json:"image_id"`
    StorageKey       string    `json:"storage_key"`
    OriginalFilename string    `json:"original_filename"`
    OwnerID          string    `json:"owner_id"`
    EnqueuedAt       time.Time `json:"enqueued_at"`
}
```

**Processing Steps**:
1. Retrieve original image from storage
2. Process image through bimg pipeline (resize, strip EXIF, re-encode)
3. Store all variants (thumbnail, small, medium, large, original)

**Configuration**:
- **Max Retries**: 3
- **Timeout**: 5 minutes
- **Queue**: `default` (priority 1)

**Failure Scenarios**:
- Storage retrieval failure → retry
- Image processing error (corrupt file) → terminal failure
- Storage upload failure → retry

### 2. Malware Scanning (`image:scan`)

**Purpose**: Scan images for malware using ClamAV daemon.

**Payload**:
```go
type ImageScanPayload struct {
    ImageID          string    `json:"image_id"`
    StorageKey       string    `json:"storage_key"`
    OriginalFilename string    `json:"original_filename"`
    OwnerID          string    `json:"owner_id"`
    EnqueuedAt       time.Time `json:"enqueued_at"`
}
```

**Processing Steps**:
1. Ping ClamAV daemon to verify connectivity
2. Retrieve image from storage
3. Scan via ClamAV TCP socket
4. Update image status based on scan result

**Configuration**:
- **Max Retries**: 2
- **Timeout**: 2 minutes
- **Queue**: `default` (priority 1)

**Scan Results**:
- **Clean**: Image is safe, update status to "clean"
- **Infected**: Malware detected, mark image as infected, notify user, delete file

**Failure Scenarios**:
- ClamAV daemon unavailable → retry
- Storage retrieval failure → retry
- Malware detected → terminal failure (no retry)

## Usage Patterns

### Enqueuing Tasks (HTTP Handler)

```go
import (
    "github.com/yegamble/goimg-datalayer/internal/infrastructure/jobs/asynq"
    "github.com/yegamble/goimg-datalayer/internal/infrastructure/jobs/tasks"
)

// In upload handler
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
    // ... validate and store original image ...

    // Enqueue image processing task
    processPayload := tasks.ImageProcessPayload{
        ImageID:          imageID.String(),
        StorageKey:       storageKey,
        OriginalFilename: filename,
        OwnerID:          userID.String(),
        EnqueuedAt:       time.Now(),
    }

    task, err := tasks.NewImageProcessTask(processPayload)
    if err != nil {
        return fmt.Errorf("create process task: %w", err)
    }

    if err := h.jobClient.EnqueueTask(ctx, tasks.TypeImageProcess, processPayload); err != nil {
        h.logger.Error().Err(err).Msg("failed to enqueue processing task")
        // Continue - processing can be retried later
    }

    // Enqueue malware scan task
    scanPayload := tasks.ImageScanPayload{
        ImageID:          imageID.String(),
        StorageKey:       storageKey,
        OriginalFilename: filename,
        OwnerID:          userID.String(),
        EnqueuedAt:       time.Now(),
    }

    if err := h.jobClient.EnqueueTask(ctx, tasks.TypeImageScan, scanPayload); err != nil {
        h.logger.Error().Err(err).Msg("failed to enqueue scan task")
        // Continue - scan can be retried later
    }

    // Return 202 Accepted - processing continues in background
    w.WriteHeader(http.StatusAccepted)
}
```

### Processing Tasks (Worker)

```go
// In worker/main.go
func main() {
    logger := zerolog.New(os.Stdout)

    // Create storage client
    storage := local.NewStorage(config)

    // Create image processor
    processor, err := processor.New(processorConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Create ClamAV scanner
    clamavClient, err := clamav.NewClient(clamav.DefaultConfig())
    if err != nil {
        log.Fatal(err)
    }

    // Create Asynq server
    serverConfig := asynq.DefaultServerConfig("localhost:6379", logger)
    serverConfig.Concurrency = 10
    serverConfig.Queues = map[string]int{
        "critical": 6,
        "default":  3,
        "low":      1,
    }

    server, err := asynq.NewServer(serverConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Register task handlers
    processHandler := tasks.NewImageProcessHandler(processor, storage, logger)
    server.RegisterHandlerFunc(tasks.TypeImageProcess, processHandler.ProcessTask)

    scanHandler := tasks.NewImageScanHandler(clamavClient, storage, logger)
    server.RegisterHandlerFunc(tasks.TypeImageScan, scanHandler.ProcessTask)

    // Start server (blocking)
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Queue Configuration

### Queue Priorities

Asynq supports multiple queues with different priorities:

```go
Queues: map[string]int{
    "critical": 6,  // Highest priority (e.g., user-facing operations)
    "default":  3,  // Normal priority (image processing)
    "low":      1,  // Background tasks (cleanup, analytics)
}
```

**Priority Calculation**: Tasks are dequeued with probability proportional to queue weight.

Example: With the above configuration:
- 60% of dequeue operations target `critical` queue
- 30% target `default` queue
- 10% target `low` queue

**Strict Priority Mode**: If `StrictPriority: true`, higher priority queues are always processed first (no round-robin).

### Concurrency

Controls the maximum number of tasks processed simultaneously:

```go
Concurrency: 10  // Process up to 10 tasks in parallel
```

**Tuning Guidelines**:
- **CPU-bound tasks** (image processing): Set concurrency ≈ number of CPU cores
- **I/O-bound tasks** (API calls, DB queries): Set concurrency higher (2-4x cores)
- **Mixed workload**: Start with 10, adjust based on metrics

## Retry Strategy

### Default Behavior

Asynq provides exponential backoff with jitter by default:

```
Retry 1: ~15 seconds
Retry 2: ~30 seconds
Retry 3: ~1 minute
Retry 4: ~2 minutes
...
```

### Custom Retry Logic

```go
// In server configuration
RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
    // n = retry count (0-indexed)
    // e = error from last attempt
    // t = task being retried

    // Example: Linear backoff
    return time.Duration(n) * 30 * time.Second

    // Example: Exponential with max
    delay := time.Duration(1<<uint(n)) * time.Second
    if delay > 10*time.Minute {
        delay = 10 * time.Minute
    }
    return delay
},
```

### Per-Task Retry Configuration

```go
// Set max retries per task
task := asynq.NewTask(
    tasks.TypeImageProcess,
    payload,
    asynq.MaxRetry(5),  // Override default
)
```

### Non-Retryable Errors

To mark an error as terminal (no retries):

```go
if malwareDetected {
    // Return SkipRetry error to prevent retries
    return fmt.Errorf("malware detected: %w", asynq.SkipRetry)
}
```

## Timeout Handling

### Task Timeouts

Each task has a deadline to prevent indefinite execution:

```go
// Default timeout
asynq.NewTask(type, payload, asynq.Timeout(5*time.Minute))

// Check context in handler
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    select {
    case <-ctx.Done():
        return fmt.Errorf("task cancelled: %w", ctx.Err())
    default:
        // Process task
    }
}
```

### Server Shutdown Timeout

Controls how long the server waits for in-flight tasks to complete:

```go
ShutdownTimeout: 30 * time.Second
```

**Graceful Shutdown**:
1. Stop accepting new tasks
2. Wait up to `ShutdownTimeout` for in-flight tasks
3. Force-kill remaining tasks

## Error Handling

### Error Handler

Asynq calls a custom error handler when a task fails after all retries:

```go
ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
    logger.Error().
        Err(err).
        Str("task_type", task.Type()).
        Str("task_id", task.ResultWriter().TaskID()).
        Int("retry_count", task.ResultWriter().MaxRetry()).
        Msg("task failed after all retries")

    // Send alert, record metric, etc.
}),
```

### Handler Error Return

**Retryable Error**: Return normal error

```go
return fmt.Errorf("temporary failure: %w", err)
```

**Terminal Error**: Use `asynq.SkipRetry`

```go
return fmt.Errorf("malware detected: %w", asynq.SkipRetry)
```

## Monitoring and Observability

### Built-in Metrics

Asynq provides metrics via Prometheus exporter (requires asynq-mon):

```go
import "github.com/hibiken/asynq/x/metrics"

// Expose Prometheus metrics
mux := http.NewServeMux()
mux.Handle("/metrics", metrics.Handler())
http.ListenAndServe(":8080", mux)
```

**Metrics Exposed**:
- `asynq_tasks_processed_total` - Total tasks processed (by queue, status)
- `asynq_tasks_failed_total` - Failed tasks
- `asynq_tasks_enqueued_total` - Enqueued tasks
- `asynq_queue_size` - Current queue size
- `asynq_processing_duration_seconds` - Task processing time histogram

### Logging

All task handlers should log:
- Task start (image_id, payload)
- Task completion (duration, result)
- Errors (with full context)

```go
logger.Info().
    Str("image_id", payload.ImageID).
    Dur("duration_ms", duration).
    Msg("image processing completed")
```

### Health Checks

Worker health check endpoint:

```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    // Check Redis connection
    if err := jobClient.Ping(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
```

## Testing

### Unit Tests

Test task handlers in isolation:

```go
func TestImageProcessHandler_ProcessTask(t *testing.T) {
    // Create mocks
    mockStorage := &MockStorage{}
    mockProcessor := &MockProcessor{}
    logger := zerolog.Nop()

    handler := tasks.NewImageProcessHandler(mockProcessor, mockStorage, logger)

    // Create test payload
    payload := tasks.ImageProcessPayload{
        ImageID:    "test-image-id",
        StorageKey: "images/test.jpg",
    }
    payloadBytes, _ := json.Marshal(payload)
    task := asynq.NewTask(tasks.TypeImageProcess, payloadBytes)

    // Setup mocks
    mockStorage.On("Get", mock.Anything, "images/test.jpg").
        Return(testImageData, nil)
    mockProcessor.On("Process", mock.Anything, testImageData, mock.Anything).
        Return(&processor.ProcessResult{...}, nil)
    mockStorage.On("Put", mock.Anything, mock.Anything, mock.Anything).
        Return(nil)

    // Execute
    err := handler.ProcessTask(context.Background(), task)

    // Assert
    assert.NoError(t, err)
    mockStorage.AssertExpectations(t)
    mockProcessor.AssertExpectations(t)
}
```

### Integration Tests

Test with real Asynq server and Redis:

```go
func TestImageProcessing_EndToEnd(t *testing.T) {
    // Start Redis container
    redisContainer := testcontainers.StartRedis(t)
    defer redisContainer.Terminate()

    // Create client and server
    client := asynq.NewClient(redisOpt)
    server := asynq.NewServer(redisOpt, config)

    // Register handler
    handler := tasks.NewImageProcessHandler(...)
    server.RegisterHandlerFunc(tasks.TypeImageProcess, handler.ProcessTask)

    // Start server in background
    go server.Start()
    defer server.Shutdown()

    // Enqueue task
    task, _ := tasks.NewImageProcessTask(payload)
    client.EnqueueContext(ctx, task)

    // Wait for completion and verify
    // ...
}
```

## Performance Tuning

### Concurrency Tuning

**Symptoms of Too Low Concurrency**:
- Tasks pile up in queue
- High queue latency
- Low CPU/memory utilization

**Symptoms of Too High Concurrency**:
- Memory pressure (OOM kills)
- CPU saturation
- Redis connection pool exhaustion

**Recommendation**: Monitor queue size and processing time, adjust concurrency accordingly.

### Redis Connection Pooling

Asynq uses Redis connection pooling internally. For high throughput:

```go
// In Redis configuration
PoolSize: 20  // Max connections per worker instance
```

### Task Batching

For high-volume tasks, consider batching:

```go
// Instead of 1000 individual tasks
// Create 10 batch tasks of 100 items each
type BatchProcessPayload struct {
    ImageIDs []string `json:"image_ids"`
}
```

## Deployment Considerations

### Separate Worker Process

Run workers separately from API servers:

```
docker-compose.yml:
  api:
    image: goimg-api
    command: ["./api"]

  worker:
    image: goimg-api
    command: ["./worker"]
    environment:
      - WORKER_CONCURRENCY=20
      - ASYNQ_REDIS_ADDR=redis:6379
```

### Scaling Workers

**Horizontal Scaling**: Run multiple worker instances
- Each instance processes tasks from the same queue
- Redis ensures tasks are distributed (no duplicates)

**Vertical Scaling**: Increase concurrency per worker
- More CPU/memory per instance
- Better for CPU-bound tasks

### Redis High Availability

For production, use Redis Sentinel or Redis Cluster:

```go
redisOpt := asynq.RedisFailoverClientOpt{
    MasterName:    "mymaster",
    SentinelAddrs: []string{"sentinel1:26379", "sentinel2:26379"},
}
```

### Resource Limits

Set resource limits to prevent resource exhaustion:

```yaml
# docker-compose.yml
worker:
  deploy:
    resources:
      limits:
        cpus: '4'
        memory: 8G
      reservations:
        cpus: '2'
        memory: 4G
```

## Troubleshooting

### Tasks Not Processing

1. Check Redis connectivity: `redis-cli -h localhost -p 6379 PING`
2. Verify worker is running: `ps aux | grep worker`
3. Check queue size: `redis-cli -h localhost -p 6379 LLEN asynq:queues:default`
4. Review worker logs for errors

### High Task Failure Rate

1. Check error logs for common patterns
2. Verify external dependencies (ClamAV, storage)
3. Review retry configuration
4. Monitor resource usage (OOM, CPU throttling)

### Slow Task Processing

1. Check task processing time histogram
2. Profile handlers for bottlenecks
3. Verify I/O latency (storage, network)
4. Consider increasing concurrency or adding workers

## Security Considerations

1. **Redis Authentication**: Always enable `requirepass` in production
2. **Redis TLS**: Use TLS for Redis connections over untrusted networks
3. **Payload Validation**: Validate all task payloads before processing
4. **Resource Limits**: Set timeouts and memory limits to prevent DoS
5. **Error Logging**: Avoid logging sensitive data in error messages

## References

- [Asynq Documentation](https://github.com/hibiken/asynq)
- [Asynq Dashboard (asynqmon)](https://github.com/hibiken/asynqmon)
- [Redis Best Practices](https://redis.io/docs/management/optimization/)
- [Image Processing Guide](../../storage/processor/CLAUDE.md)
- [ClamAV Integration](../../security/CLAUDE.md)

## Related Guides

- Storage layer: `internal/infrastructure/storage/CLAUDE.md`
- Image processor: `internal/infrastructure/storage/processor/CLAUDE.md`
- Security (ClamAV): `internal/infrastructure/security/CLAUDE.md`
- Testing: `claude/test_strategy.md`
