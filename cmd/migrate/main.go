package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"myapp/config"
	"myapp/internal/db"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	dialect    = "postgres"
	migrationsDir = "migrations"
)

var (
	flags = flag.NewFlagSet("migrate", flag.ExitOnError)
	dir   = flags.String("dir", migrationsDir, "directory with migration files")
	test  = flags.Bool("test", false, "use test database instead of main database")
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()
	if len(args) == 0 {
		flags.Usage()
		return
	}

	command := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Use test database if --test flag is provided
	if *test {
		if cfg.TestDatabaseURL == "" {
			log.Fatalf("TEST_DATABASE_URL is required when using --test flag")
		}
		cfg.DatabaseURL = cfg.TestDatabaseURL
	}

	// Connect to database
	database, err := db.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Get underlying database connection for goose compatibility
	sqlDB := stdlib.OpenDB(*database.Config().ConnConfig)

	// Run the migration command
	if err := runGoose(sqlDB, command, args[1:]...); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func runGoose(db *sql.DB, command string, args ...string) error {
	switch command {
	case "up":
		return goose.Up(db, *dir)
	case "up-by-one":
		return goose.UpByOne(db, *dir)
	case "up-to":
		if len(args) == 0 {
			return fmt.Errorf("up-to requires a version argument")
		}
		return goose.UpTo(db, *dir, parseInt64(args[0]))
	case "down":
		return goose.Down(db, *dir)
	case "down-to":
		if len(args) == 0 {
			return fmt.Errorf("down-to requires a version argument")
		}
		return goose.DownTo(db, *dir, parseInt64(args[0]))
	case "redo":
		return goose.Redo(db, *dir)
	case "reset":
		return goose.Reset(db, *dir)
	case "status":
		return goose.Status(db, *dir)
	case "version":
		return goose.Version(db, *dir)
	case "create":
		if len(args) == 0 {
			return fmt.Errorf("create requires a name argument")
		}
		return goose.Create(db, *dir, args[0], "sql")
	case "fix":
		return goose.Fix(*dir)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func parseInt64(s string) int64 {
	var version int64
	if _, err := fmt.Sscanf(s, "%d", &version); err != nil {
		log.Fatalf("Invalid version: %s", s)
	}
	return version
}

func usage() {
	fmt.Println("Usage: go run cmd/migrate/main.go [OPTIONS] COMMAND")
	fmt.Println("\nCommands:")
	fmt.Println("  up                   Migrate the DB to the most recent version available")
	fmt.Println("  up-by-one           Migrate the DB up by 1")
	fmt.Println("  up-to VERSION       Migrate the DB to a specific VERSION")
	fmt.Println("  down                Roll back the version by 1")
	fmt.Println("  down-to VERSION     Roll back to a specific VERSION")
	fmt.Println("  redo                Re-run the latest migration")
	fmt.Println("  reset               Roll back all migrations")
	fmt.Println("  status              Print the status of all migrations")
	fmt.Println("  version             Print the current version of the database")
	fmt.Println("  create NAME         Creates new migration file with the current timestamp")
	fmt.Println("  fix                 Apply sequential ordering to migrations")
	fmt.Println("\nOptions:")
	flags.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  go run cmd/migrate/main.go up")
	fmt.Println("  go run cmd/migrate/main.go --test up")
	fmt.Println("  go run cmd/migrate/main.go down")
	fmt.Println("  go run cmd/migrate/main.go status")
	fmt.Println("  go run cmd/migrate/main.go create add_user_avatar")
}