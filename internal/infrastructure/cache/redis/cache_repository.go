// Package redis provides Redis cache connection using go-redis.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/redis/go-redis/v9"
)

// Ensure CacheRepository implements the output.CacheRepository interface.
var _ output.CacheRepository = (*CacheRepository)(nil)

// CacheRepository implements the cache repository using Redis.
type CacheRepository struct {
	client *redis.Client
	logger *slog.Logger
}

// NewCacheRepository creates a new Redis cache repository.
func NewCacheRepository(ctx context.Context, cfg config.RedisConfig, logger *slog.Logger) (*CacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("redis connection established",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DB,
	)

	return &CacheRepository{
		client: client,
		logger: logger,
	}, nil
}

// Get retrieves a value by key.
func (r *CacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

// Set stores a value with an optional expiration time.
func (r *CacheRepository) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	return nil
}

// Delete removes a value by key.
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// Exists checks if a key exists.
func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists failed: %w", err)
	}
	return result > 0, nil
}

// SetNX sets a value only if the key does not exist.
func (r *CacheRepository) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	result, err := r.client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx failed: %w", err)
	}
	return result, nil
}

// Expire sets an expiration time on an existing key.
func (r *CacheRepository) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := r.client.Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("redis expire failed: %w", err)
	}
	return nil
}

// Increment increments a counter by the given value.
func (r *CacheRepository) Increment(ctx context.Context, key string, value int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incrby failed: %w", err)
	}
	return result, nil
}

// DeletePattern deletes all keys matching a pattern.
func (r *CacheRepository) DeletePattern(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("redis delete pattern failed: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan failed: %w", err)
	}
	return nil
}

// GetMultiple retrieves multiple values by keys.
func (r *CacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget failed: %w", err)
	}

	result := make(map[string][]byte)
	for i, key := range keys {
		if values[i] != nil {
			if str, ok := values[i].(string); ok {
				result[key] = []byte(str)
			}
		}
	}
	return result, nil
}

// SetMultiple stores multiple key-value pairs.
func (r *CacheRepository) SetMultiple(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	pipe := r.client.Pipeline()
	for key, value := range values {
		pipe.Set(ctx, key, value, expiration)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline exec failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (r *CacheRepository) Close() error {
	return r.client.Close()
}

// Health checks if Redis is healthy.
func (r *CacheRepository) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetJSON retrieves and unmarshals a JSON value.
func (r *CacheRepository) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return output.ErrCacheMiss
	}
	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a value as JSON.
func (r *CacheRepository) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	return r.Set(ctx, key, data, expiration)
}
