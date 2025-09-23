package commands

import (
	"flag"
	"fmt"
	"log"

	"app/cmd/cli/internal"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	dialect       = "postgres"
	migrationsDir = "migrations"
)

// RunMigrate runs database migrations
func RunMigrate(app *internal.CLIApp, args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: go run cmd/cli migrate [OPTIONS] COMMAND")
		fmt.Println()
		fmt.Println("Run database migrations")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  up                   Migrate the DB to the most recent version available")
		fmt.Println("  down                 Roll back the version by 1")
		fmt.Println("  status               Dump the migration status for the current DB")
		fmt.Println("  version              Print the current version of the database")
		fmt.Println("  create NAME          Create a new migration file")
		fmt.Println("  reset                Roll back all migrations")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run cmd/cli migrate up")
		fmt.Println("  go run cmd/cli migrate status")
		fmt.Println("  go run cmd/cli migrate create add_users_table")
		fmt.Println("  go run cmd/cli migrate --test up    # Use test database")
	}

	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
		return
	}

	command := fs.Arg(0)

	// Convert pgx connection to sql.DB for goose
	sqlDB := stdlib.OpenDB(*app.Database.Config().ConnConfig)
	defer sqlDB.Close()

	// Set up goose
	if err := goose.SetDialect(dialect); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	switch command {
	case "up":
		if err := goose.Up(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("✅ Migrations applied successfully")

	case "down":
		if err := goose.Down(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("✅ Migration rolled back successfully")

	case "status":
		if err := goose.Status(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}

	case "version":
		version, err := goose.GetDBVersion(sqlDB)
		if err != nil {
			log.Fatalf("Failed to get database version: %v", err)
		}
		fmt.Printf("Database version: %d\n", version)

	case "create":
		if fs.NArg() < 2 {
			fmt.Println("Error: migration name is required")
			fmt.Println("Usage: go run cmd/cli migrate create NAME")
			return
		}
		name := fs.Arg(1)
		if err := goose.Create(sqlDB, migrationsDir, name, "sql"); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("✅ Created migration: %s\n", name)

	case "reset":
		if err := goose.Reset(sqlDB, migrationsDir); err != nil {
			log.Fatalf("Migration reset failed: %v", err)
		}
		fmt.Println("✅ All migrations rolled back successfully")

	default:
		fmt.Printf("Unknown migration command: %s\n", command)
		fs.Usage()
	}
}
