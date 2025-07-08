package db

import (
	"database/sql"
	"fmt"
	"time"

	"myapp/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewConnection(cfg *config.Config) (*sql.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	dsn := cfg.DatabaseURL

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for production
	configureConnectionPool(db)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// configureConnectionPool sets up production-ready connection pool settings
func configureConnectionPool(db *sql.DB) {
	// Maximum number of open connections to the database
	db.SetMaxOpenConns(25)

	// Maximum number of idle connections in the pool
	db.SetMaxIdleConns(5)

	// Maximum amount of time a connection may be reused
	db.SetConnMaxLifetime(5 * time.Minute)

	// Maximum amount of time a connection may be idle before being closed
	db.SetConnMaxIdleTime(5 * time.Minute)
}