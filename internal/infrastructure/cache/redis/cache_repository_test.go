// Package redis_test contains tests for the Redis cache repository.
package redis_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	rediscache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/redis"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestCacheRepository creates a new CacheRepository with miniredis for testing.
func setupTestCacheRepository(t *testing.T) (*rediscache.CacheRepository, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	cfg := config.RedisConfig{
		Host:         mr.Host(),
		Port:         mr.Server().Addr().Port,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 1,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	repo, err := rediscache.NewCacheRepository(context.Background(), cfg, logger)
	require.NoError(t, err)

	return repo, mr
}

func TestNewCacheRepository(t *testing.T) {
	t.Run("should create cache repository successfully", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		assert.NotNil(t, repo)
	})

	t.Run("should fail with invalid connection", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))

		cfg := config.RedisConfig{
			Host:        "invalid-host",
			Port:        12345,
			DialTimeout: 100 * time.Millisecond,
		}

		_, err := rediscache.NewCacheRepository(context.Background(), cfg, logger)
		assert.Error(t, err)
	})
}

func TestCacheRepository_SetAndGet(t *testing.T) {
	ctx := context.Background()

	t.Run("should set and get a value", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "test-key"
		value := []byte("test-value")

		err := repo.Set(ctx, key, value, 0)
		require.NoError(t, err)

		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("should return nil for non-existent key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		result, err := repo.Get(ctx, "non-existent")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should overwrite existing value", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "test-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := repo.Set(ctx, key, value1, 0)
		require.NoError(t, err)

		err = repo.Set(ctx, key, value2, 0)
		require.NoError(t, err)

		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value2, result)
	})
}

func TestCacheRepository_Expiration(t *testing.T) {
	ctx := context.Background()

	t.Run("should expire value after TTL", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "expiring-key"
		value := []byte("expiring-value")

		err := repo.Set(ctx, key, value, 1*time.Second)
		require.NoError(t, err)

		// Value should exist before expiration
		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Fast-forward time in miniredis
		mr.FastForward(2 * time.Second)

		// Value should be nil after expiration
		result, err = repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCacheRepository_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete existing key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "delete-key"
		value := []byte("delete-value")

		err := repo.Set(ctx, key, value, 0)
		require.NoError(t, err)

		err = repo.Delete(ctx, key)
		require.NoError(t, err)

		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("should not error when deleting non-existent key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		err := repo.Delete(ctx, "non-existent")
		require.NoError(t, err)
	})
}

func TestCacheRepository_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("should return true for existing key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "exists-key"
		value := []byte("exists-value")

		err := repo.Set(ctx, key, value, 0)
		require.NoError(t, err)

		exists, err := repo.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false for non-existent key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		exists, err := repo.Exists(ctx, "non-existent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestCacheRepository_SetNX(t *testing.T) {
	ctx := context.Background()

	t.Run("should set value when key does not exist", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "setnx-key"
		value := []byte("setnx-value")

		ok, err := repo.SetNX(ctx, key, value, 0)
		require.NoError(t, err)
		assert.True(t, ok)

		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("should not set value when key exists", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "setnx-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := repo.Set(ctx, key, value1, 0)
		require.NoError(t, err)

		ok, err := repo.SetNX(ctx, key, value2, 0)
		require.NoError(t, err)
		assert.False(t, ok)

		// Original value should still be there
		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value1, result)
	})
}

func TestCacheRepository_Expire(t *testing.T) {
	ctx := context.Background()

	t.Run("should set expiration on existing key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "expire-key"
		value := []byte("expire-value")

		err := repo.Set(ctx, key, value, 0)
		require.NoError(t, err)

		err = repo.Expire(ctx, key, 1*time.Second)
		require.NoError(t, err)

		// Value should still exist
		result, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Fast-forward time
		mr.FastForward(2 * time.Second)

		result, err = repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCacheRepository_Increment(t *testing.T) {
	ctx := context.Background()

	t.Run("should increment non-existent key from zero", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		result, err := repo.Increment(ctx, "counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result)
	})

	t.Run("should increment existing counter", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "counter"

		result1, err := repo.Increment(ctx, key, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result1)

		result2, err := repo.Increment(ctx, key, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(15), result2)
	})

	t.Run("should handle negative increment", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "counter"

		result1, err := repo.Increment(ctx, key, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result1)

		result2, err := repo.Increment(ctx, key, -3)
		require.NoError(t, err)
		assert.Equal(t, int64(7), result2)
	})
}

