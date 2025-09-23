package main

import (
	"fmt"
	"log"
	"os"

	"app/cmd/cli/internal"
	"app/cmd/cli/internal/commands"
	"app/config"
	"app/internal/db"
	"app/internal/logger"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	commandName := os.Args[1]
	args := os.Args[2:]

	// Handle help and non-database commands first
	switch commandName {
	case "help", "--help", "-h":
		printUsage()
		return
	}

	// Initialize shared app components for commands that need them
	app, err := initializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer app.Database.Close()

	switch commandName {
	case "migrate":
		commands.RunMigrate(app, args)
	case "test":
		commands.RunTest(app, args)
	default:
		fmt.Printf("Unknown command: %s\n\n", commandName)
		printUsage()
		os.Exit(1)
	}
}

func initializeApp() (*internal.CLIApp, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Check for test database flag
	useTestDB := false
	for _, arg := range os.Args {
		if arg == "--test" {
			useTestDB = true
			break
		}
	}

	// Override database URL if using test database
	if useTestDB {
		testDBURL := os.Getenv("TEST_DATABASE_URL")
		if testDBURL == "" {
			return nil, fmt.Errorf("TEST_DATABASE_URL environment variable is required when using --test flag")
		}
		cfg.DatabaseURL = testDBURL
		log.Println("Using TEST_DATABASE_URL for database connection")
	}

	// Initialize logger
	appLogger, err := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		Output:    cfg.LogOutput,
		AddSource: cfg.Debug,
		RequestID: false, // Not needed for CLI
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize database
	database, err := db.NewConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize other services
	queries := db.New(database)

	return &internal.CLIApp{
		Config:   cfg,
		Database: database,
		Queries:  queries,
		Logger:   appLogger,
	}, nil
}

func printUsage() {
	fmt.Println("Gogo CLI - Command line interface for app management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/cli <command> [options]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  migrate              Run database migrations")
	fmt.Println("  test                 Run various tests")
	fmt.Println("  help                 Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/cli migrate up")
	fmt.Println("  go run cmd/cli migrate status")
	fmt.Println("  go run cmd/cli test")
	fmt.Println()
	fmt.Println("For more information on a specific command:")
	fmt.Println("  go run cmd/cli <command> --help")
}
