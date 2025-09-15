package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"app/config"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/scheduler"
)

func main() {
	fmt.Println("🕒 Gogo Cron Jobs Server")
	fmt.Println("========================")

	var (
		useTestDB = flag.Bool("test-db", false, "Use TEST_DATABASE_URL instead of DATABASE_URL")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override database URL if using test database
	if *useTestDB {
		testDBURL := os.Getenv("TEST_DATABASE_URL")
		if testDBURL == "" {
			log.Fatal("TEST_DATABASE_URL environment variable is required when using --test-db flag")
		}
		cfg.DatabaseURL = testDBURL
		log.Println("🧪 Using TEST_DATABASE_URL for database connection")
	}

	// Initialize logger
	appLogger, err := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		Output:    cfg.LogOutput,
		AddSource: cfg.Debug,
		RequestID: false, // Not needed for cron jobs
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	appLogger.Info(context.Background(), "Starting Gogo Cron Server")

	// Initialize database
	database, err := db.NewConnection(cfg)
	if err != nil {
		appLogger.Error(context.Background(), "Failed to connect to database", err)
		os.Exit(1)
	}
	defer database.Close()

	appLogger.Info(context.Background(), "Database connection established")

	// Initialize other services
	queries := db.New(database)

	// Create scheduler dependencies
	deps := &scheduler.Dependencies{
		Config:  cfg,
		DB:      database,
		Queries: queries,
		Logger:  appLogger,
	}

	// Initialize and configure scheduler
	cronScheduler := scheduler.NewScheduler(deps)

	// Register all cron jobs
	if err := cronScheduler.RegisterJobs(); err != nil {
		appLogger.Error(context.Background(), "Failed to register cron jobs", err)
		os.Exit(1)
	}

	// Start the scheduler
	cronScheduler.Start()

	// Log registered jobs for debugging
	entries := cronScheduler.GetEntries()
	appLogger.Info(context.Background(), "Scheduler started with jobs", "job_count", len(entries))
	for _, entry := range entries {
		appLogger.Info(context.Background(), "Registered cron job", 
			"next_run", entry.Next.Format("2006-01-02 15:04:05"))
	}

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info(context.Background(), "Cron server is running. Press Ctrl+C to exit.")

	// Wait for shutdown signal
	<-quit
	appLogger.Info(context.Background(), "Shutdown signal received, stopping scheduler...")

	// Graceful shutdown
	cronScheduler.Stop()
	appLogger.Info(context.Background(), "Gogo Cron Server stopped successfully")
}