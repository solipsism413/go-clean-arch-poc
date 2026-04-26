// Package memory provides an in-memory cache implementation.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
)

// Ensure MemoryCache implements the output.CacheRepository interface.
var _ output.CacheRepository = (*MemoryCache)(nil)

// cacheItem represents an item in the cache with expiration.
type cacheItem struct {
	value      []byte
	expiration time.Time
}

// MemoryCache implements an in-memory cache with TTL support.
type MemoryCache struct {
	store map[string]cacheItem
	mu    sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		store: make(map[string]cacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// cleanup removes expired items periodically.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.store {
			if !item.expiration.IsZero() && now.After(item.expiration) {
				delete(c.store, key)
			}
		}
		c.mu.Unlock()
	}
}

// Get retrieves a value by key.
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.store[key]
	if !ok {
		return nil, nil
	}

	// Check expiration
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return nil, nil
	}

	return item.value, nil
}

// Set stores a value with an optional expiration time.
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	c.store[key] = cacheItem{
		value:      value,
		expiration: exp,
	}

	return nil
}

// Delete removes a value by key.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}

// Exists checks if a key exists.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.store[key]
	if !ok {
		return false, nil
	}

	// Check expiration
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return false, nil
	}

	return true, nil
}

// SetNX sets a value only if the key does not exist.
func (c *MemoryCache) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.store[key]; ok {
		// Check if existing item is still valid
		if item.expiration.IsZero() || time.Now().Before(item.expiration) {
			return false, nil
		}
	}

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	c.store[key] = cacheItem{
		value:      value,
		expiration: exp,
	}

	return true, nil
}

// Expire sets an expiration time on an existing key.
func (c *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.store[key]; ok {
		item.expiration = time.Now().Add(expiration)
		c.store[key] = item
	}

	return nil
}

// Increment increments a counter by the given value.
func (c *MemoryCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var current int64
	if item, ok := c.store[key]; ok {
		// Try to parse existing value as int64
		if len(item.value) == 8 {
			for i := 0; i < 8; i++ {
				current |= int64(item.value[i]) << (i * 8)
			}
		}
	}

	newValue := current + value

	// Store as bytes
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(newValue >> (i * 8))
	}

	c.store[key] = cacheItem{
		value:      bytes,
		expiration: time.Time{}, // No expiration for counters
	}

	return newValue, nil
}

// DeletePattern deletes all keys matching a pattern.
func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple pattern matching (only supports prefix*)
	prefix := pattern
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix = pattern[:len(pattern)-1]
	}

	for key := range c.store {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.store, key)
		}
	}

	return nil
}

// GetMultiple retrieves multiple values by keys.
func (c *MemoryCache) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]byte)
	now := time.Now()

	for _, key := range keys {
		if item, ok := c.store[key]; ok {
			if item.expiration.IsZero() || now.Before(item.expiration) {
				result[key] = item.value
			}
		}
	}

	return result, nil
}

// SetMultiple stores multiple key-value pairs.
func (c *MemoryCache) SetMultiple(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	for key, value := range values {
		c.store[key] = cacheItem{
			value:      value,
			expiration: exp,
		}
	}

	return nil
}

// GetJSON retrieves and unmarshals a JSON value.
func (c *MemoryCache) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return output.ErrCacheMiss
	}
	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a value as JSON.
func (c *MemoryCache) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	return c.Set(ctx, key, data, expiration)
}

// Close clears the cache.
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]cacheItem)
	return nil
}

// Size returns the number of items in the cache.
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.store)
}
