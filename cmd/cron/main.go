package main

import (
	"flag"
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
	var (
		useTestDB = flag.Bool("test-db", false, "Use TEST_DATABASE_URL instead of DATABASE_URL")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *useTestDB {
		if err := cfg.UseTestDatabase(); err != nil {
			log.Fatal(err)
		}
		log.Println("Using TEST_DATABASE_URL for database connection")
	}

	appLogger, err := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		Output:    cfg.LogOutput,
		AddSource: cfg.Debug,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	appLogger.Info("Starting Gogo Cron Server")

	database, err := db.NewConnection(cfg)
	if err != nil {
		appLogger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	appLogger.Info("Database connection established")

	queries := db.New(database)

	deps := &scheduler.Dependencies{
		Config:  cfg,
		DB:      database,
		Queries: queries,
		Logger:  appLogger,
	}

	cronScheduler := scheduler.NewScheduler(deps)

	if err := cronScheduler.RegisterJobs(); err != nil {
		appLogger.Error("Failed to register cron jobs", "error", err)
		os.Exit(1)
	}

	cronScheduler.Start()

	entries := cronScheduler.GetEntries()
	appLogger.Info("Scheduler started with jobs", "job_count", len(entries))
	for _, entry := range entries {
		appLogger.Info("Registered cron job",
			"next_run", entry.Next.Format("2006-01-02 15:04:05"))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info("Cron server is running. Press Ctrl+C to exit.")

	<-quit
	appLogger.Info("Shutdown signal received, stopping scheduler...")

	cronScheduler.Stop()
	appLogger.Info("Gogo Cron Server stopped successfully")
}
