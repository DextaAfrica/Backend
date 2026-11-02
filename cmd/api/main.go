package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DextaAfrica/Backend/internal/application"
	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/infrastructure/cache"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"
	httpInterface "github.com/DextaAfrica/Backend/internal/interfaces/http"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// 2. Initialize Structured Logger
	logger.Init(cfg.App.Environment)
	logger.Log.Info().
		Str("app", cfg.App.Name).
		Str("env", cfg.App.Environment).
		Msg("Initializing Dexta Africa Enterprise Core Engine...")

	// 3. Connect to PostgreSQL
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("PostgreSQL connection deferred or unavailable in current environment")
	} else {
		// Run auto-migrations
		if err := db.AutoMigrate(); err != nil {
			logger.Log.Error().Err(err).Msg("Database schema migration error")
		}
	}

	// 4. Connect to Redis
	var redisClient *cache.RedisClient
	if cfg.RateLimiter.Enabled {
		rClient, err := cache.Connect(&cfg.Redis)
		if err != nil {
			logger.Log.Warn().Err(err).Msg("Redis unavailable, rate limiter operating in fallback mode")
		} else {
			redisClient = rClient
		}
	}

	// 5. Start Transactional Outbox Worker if database is connected
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	if db != nil {
		outboxWorker := application.NewOutboxWorker(db, cfg)
		go outboxWorker.Start(workerCtx)
	}

	// 6. Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.App.ReadTimeout,
		WriteTimeout:          cfg.App.WriteTimeout,
		DisableStartupMessage: false,
	})

	// 7. Setup HTTP Routers & Middlewares
	httpInterface.SetupRouter(app, cfg, db, redisClient)

	// 8. Graceful Shutdown Listener
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownCh
		logger.Log.Info().Msg("Graceful shutdown signal received. Shutting down HTTP listeners...")
		cancelWorker()
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			logger.Log.Error().Err(err).Msg("Error while shutting down HTTP server")
		}
	}()

	// 9. Start Server
	addr := ":" + cfg.App.Port
	logger.Log.Info().Str("address", addr).Msg("Dexta Africa Enterprise API is running")

	if err := app.Listen(addr); err != nil {
		logger.Log.Info().Err(err).Msg("HTTP server closed")
	}
}

