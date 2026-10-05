package db

import (
	"context"
	"fmt"
	"time"

	"app/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	configureConnectionPool(poolConfig)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// configureConnectionPool sets up production-ready connection pool settings
func configureConnectionPool(config *pgxpool.Config) {
	config.MaxConns = 25

	config.MinConns = 5

	config.MaxConnLifetime = 5 * time.Minute

	config.MaxConnIdleTime = 5 * time.Minute

	config.HealthCheckPeriod = 1 * time.Minute
}
