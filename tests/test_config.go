package tests

import (
	"context"
	"log"
	"os"
	"sync"

	"myapp/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var (
	testPool *pgxpool.Pool
	once     sync.Once
)

// GetTestDBPool returns a shared test database connection pool
func GetTestDBPool() *pgxpool.Pool {
	once.Do(func() {
		// Try to load .env file from common locations
		envPaths := []string{
			".env",                    // Current directory
			"../.env",                 // Parent directory
			"../../.env",              // Two levels up
		}
		
		var envLoaded bool
		for _, path := range envPaths {
			if err := godotenv.Load(path); err == nil {
				log.Printf("Loaded .env file from: %s", path)
				envLoaded = true
				break
			}
		}
		
		if !envLoaded {
			log.Printf("Could not load .env file from any location, using system environment variables")
		}

		testDBURL := os.Getenv("TEST_DATABASE_URL")
		if testDBURL == "" {
			panic("TEST_DATABASE_URL environment variable is required for tests")
		}

		// Create test configuration
		cfg := &config.Config{
			DatabaseURL: testDBURL,
		}

		// Create connection pool with same settings as production
		poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			panic("Failed to parse test database URL: " + err.Error())
		}

		// Configure for testing - smaller pool
		poolConfig.MaxConns = 5
		poolConfig.MinConns = 1

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			panic("Failed to create test database pool: " + err.Error())
		}

		if err := pool.Ping(context.Background()); err != nil {
			pool.Close()
			panic("Failed to ping test database: " + err.Error())
		}

		testPool = pool
	})

	return testPool
}

// CloseTestDB closes the test database connection
func CloseTestDB() {
	if testPool != nil {
		testPool.Close()
	}
}