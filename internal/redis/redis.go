package redis

import (
	"context"
	"fmt"
	"time"

	"app/config"

	"github.com/redis/go-redis/v9"
)

// NewConnection creates a new Redis client with production-ready settings
func NewConnection(cfg *config.Config) (*redis.Client, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure production settings
	configureRedisOptions(opt)

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

// configureRedisOptions sets production-ready Redis options
func configureRedisOptions(opt *redis.Options) {
	// Connection pool settings
	opt.PoolSize = 20     // Max number of connections
	opt.MinIdleConns = 5  // Min idle connections to keep
	opt.MaxIdleConns = 10 // Max idle connections
	opt.ConnMaxLifetime = 5 * time.Minute
	opt.ConnMaxIdleTime = 5 * time.Minute

	// Timeout settings
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second

	// Retry settings
	opt.MaxRetries = 3
	opt.MinRetryBackoff = 8 * time.Millisecond
	opt.MaxRetryBackoff = 512 * time.Millisecond
}
