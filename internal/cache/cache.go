package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides a Go-idiomatic caching interface
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Remember(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error), dest interface{}) error
	Forget(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Has(ctx context.Context, key string) (bool, error)
}

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	if prefix == "" {
		prefix = "cache:"
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a value from cache and unmarshals it to dest
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, c.key(key)).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrKeyNotFound
		}
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete removes a key from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

// Remember gets a value from cache or stores it if it doesn't exist
func (c *RedisCache) Remember(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error), dest interface{}) error {
	// Try to get from cache first
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil // Found in cache
	}
	if err != ErrKeyNotFound {
		return err // Real error occurred
	}

	// Not in cache, call callback to get value
	value, err := callback()
	if err != nil {
		return err
	}

	// Store in cache for next time
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	// Marshal the value to dest
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Forget is an alias for Delete (Laravel-style)
func (c *RedisCache) Forget(ctx context.Context, key string) error {
	return c.Delete(ctx, key)
}

// Flush clears all cache entries with the prefix
func (c *RedisCache) Flush(ctx context.Context) error {
	pattern := c.prefix + "*"
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

// Has checks if a key exists in cache
func (c *RedisCache) Has(ctx context.Context, key string) (bool, error) {
	result := c.client.Exists(ctx, c.key(key))
	count, err := result.Result()
	return count > 0, err
}

// key adds the prefix to the key
func (c *RedisCache) key(key string) string {
	return c.prefix + key
}

// Cache errors
var (
	ErrKeyNotFound = redis.Nil
)
