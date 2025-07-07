package db

import (
	"database/sql"
	"fmt"

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

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}