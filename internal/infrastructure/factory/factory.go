// Package factory provides infrastructure factory for creating adapters.
// This enables easy swapping of implementations (PostgreSQL/MySQL, Redis/Memory, Kafka/MQTT, etc.)
package factory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	memoryCache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/memory"
	redisCache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/redis"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/kafka"
	memoryMessaging "github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/memory"
	localStorage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/local"
	s3Storage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/s3"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

// DriverType represents the type of infrastructure driver.
type DriverType string

// Database driver types
const (
	DatabaseDriverPostgres DriverType = "postgres"
)

// Cache driver types
const (
	CacheDriverRedis  DriverType = "redis"
	CacheDriverMemory DriverType = "memory"
)

// Message broker driver types
const (
	MessageDriverKafka  DriverType = "kafka"
	MessageDriverMemory DriverType = "memory"
)

// Storage driver types
const (
	StorageDriverS3    DriverType = "s3"
	StorageDriverLocal DriverType = "local"
)

// InfrastructureConfig holds configuration for all infrastructure components.
type InfrastructureConfig struct {
	DatabaseDriver DriverType
	CacheDriver    DriverType
	MessageDriver  DriverType
	StorageDriver  DriverType

	// Configurations
	Database config.DatabaseConfig
	Redis    config.RedisConfig
	Kafka    config.KafkaConfig
	S3       config.S3Config
}

// Infrastructure holds all infrastructure adapters.
type Infrastructure struct {
	Database        Database
	Cache           output.CacheRepository
	EventPublisher  output.EventPublisher
	EventSubscriber output.EventSubscriber
	FileStorage     output.FileStorage
	Logger          *slog.Logger
}

// Database interface abstracts the database connection.
type Database interface {
	// GetPool returns the underlying connection pool (type-specific)
	GetPool() any
	// Close closes the database connection
	Close()
	// Health checks database connectivity
	Health(ctx context.Context) error
}

// CreateDatabase creates a database adapter.
func CreateDatabase(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (Database, error) {
	switch driver {
	case DatabaseDriverPostgres:
		dbCfg, ok := cfg.(config.DatabaseConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return postgres.NewDatabase(ctx, dbCfg, logger)
	default:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotFound, driver)
	}
}

// CreateCache creates a cache adapter.
func CreateCache(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.CacheRepository, error) {
	switch driver {
	case CacheDriverRedis:
		redisCfg, ok := cfg.(config.RedisConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return redisCache.NewCacheRepository(ctx, redisCfg, logger)
	case CacheDriverMemory:
		return memoryCache.NewMemoryCache(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotFound, driver)
	}
}

// CreateEventPublisher creates an event publisher adapter.
func CreateEventPublisher(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.EventPublisher, error) {
	switch driver {
	case MessageDriverKafka:
		kafkaCfg, ok := cfg.(config.KafkaConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return kafka.NewEventPublisher(ctx, kafkaCfg, logger)
	case MessageDriverMemory:
		return memoryMessaging.NewEventBus(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotFound, driver)
	}
}

// CreateEventSubscriber creates an event subscriber adapter.
func CreateEventSubscriber(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.EventSubscriber, error) {
	switch driver {
	case MessageDriverKafka:
		kafkaCfg, ok := cfg.(config.KafkaConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return kafka.NewEventSubscriber(ctx, kafkaCfg, logger)
	case MessageDriverMemory:
		return memoryMessaging.NewEventBus(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotFound, driver)
	}
}

// CreateFileStorage creates a file storage adapter.
func CreateFileStorage(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.FileStorage, error) {
	switch driver {
	case StorageDriverS3:
		s3Cfg, ok := cfg.(config.S3Config)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return s3Storage.NewFileStorage(ctx, s3Cfg, logger)
	case StorageDriverLocal:
		localCfg, ok := cfg.(localStorage.Config)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return localStorage.NewLocalStorage(localCfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrDriverNotFound, driver)
	}
}
