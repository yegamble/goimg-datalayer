# Command Entrypoints Guide

> Application entry points: API server, background worker, migration CLI.

## Structure

```
cmd/
├── api/
│   └── main.go           # HTTP API server
├── worker/
│   └── main.go           # Background job processor
└── migrate/
    └── main.go           # Database migration CLI
```

## API Server (`cmd/api`)

```go
func main() {
    // 1. Load configuration
    cfg := config.Load()

    // 2. Initialize infrastructure
    db := postgres.NewConnection(cfg.DatabaseURL)
    redis := redis.NewClient(cfg.RedisURL)

    // 3. Initialize repositories
    userRepo := postgres.NewUserRepository(db)

    // 4. Initialize application services
    createUser := commands.NewCreateUserHandler(userRepo, publisher)

    // 5. Initialize HTTP handlers
    userHandler := handlers.NewUserHandler(createUser)

    // 6. Setup router with middleware
    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.Logger)

    // 7. Start server
    server.ListenAndServe(cfg.Port, router)
}
```

## Worker (`cmd/worker`)

Processes background jobs via Asynq (Redis-based queue):

- Image variant generation
- Malware scanning
- Email notifications
- Cleanup tasks

## Migrate (`cmd/migrate`)

```bash
# Run migrations
./bin/migrate up

# Rollback
./bin/migrate down

# Create new migration
./bin/migrate create add_users_table
```

## Running Locally

```bash
make run              # Start API server
make run-worker       # Start background worker
make migrate-up       # Run migrations
```

## See Also

- Architecture: `claude/architecture.md`
- Docker setup: `docker/docker-compose.yml`
