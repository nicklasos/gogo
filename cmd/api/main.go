package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app/config"
	"app/docs"
	"app/internal"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/mail"
	"app/internal/redis"
	"app/internal/scheduler"
	"app/internal/server"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Gogo API Template
// @version         1.0
// @description     A production-ready Go API template with authentication and CRUD examples
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  WTFPL
// @license.url   http://www.wtfpl.net/

// @host      localhost:8181
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	var (
		port      = flag.String("port", "", "Server port (overrides config)")
		useTestDB = flag.Bool("test-db", false, "Use TEST_DATABASE_URL instead of DATABASE_URL")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	if *useTestDB {
		if err := cfg.UseTestDatabase(); err != nil {
			log.Fatal(err)
		}
		log.Println("Using TEST_DATABASE_URL for database connection")
	}

	if *port != "" {
		cfg.Port = *port
	}

	if err := cfg.ValidateJWTSecret(); err != nil {
		log.Fatal(err)
	}

	logger, err := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		Output:    cfg.LogOutput,
		AddSource: cfg.Debug,
		RequestID: true,
	})
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	logger.Info("Starting application",
		"app_name", cfg.AppName,
		"version", cfg.AppVersion,
		"environment", cfg.Environment,
		"debug", cfg.Debug,
	)

	if cfg.JWTSecretIsWeak() {
		logger.Warn("JWT_SECRET is weak: fine for local development, refused when APP_ENV=production")
	}

	database, err := db.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	queries := db.New(database)

	redisClient, err := redis.NewConnection(cfg)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	r := server.NewEngine(cfg, logger)

	app := &internal.App{
		Config:  cfg,
		DB:      database,
		Queries: queries,
		Tx:      db.NewTxRunner(database, queries),
		Cache:   cache.NewRedisCache(redisClient, cfg.AppName+":"),
		Logger:  logger,
		Mail:    mail.NewService(cfg, logger),
		Api:     r.Group("/api/v1"),
		Images:  internal.NewImageService(cfg.FilesBaseURL),
	}

	if cfg.EnableScheduler {
		deps := &scheduler.Dependencies{
			Config:  cfg,
			DB:      database,
			Queries: app.Queries,
			Logger:  logger,
		}

		cronScheduler := scheduler.NewScheduler(deps)
		if err := cronScheduler.RegisterJobs(); err != nil {
			logger.Error("Failed to register scheduler jobs", "error", err)
			log.Fatal("Failed to register scheduler jobs:", err)
		}

		cronScheduler.Start()
		logger.Info("Scheduler started in integrated mode")
		defer cronScheduler.Stop()
	}

	server.RegisterRoutes(r, app)

	docs.SwaggerInfo.Host = cfg.AppURL
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	address := ":" + cfg.Port
	srv := &http.Server{
		Addr:              address,
		Handler:           r.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("Server starting", "address", address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", "error", err, "address", address)
			log.Fatal(err)
		}
	}()

	// Without this a stop signal kills the process at once: in-flight requests are cut off
	// and the deferred scheduler stop and connection closes never run.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown did not complete cleanly", "error", err)
	}
}