func TestCacheRepository_DeletePattern(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete keys matching pattern", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		_ = repo.Set(ctx, "user:1", []byte("data1"), 0)
		_ = repo.Set(ctx, "user:2", []byte("data2"), 0)
		_ = repo.Set(ctx, "task:1", []byte("data3"), 0)

		err := repo.DeletePattern(ctx, "user:*")
		require.NoError(t, err)

		// User keys should be deleted
		result, _ := repo.Get(ctx, "user:1")
		assert.Nil(t, result)
		result, _ = repo.Get(ctx, "user:2")
		assert.Nil(t, result)

		// Task key should still exist
		result, _ = repo.Get(ctx, "task:1")
		assert.Equal(t, []byte("data3"), result)
	})
}

func TestCacheRepository_GetMultiple(t *testing.T) {
	ctx := context.Background()

	t.Run("should get multiple values", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		_ = repo.Set(ctx, "key1", []byte("value1"), 0)
		_ = repo.Set(ctx, "key2", []byte("value2"), 0)
		_ = repo.Set(ctx, "key3", []byte("value3"), 0)

		result, err := repo.GetMultiple(ctx, []string{"key1", "key2", "key4"})
		require.NoError(t, err)

		assert.Len(t, result, 2)
		assert.Equal(t, []byte("value1"), result["key1"])
		assert.Equal(t, []byte("value2"), result["key2"])
		_, exists := result["key4"]
		assert.False(t, exists)
	})

	t.Run("should return empty map for empty keys", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		result, err := repo.GetMultiple(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestCacheRepository_SetMultiple(t *testing.T) {
	ctx := context.Background()

	t.Run("should set multiple values", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		values := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
			"key3": []byte("value3"),
		}

		err := repo.SetMultiple(ctx, values, 0)
		require.NoError(t, err)

		for key, expected := range values {
			result, err := repo.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("should set multiple values with expiration", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		values := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}

		err := repo.SetMultiple(ctx, values, 1*time.Second)
		require.NoError(t, err)

		// Values should exist before expiration
		result, _ := repo.Get(ctx, "key1")
		assert.Equal(t, []byte("value1"), result)

		// Fast-forward time
		mr.FastForward(2 * time.Second)

		result, _ = repo.Get(ctx, "key1")
		assert.Nil(t, result)
	})
}

func TestCacheRepository_Health(t *testing.T) {
	ctx := context.Background()

	t.Run("should return nil when healthy", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		err := repo.Health(ctx)
		assert.NoError(t, err)
	})
}

func TestCacheRepository_JSON(t *testing.T) {
	ctx := context.Background()

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("should set and get JSON data", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		key := "json-key"
		input := TestData{Name: "test", Value: 42}

		err := repo.SetJSON(ctx, key, input, 0)
		require.NoError(t, err)

		var output TestData
		err = repo.GetJSON(ctx, key, &output)
		require.NoError(t, err)
		assert.Equal(t, input, output)
	})

	t.Run("should return error for non-existent JSON key", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()
		defer repo.Close()

		var output TestData
		err := repo.GetJSON(ctx, "non-existent", &output)
		assert.Error(t, err)
	})
}

func TestCacheRepository_Close(t *testing.T) {
	t.Run("should close without error", func(t *testing.T) {
		repo, mr := setupTestCacheRepository(t)
		defer mr.Close()

		err := repo.Close()
		assert.NoError(t, err)
	})
}
