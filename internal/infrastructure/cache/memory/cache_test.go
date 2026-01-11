// Package memory_test contains tests for the in-memory cache.
package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	t.Run("should create new memory cache", func(t *testing.T) {
		cache := memory.NewMemoryCache()

		assert.NotNil(t, cache)
		assert.Equal(t, 0, cache.Size())
	})
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	ctx := context.Background()

	t.Run("should set and get a value", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "test-key"
		value := []byte("test-value")

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("should return nil for non-existent key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		result, err := cache.Get(ctx, "non-existent")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should overwrite existing value", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "test-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := cache.Set(ctx, key, value1, 0)
		require.NoError(t, err)

		err = cache.Set(ctx, key, value2, 0)
		require.NoError(t, err)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value2, result)
	})

	t.Run("should handle empty value", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "test-key"
		value := []byte{}

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})
}

func TestMemoryCache_Expiration(t *testing.T) {
	ctx := context.Background()

	t.Run("should expire value after TTL", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "expiring-key"
		value := []byte("expiring-value")

		err := cache.Set(ctx, key, value, 50*time.Millisecond)
		require.NoError(t, err)

		// Value should exist before expiration
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		// Value should be nil after expiration
		result, err = cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should not expire value with zero TTL", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "persistent-key"
		value := []byte("persistent-value")

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		// Value should exist
		time.Sleep(10 * time.Millisecond)
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})
}

func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete existing key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "delete-key"
		value := []byte("delete-value")

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		err = cache.Delete(ctx, key)
		require.NoError(t, err)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should not error when deleting non-existent key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		err := cache.Delete(ctx, "non-existent")
		require.NoError(t, err)
	})
}

func TestMemoryCache_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("should return true for existing key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "exists-key"
		value := []byte("exists-value")

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false for non-existent key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		exists, err := cache.Exists(ctx, "non-existent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("should return false for expired key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "expired-key"
		value := []byte("expired-value")

		err := cache.Set(ctx, key, value, 10*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryCache_SetNX(t *testing.T) {
	ctx := context.Background()

	t.Run("should set value when key does not exist", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "setnx-key"
		value := []byte("setnx-value")

		ok, err := cache.SetNX(ctx, key, value, 0)
		require.NoError(t, err)
		assert.True(t, ok)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("should not set value when key exists", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "setnx-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := cache.Set(ctx, key, value1, 0)
		require.NoError(t, err)

		ok, err := cache.SetNX(ctx, key, value2, 0)
		require.NoError(t, err)
		assert.False(t, ok)

		// Original value should still be there
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value1, result)
	})

	t.Run("should set value when existing key has expired", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "setnx-expired-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := cache.Set(ctx, key, value1, 10*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		ok, err := cache.SetNX(ctx, key, value2, 0)
		require.NoError(t, err)
		assert.True(t, ok)

		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value2, result)
	})
}

func TestMemoryCache_Expire(t *testing.T) {
	ctx := context.Background()

	t.Run("should set expiration on existing key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "expire-key"
		value := []byte("expire-value")

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		err = cache.Expire(ctx, key, 50*time.Millisecond)
		require.NoError(t, err)

		// Value should still exist
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		result, err = cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should not error on non-existent key", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		err := cache.Expire(ctx, "non-existent", time.Hour)
		require.NoError(t, err)
	})
}

func TestMemoryCache_Increment(t *testing.T) {
	ctx := context.Background()

	t.Run("should increment non-existent key from zero", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		result, err := cache.Increment(ctx, "counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result)
	})

	t.Run("should increment existing counter", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "counter"

		result1, err := cache.Increment(ctx, key, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result1)

		result2, err := cache.Increment(ctx, key, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(15), result2)
	})

	t.Run("should handle negative increment", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		key := "counter"

		result1, err := cache.Increment(ctx, key, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result1)

		result2, err := cache.Increment(ctx, key, -3)
		require.NoError(t, err)
		assert.Equal(t, int64(7), result2)
	})
}

