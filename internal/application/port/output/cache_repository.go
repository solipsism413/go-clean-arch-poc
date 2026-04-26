package output

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when a cached key is not found.
var ErrCacheMiss = errors.New("cache miss")

// CacheRepository defines the output port for caching operations.
type CacheRepository interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with an optional expiration time.
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error

	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// SetNX sets a value only if the key does not exist (useful for distributed locks).
	SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error)

	// Expire sets an expiration time on an existing key.
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// Increment increments a counter by the given value.
	Increment(ctx context.Context, key string, value int64) (int64, error)

	// DeletePattern deletes all keys matching a pattern.
	DeletePattern(ctx context.Context, pattern string) error

	// GetMultiple retrieves multiple values by keys.
	GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error)

	// SetMultiple stores multiple key-value pairs.
	SetMultiple(ctx context.Context, values map[string][]byte, expiration time.Duration) error

	// GetJSON retrieves and unmarshals a JSON value.
	GetJSON(ctx context.Context, key string, dest any) error

	// SetJSON marshals and stores a value as JSON.
	SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error
}

// CacheKeyBuilder helps build consistent cache keys.
type CacheKeyBuilder struct {
	prefix string
}

// NewCacheKeyBuilder creates a new cache key builder with a prefix.
func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

// Task returns a cache key for a task.
func (b *CacheKeyBuilder) Task(id string) string {
	return b.prefix + ":task:" + id
}

// User returns a cache key for a user.
func (b *CacheKeyBuilder) User(id string) string {
	return b.prefix + ":user:" + id
}

// UserByEmail returns a cache key for a user by email.
func (b *CacheKeyBuilder) UserByEmail(email string) string {
	return b.prefix + ":user:email:" + email
}

// TaskList returns a cache key for a task list with filter hash.
func (b *CacheKeyBuilder) TaskList(filterHash string) string {
	return b.prefix + ":tasks:list:" + filterHash
}

// Session returns a cache key for a user session.
func (b *CacheKeyBuilder) Session(sessionID string) string {
	return b.prefix + ":session:" + sessionID
}

// TokenBlacklist returns a cache key for a revoked token.
func (b *CacheKeyBuilder) TokenBlacklist(tokenID string) string {
	return b.prefix + ":token:blacklist:" + tokenID
}

// UserSessions returns a pattern for all sessions of a user.
func (b *CacheKeyBuilder) UserSessions(userID string) string {
	return b.prefix + ":session:" + userID + ":*"
}

// UserSession returns a cache key for a specific user session.
func (b *CacheKeyBuilder) UserSession(userID, tokenID string) string {
	return b.prefix + ":session:" + userID + ":" + tokenID
}

// UserList returns a cache key for a user list with filter hash.
func (b *CacheKeyBuilder) UserList(filterHash string) string {
	return b.prefix + ":users:list:" + filterHash
}
