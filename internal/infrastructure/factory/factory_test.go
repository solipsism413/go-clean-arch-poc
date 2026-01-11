// Package factory_test contains tests for the infrastructure factory.
package factory_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/factory"
	localStorage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/local"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDriverType(t *testing.T) {
	t.Run("should have correct database driver constants", func(t *testing.T) {
		assert.Equal(t, factory.DriverType("postgres"), factory.DatabaseDriverPostgres)
	})

	t.Run("should have correct cache driver constants", func(t *testing.T) {
		assert.Equal(t, factory.DriverType("redis"), factory.CacheDriverRedis)
		assert.Equal(t, factory.DriverType("memory"), factory.CacheDriverMemory)
	})

	t.Run("should have correct message driver constants", func(t *testing.T) {
		assert.Equal(t, factory.DriverType("kafka"), factory.MessageDriverKafka)
		assert.Equal(t, factory.DriverType("memory"), factory.MessageDriverMemory)
	})

	t.Run("should have correct storage driver constants", func(t *testing.T) {
		assert.Equal(t, factory.DriverType("s3"), factory.StorageDriverS3)
		assert.Equal(t, factory.DriverType("local"), factory.StorageDriverLocal)
	})
}

func TestCreateDatabase(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should return error for unknown driver", func(t *testing.T) {
		db, err := factory.CreateDatabase(ctx, "unknown", nil, logger)

		assert.Error(t, err)
		assert.Nil(t, db)
		assert.ErrorIs(t, err, factory.ErrDriverNotFound)
	})

	t.Run("should return error for invalid postgres config", func(t *testing.T) {
		db, err := factory.CreateDatabase(ctx, factory.DatabaseDriverPostgres, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, db)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})
}

func TestCreateCache(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should create memory cache successfully", func(t *testing.T) {
		cache, err := factory.CreateCache(ctx, factory.CacheDriverMemory, nil, logger)

		require.NoError(t, err)
		assert.NotNil(t, cache)
	})

	t.Run("should return error for unknown driver", func(t *testing.T) {
		cache, err := factory.CreateCache(ctx, "unknown", nil, logger)

		assert.Error(t, err)
		assert.Nil(t, cache)
		assert.ErrorIs(t, err, factory.ErrDriverNotFound)
	})

	t.Run("should return error for invalid redis config", func(t *testing.T) {
		cache, err := factory.CreateCache(ctx, factory.CacheDriverRedis, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, cache)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})
}

func TestCreateEventPublisher(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should create memory event publisher successfully", func(t *testing.T) {
		publisher, err := factory.CreateEventPublisher(ctx, factory.MessageDriverMemory, nil, logger)

		require.NoError(t, err)
		assert.NotNil(t, publisher)
	})

	t.Run("should return error for unknown driver", func(t *testing.T) {
		publisher, err := factory.CreateEventPublisher(ctx, "unknown", nil, logger)

		assert.Error(t, err)
		assert.Nil(t, publisher)
		assert.ErrorIs(t, err, factory.ErrDriverNotFound)
	})

	t.Run("should return error for invalid kafka config", func(t *testing.T) {
		publisher, err := factory.CreateEventPublisher(ctx, factory.MessageDriverKafka, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, publisher)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})
}

func TestCreateEventSubscriber(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should create memory event subscriber successfully", func(t *testing.T) {
		subscriber, err := factory.CreateEventSubscriber(ctx, factory.MessageDriverMemory, nil, logger)

		require.NoError(t, err)
		assert.NotNil(t, subscriber)
	})

	t.Run("should return error for unknown driver", func(t *testing.T) {
		subscriber, err := factory.CreateEventSubscriber(ctx, "unknown", nil, logger)

		assert.Error(t, err)
		assert.Nil(t, subscriber)
		assert.ErrorIs(t, err, factory.ErrDriverNotFound)
	})

	t.Run("should return error for invalid kafka config", func(t *testing.T) {
		subscriber, err := factory.CreateEventSubscriber(ctx, factory.MessageDriverKafka, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, subscriber)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})
}

func TestCreateFileStorage(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should create local file storage successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := localStorage.Config{
			BasePath: tempDir,
			BaseURL:  "http://localhost:8080/files",
		}

		storage, err := factory.CreateFileStorage(ctx, factory.StorageDriverLocal, cfg, logger)

		require.NoError(t, err)
		assert.NotNil(t, storage)
	})

	t.Run("should return error for unknown driver", func(t *testing.T) {
		storage, err := factory.CreateFileStorage(ctx, "unknown", nil, logger)

		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.ErrorIs(t, err, factory.ErrDriverNotFound)
	})

	t.Run("should return error for invalid local config", func(t *testing.T) {
		storage, err := factory.CreateFileStorage(ctx, factory.StorageDriverLocal, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})

	t.Run("should return error for invalid S3 config", func(t *testing.T) {
		storage, err := factory.CreateFileStorage(ctx, factory.StorageDriverS3, "invalid-config", logger)

		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.ErrorIs(t, err, factory.ErrInvalidConfig)
	})
}

func TestErrors(t *testing.T) {
	t.Run("ErrInvalidConfig should have correct message", func(t *testing.T) {
		assert.Equal(t, "invalid configuration type", factory.ErrInvalidConfig.Error())
	})

	t.Run("ErrDriverNotFound should have correct message", func(t *testing.T) {
		assert.Equal(t, "driver not found", factory.ErrDriverNotFound.Error())
	})
}

func TestInfrastructureConfig(t *testing.T) {
	t.Run("should create InfrastructureConfig with all fields", func(t *testing.T) {
		cfg := factory.InfrastructureConfig{
			DatabaseDriver: factory.DatabaseDriverPostgres,
			CacheDriver:    factory.CacheDriverRedis,
			MessageDriver:  factory.MessageDriverKafka,
			StorageDriver:  factory.StorageDriverS3,
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Name:     "testdb",
			},
			Redis: config.RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
			},
			Kafka: config.KafkaConfig{
				Brokers: []string{"localhost:9092"},
			},
			S3: config.S3Config{
				Endpoint:        "http://localhost:9000",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		}

		assert.Equal(t, factory.DatabaseDriverPostgres, cfg.DatabaseDriver)
		assert.Equal(t, factory.CacheDriverRedis, cfg.CacheDriver)
		assert.Equal(t, factory.MessageDriverKafka, cfg.MessageDriver)
		assert.Equal(t, factory.StorageDriverS3, cfg.StorageDriver)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, "localhost:6379", cfg.Redis.Addr())
		assert.Equal(t, []string{"localhost:9092"}, cfg.Kafka.Brokers)
		assert.Equal(t, "test-bucket", cfg.S3.Bucket)
	})
}

func TestInfrastructure(t *testing.T) {
	t.Run("should create Infrastructure struct", func(t *testing.T) {
		infra := factory.Infrastructure{
			Logger: getTestLogger(),
		}

		assert.NotNil(t, infra.Logger)
		assert.Nil(t, infra.Database)
		assert.Nil(t, infra.Cache)
		assert.Nil(t, infra.EventPublisher)
		assert.Nil(t, infra.EventSubscriber)
		assert.Nil(t, infra.FileStorage)
	})
}
