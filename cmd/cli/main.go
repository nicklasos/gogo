package main

import (
	"flag"
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
	global := flag.NewFlagSet("cli", flag.ExitOnError)
	useTestDB := global.Bool("test", false, "Use TEST_DATABASE_URL instead of DATABASE_URL")
	global.Usage = printUsage
	global.Parse(os.Args[1:])

	if global.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}

	commandName := global.Arg(0)
	args := global.Args()[1:]

	if commandName == "help" {
		printUsage()
		return
	}

	app, err := initializeApp(*useTestDB)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer app.Database.Close()

	switch commandName {
	case "migrate":
		commands.RunMigrate(app, args)
	case "create-user":
		commands.RunCreateUser(app, args)
	case "test":
		commands.RunTest(app, args)
	default:
		fmt.Printf("Unknown command: %s\n\n", commandName)
		printUsage()
		os.Exit(1)
	}
}

func initializeApp(useTestDB bool) (*internal.CLIApp, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if useTestDB {
		if err := cfg.UseTestDatabase(); err != nil {
			return nil, err
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
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	database, err := db.NewConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &internal.CLIApp{
		Config:   cfg,
		Database: database,
		Queries:  db.New(database),
		Logger:   appLogger,
	}, nil
}

func printUsage() {
	fmt.Println("Gogo CLI - Command line interface for app management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/cli [--test] <command> [options]")
	fmt.Println()
	fmt.Println("Global options:")
	fmt.Println("  --test               Use TEST_DATABASE_URL instead of DATABASE_URL")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  migrate              Run database migrations")
	fmt.Println("  create-user          Create a user (default role: super-admin)")
	fmt.Println("  test                 Check the database connection")
	fmt.Println("  help                 Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./cmd/cli migrate up")
	fmt.Println("  go run ./cmd/cli --test migrate status")
	fmt.Println("  go run ./cmd/cli create-user --email admin@example.com --password password123")
	fmt.Println()
	fmt.Println("For more information on a specific command:")
	fmt.Println("  go run ./cmd/cli <command> --help")
}
