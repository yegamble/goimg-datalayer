# Asynq Background Job Patterns

Redis-backed async task queue for image processing and malware scanning.

## Architecture

```
infrastructure/jobs/
├── asynq/
│   ├── client.go    # Task enqueuing (used by HTTP handlers)
│   └── server.go    # Task processing (run in worker binary)
└── tasks/
    ├── image_process.go   # TypeImageProcess = "image:process"
    └── image_scan.go      # TypeImageScan = "image:scan"
```

## Current Task Types

| Type | Const | Queue | Retries | Timeout |
|------|-------|-------|---------|---------|
| Image processing | `TypeImageProcess` | default | 3 | 5 min |
| Malware scan | `TypeImageScan` | default | 2 | 2 min |

## Task Definition Pattern

```go
// 1. Define type constant
const TypeFooProcess = "foo:process"

// 2. Define payload struct (JSON-serializable)
type FooProcessPayload struct {
    FooID      string    `json:"foo_id"`
    StorageKey string    `json:"storage_key"`
    EnqueuedAt time.Time `json:"enqueued_at"`
}

// 3. Implement asynq.Handler interface
type FooProcessHandler struct {
    // dependencies injected via constructor
}

func (h *FooProcessHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload FooProcessPayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return fmt.Errorf("unmarshal payload: %w", err)
    }
    // ... process ...
    // Return asynq.SkipRetry for terminal failures (e.g., malware detected)
}
```

## Enqueuing (from HTTP handlers / application layer)

```go
// Use the injected asynq.Client
if err := h.jobClient.EnqueueTask(ctx, tasks.TypeImageProcess, payload); err != nil {
    h.logger.Error().Err(err).Msg("failed to enqueue task")
    // Don't fail the HTTP request — processing retried later
}
// Upload endpoint returns 202 Accepted (not 200)
```

## Worker Registration (cmd/worker/main.go)

```go
server.RegisterHandlerFunc(tasks.TypeFooProcess, fooHandler.ProcessTask)
```

## Retry Behavior

- **Retryable**: return normal `fmt.Errorf(...)` — uses exponential backoff
- **Terminal failure**: `return fmt.Errorf("reason: %w", asynq.SkipRetry)` — no retry
- Error handler logs failures after all retries via `ErrorHandler` config option

## Queue Priority

```go
Queues: map[string]int{
    "critical": 6,  // User-facing operations
    "default":  3,  // Image processing, scanning
    "low":      1,  // Analytics, cleanup
}
```

## Enqueue to Non-Default Queue

```go
client.EnqueueTask(ctx, taskType, payload, asynq.Queue("critical"))
```

## Testing

Unit tests: mock `storage.Storage` and `processor.Processor` interfaces, create task with `asynq.NewTask(type, jsonPayload)`, call `ProcessTask` directly.

```go
task := asynq.NewTask(tasks.TypeImageProcess, payloadBytes)
err := handler.ProcessTask(context.Background(), task)
```

## Key Rules

- Workers run in separate binary (`make run-worker`)
- Always set `EnqueuedAt: time.Now()` in payload for latency tracking
- Log task start, completion, and errors with `image_id` field
- Never block HTTP responses on job completion — use 202 Accepted
- ClamAV "malware detected" → `asynq.SkipRetry` (no point retrying)
