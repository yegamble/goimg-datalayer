package asynq

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

const (
	// Default server configuration values.
	defaultConcurrency        = 10
	defaultShutdownTimeoutSec = 30
)

// Server wraps the asynq.Server for background job processing.
// It provides configuration and lifecycle management for task workers.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	logger zerolog.Logger
}

// ServerConfig holds configuration for the Asynq server.
type ServerConfig struct {
	// RedisAddr is the Redis server address (host:port).
	RedisAddr string

	// RedisPassword is the Redis password (optional).
	RedisPassword string

	// RedisDB is the Redis database number.
	RedisDB int

	// Concurrency is the maximum number of concurrent task processing.
	// Default: 10
	Concurrency int

	// Queues defines queue priorities. Higher value = higher priority.
	// Example: {"critical": 6, "default": 3, "low": 1}
	Queues map[string]int

	// StrictPriority enforces strict queue priority (no round-robin).
	// If true, tasks in higher priority queues are always processed first.
	// Default: false
	StrictPriority bool

	// ShutdownTimeout is the maximum time to wait for tasks to finish during shutdown.
	// Default: 30 seconds
	ShutdownTimeout time.Duration

	// Logger is the structured logger for server operations.
	Logger zerolog.Logger

	// RetryDelayFunc calculates retry delay based on retry count and error.
	// Default: exponential backoff with jitter
	RetryDelayFunc asynq.RetryDelayFunc

	// ErrorHandler is called when task processing fails after all retries.
	ErrorHandler asynq.ErrorHandler
}

// DefaultServerConfig returns sensible defaults for the Asynq server.
func DefaultServerConfig(redisAddr string, logger zerolog.Logger) ServerConfig {
	return ServerConfig{
		RedisAddr:       redisAddr,
		RedisPassword:   "",
		RedisDB:         0,
		Concurrency:     defaultConcurrency,
		Queues:          map[string]int{"default": 1},
		StrictPriority:  false,
		ShutdownTimeout: defaultShutdownTimeoutSec * time.Second,
		Logger:          logger,
		RetryDelayFunc:  nil, // Use asynq default
		ErrorHandler:    nil, // No custom handler
	}
}

// NewServer creates a new Asynq server for processing background tasks.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	// Apply defaults
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConcurrency
	}
	if len(cfg.Queues) == 0 {
		cfg.Queues = map[string]int{"default": 1}
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeoutSec * time.Second
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	serverConfig := asynq.Config{
		Concurrency:     cfg.Concurrency,
		Queues:          cfg.Queues,
		StrictPriority:  cfg.StrictPriority,
		ShutdownTimeout: cfg.ShutdownTimeout,
		Logger:          newAsynqLogger(cfg.Logger),
		RetryDelayFunc:  cfg.RetryDelayFunc,
		ErrorHandler:    cfg.ErrorHandler,
	}

	server := asynq.NewServer(redisOpt, serverConfig)
	mux := asynq.NewServeMux()

	return &Server{
		server: server,
		mux:    mux,
		logger: cfg.Logger,
	}, nil
}

// RegisterHandler registers a task handler for the given task type.
// The handler will be called when a task of this type is dequeued.
func (s *Server) RegisterHandler(taskType string, handler asynq.Handler) {
	s.mux.Handle(taskType, handler)
	s.logger.Info().
		Str("task_type", taskType).
		Msg("registered task handler")
}

// RegisterHandlerFunc registers a function as a task handler.
func (s *Server) RegisterHandlerFunc(taskType string, handler func(context.Context, *asynq.Task) error) {
	s.mux.HandleFunc(taskType, handler)
	s.logger.Info().
		Str("task_type", taskType).
		Msg("registered task handler function")
}

// Start starts the Asynq server and begins processing tasks.
// This is a blocking call; run in a goroutine for background operation.
func (s *Server) Start() error {
	s.logger.Info().
		Msg("starting asynq server")

	if err := s.server.Run(s.mux); err != nil {
		s.logger.Error().
			Err(err).
			Msg("asynq server stopped with error")
		return fmt.Errorf("asynq server run: %w", err)
	}

	s.logger.Info().Msg("asynq server stopped")
	return nil
}

// Shutdown gracefully shuts down the Asynq server.
// It waits for in-flight tasks to complete up to ShutdownTimeout.
func (s *Server) Shutdown() {
	s.logger.Info().Msg("shutting down asynq server")
	s.server.Shutdown()
	s.logger.Info().Msg("asynq server shutdown complete")
}

// asynqLogger adapts zerolog.Logger to asynq.Logger interface.
type asynqLogger struct {
	logger zerolog.Logger
}

// newAsynqLogger creates a new asynq-compatible logger wrapper.
func newAsynqLogger(logger zerolog.Logger) *asynqLogger {
	return &asynqLogger{logger: logger}
}

func (l *asynqLogger) Debug(args ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(args...))
}
