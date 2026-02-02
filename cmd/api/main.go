package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func main() {
	// 1. Setup Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().Msg("Starting goimg-datalayer API server...")

	// 2. Load Configuration (Environment Variables)
	dbConfig := postgres.ConfigFromEnv()
	redisConfig := redis.DefaultConfig() // TODO: Load from Env if needed

	// 3. Initialize Infrastructure
	// Database
	db, err := postgres.NewDB(dbConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database (continuing in degraded mode)")
	} else {
		log.Info().Msg("Connected to database")
		defer postgres.Close(db)
	}

	// Redis
	redisClient, err := redis.NewClient(redisConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis (continuing in degraded mode)")
	} else {
		log.Info().Msg("Connected to Redis")
		defer redisClient.Close()
	}

	// 4. Initialize Handlers
	// Health Handler
	healthHandler := handlers.NewHealthHandler(
		db,
		redisClient,
		nil, // storage (nil for now)
		nil, // clamav (nil for now)
		log.Logger,
	)

	// Middleware Configuration
	middlewareConfig := handlers.MiddlewareConfig{
		Logger: log.Logger,
		// JWTService: jwtService, // TODO: Initialize JWT Service
		// TokenBlacklist: tokenBlacklist, // TODO: Initialize Token Blacklist
	}

	// 5. Initialize Router
	router := handlers.NewRouter(
		nil, // authHandler
		nil, // userHandler
		nil, // imageHandler
		nil, // albumHandler
		nil, // socialHandler
		nil, // exploreHandler
		healthHandler,
		nil, // twoFAHandler
		nil, // oauthHandler
		nil, // followHandler
		nil, // activityHandler
		nil, // notificationHandler
		nil, // ipfsHandler
		nil, // moderationHandler
		nil, // guestHandler
		nil, // oembedHandler
		nil, // previewHandler
		nil, // variantConfigHandler
		nil, // tagHandler
		nil, // featuredHandler
		nil, // groupHandler
		nil, // groupAlbumHandler
		middleware.NewMetricsCollector(),
		middlewareConfig,
		false, // isProd (TODO: Load from env)
	)

	// 6. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful Shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server startup failed")
		}
	}()

	log.Info().Msgf("Server listening on port %s", port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}
