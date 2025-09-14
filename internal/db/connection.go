package db

import (
	"context"
	"fmt"
	"time"

	"app/config"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	// Configure pgxpool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool for production
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
	// Maximum number of connections in the pool
	config.MaxConns = 25

	// Minimum number of connections to maintain in the pool
	config.MinConns = 5

	// Maximum amount of time a connection may be reused
	config.MaxConnLifetime = 5 * time.Minute

	// Maximum amount of time a connection may be idle before being closed
	config.MaxConnIdleTime = 5 * time.Minute

	// How long to wait for a connection from the pool
	config.HealthCheckPeriod = 1 * time.Minute
}

// Float64ToNumeric converts a float64 to pgtype.Numeric for database storage
func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	err := numeric.Scan(f)
	return numeric, err
}