func TestMemoryCache_DeletePattern(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete keys matching pattern", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		_ = cache.Set(ctx, "user:1", []byte("data1"), 0)
		_ = cache.Set(ctx, "user:2", []byte("data2"), 0)
		_ = cache.Set(ctx, "task:1", []byte("data3"), 0)

		err := cache.DeletePattern(ctx, "user:*")
		require.NoError(t, err)

		// User keys should be deleted
		result, _ := cache.Get(ctx, "user:1")
		assert.Nil(t, result)
		result, _ = cache.Get(ctx, "user:2")
		assert.Nil(t, result)

		// Task key should still exist
		result, _ = cache.Get(ctx, "task:1")
		assert.Equal(t, []byte("data3"), result)
	})

	t.Run("should handle pattern without wildcard", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		_ = cache.Set(ctx, "prefix-key1", []byte("data1"), 0)
		_ = cache.Set(ctx, "prefix-key2", []byte("data2"), 0)
		_ = cache.Set(ctx, "other-key", []byte("data3"), 0)

		err := cache.DeletePattern(ctx, "prefix")
		require.NoError(t, err)

		// Keys starting with prefix should be deleted
		result, _ := cache.Get(ctx, "prefix-key1")
		assert.Nil(t, result)
		result, _ = cache.Get(ctx, "prefix-key2")
		assert.Nil(t, result)

		// Other key should still exist
		result, _ = cache.Get(ctx, "other-key")
		assert.Equal(t, []byte("data3"), result)
	})
}

func TestMemoryCache_GetMultiple(t *testing.T) {
	ctx := context.Background()

	t.Run("should get multiple values", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value1"), 0)
		_ = cache.Set(ctx, "key2", []byte("value2"), 0)
		_ = cache.Set(ctx, "key3", []byte("value3"), 0)

		result, err := cache.GetMultiple(ctx, []string{"key1", "key2", "key4"})
		require.NoError(t, err)

		assert.Len(t, result, 2)
		assert.Equal(t, []byte("value1"), result["key1"])
		assert.Equal(t, []byte("value2"), result["key2"])
		_, exists := result["key4"]
		assert.False(t, exists)
	})

	t.Run("should return empty map for empty keys", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		result, err := cache.GetMultiple(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("should exclude expired keys", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		_ = cache.Set(ctx, "valid", []byte("valid-value"), 0)
		_ = cache.Set(ctx, "expired", []byte("expired-value"), 10*time.Millisecond)

		time.Sleep(20 * time.Millisecond)

		result, err := cache.GetMultiple(ctx, []string{"valid", "expired"})
		require.NoError(t, err)

		assert.Len(t, result, 1)
		assert.Equal(t, []byte("valid-value"), result["valid"])
	})
}

func TestMemoryCache_SetMultiple(t *testing.T) {
	ctx := context.Background()

	t.Run("should set multiple values", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		values := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
			"key3": []byte("value3"),
		}

		err := cache.SetMultiple(ctx, values, 0)
		require.NoError(t, err)

		for key, expected := range values {
			result, err := cache.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("should set multiple values with expiration", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		values := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}

		err := cache.SetMultiple(ctx, values, 50*time.Millisecond)
		require.NoError(t, err)

		// Values should exist before expiration
		result, _ := cache.Get(ctx, "key1")
		assert.Equal(t, []byte("value1"), result)

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		result, _ = cache.Get(ctx, "key1")
		assert.Nil(t, result)
	})
}

func TestMemoryCache_Close(t *testing.T) {
	t.Run("should clear all data on close", func(t *testing.T) {
		ctx := context.Background()
		cache := memory.NewMemoryCache()

		_ = cache.Set(ctx, "key1", []byte("value1"), 0)
		_ = cache.Set(ctx, "key2", []byte("value2"), 0)

		assert.Equal(t, 2, cache.Size())

		err := cache.Close()
		require.NoError(t, err)

		assert.Equal(t, 0, cache.Size())
	})
}

func TestMemoryCache_Size(t *testing.T) {
	ctx := context.Background()

	t.Run("should return correct size", func(t *testing.T) {
		cache := memory.NewMemoryCache()
		defer cache.Close()

		assert.Equal(t, 0, cache.Size())

		_ = cache.Set(ctx, "key1", []byte("value1"), 0)
		assert.Equal(t, 1, cache.Size())

		_ = cache.Set(ctx, "key2", []byte("value2"), 0)
		assert.Equal(t, 2, cache.Size())

		_ = cache.Delete(ctx, "key1")
		assert.Equal(t, 1, cache.Size())
	})
}
