// Package factory provides infrastructure factory for creating adapters.
// This enables easy swapping of implementations (PostgreSQL/MySQL, Redis/Memory, Kafka/MQTT, etc.)
package factory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

// DriverType represents the type of infrastructure driver.
type DriverType string

// Database driver types
const (
	DatabaseDriverPostgres DriverType = "postgres"
	DatabaseDriverMySQL    DriverType = "mysql"
	DatabaseDriverSQLite   DriverType = "sqlite"
)

// Cache driver types
const (
	CacheDriverRedis  DriverType = "redis"
	CacheDriverMemory DriverType = "memory"
)

// Message broker driver types
const (
	MessageDriverKafka  DriverType = "kafka"
	MessageDriverMQTT   DriverType = "mqtt"
	MessageDriverNats   DriverType = "nats"
	MessageDriverMemory DriverType = "memory"
)

// Storage driver types
const (
	StorageDriverS3        DriverType = "s3"
	StorageDriverLocal     DriverType = "local"
	StorageDriverGCS       DriverType = "gcs"
	StorageDriverAzureBlob DriverType = "azure"
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

// Factory interface for creating infrastructure components.
type Factory interface {
	CreateDatabase(ctx context.Context, driver DriverType, cfg any) (Database, error)
	CreateCache(ctx context.Context, driver DriverType, cfg any) (output.CacheRepository, error)
	CreateEventPublisher(ctx context.Context, driver DriverType, cfg any) (output.EventPublisher, error)
	CreateEventSubscriber(ctx context.Context, driver DriverType, cfg any) (output.EventSubscriber, error)
	CreateFileStorage(ctx context.Context, driver DriverType, cfg any) (output.FileStorage, error)
}

// registry holds registered adapters
var registry = &adapterRegistry{
	databases:        make(map[DriverType]DatabaseFactory),
	caches:           make(map[DriverType]CacheFactory),
	eventPublishers:  make(map[DriverType]EventPublisherFactory),
	eventSubscribers: make(map[DriverType]EventSubscriberFactory),
	fileStorages:     make(map[DriverType]FileStorageFactory),
}

// Factory function types
type DatabaseFactory func(ctx context.Context, cfg any, logger *slog.Logger) (Database, error)
type CacheFactory func(ctx context.Context, cfg any, logger *slog.Logger) (output.CacheRepository, error)
type EventPublisherFactory func(ctx context.Context, cfg any, logger *slog.Logger) (output.EventPublisher, error)
type EventSubscriberFactory func(ctx context.Context, cfg any, logger *slog.Logger) (output.EventSubscriber, error)
type FileStorageFactory func(ctx context.Context, cfg any, logger *slog.Logger) (output.FileStorage, error)

type adapterRegistry struct {
	databases        map[DriverType]DatabaseFactory
	caches           map[DriverType]CacheFactory
	eventPublishers  map[DriverType]EventPublisherFactory
	eventSubscribers map[DriverType]EventSubscriberFactory
	fileStorages     map[DriverType]FileStorageFactory
}

// RegisterDatabase registers a database adapter factory.
func RegisterDatabase(driver DriverType, factory DatabaseFactory) {
	registry.databases[driver] = factory
}

// RegisterCache registers a cache adapter factory.
func RegisterCache(driver DriverType, factory CacheFactory) {
	registry.caches[driver] = factory
}

// RegisterEventPublisher registers an event publisher adapter factory.
func RegisterEventPublisher(driver DriverType, factory EventPublisherFactory) {
	registry.eventPublishers[driver] = factory
}

// RegisterEventSubscriber registers an event subscriber adapter factory.
func RegisterEventSubscriber(driver DriverType, factory EventSubscriberFactory) {
	registry.eventSubscribers[driver] = factory
}

// RegisterFileStorage registers a file storage adapter factory.
func RegisterFileStorage(driver DriverType, factory FileStorageFactory) {
	registry.fileStorages[driver] = factory
}

// CreateDatabase creates a database adapter using the registered factory.
func CreateDatabase(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (Database, error) {
	factory, ok := registry.databases[driver]
	if !ok {
		return nil, fmt.Errorf("database driver not registered: %s", driver)
	}
	return factory(ctx, cfg, logger)
}

// CreateCache creates a cache adapter using the registered factory.
func CreateCache(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.CacheRepository, error) {
	factory, ok := registry.caches[driver]
	if !ok {
		return nil, fmt.Errorf("cache driver not registered: %s", driver)
	}
	return factory(ctx, cfg, logger)
}

// CreateEventPublisher creates an event publisher adapter using the registered factory.
func CreateEventPublisher(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.EventPublisher, error) {
	factory, ok := registry.eventPublishers[driver]
	if !ok {
		return nil, fmt.Errorf("event publisher driver not registered: %s", driver)
	}
	return factory(ctx, cfg, logger)
}

// CreateEventSubscriber creates an event subscriber adapter using the registered factory.
func CreateEventSubscriber(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.EventSubscriber, error) {
	factory, ok := registry.eventSubscribers[driver]
	if !ok {
		return nil, fmt.Errorf("event subscriber driver not registered: %s", driver)
	}
	return factory(ctx, cfg, logger)
}

// CreateFileStorage creates a file storage adapter using the registered factory.
func CreateFileStorage(ctx context.Context, driver DriverType, cfg any, logger *slog.Logger) (output.FileStorage, error) {
	factory, ok := registry.fileStorages[driver]
	if !ok {
		return nil, fmt.Errorf("file storage driver not registered: %s", driver)
	}
	return factory(ctx, cfg, logger)
}

// ListRegisteredDrivers returns all registered drivers for each component type.
func ListRegisteredDrivers() map[string][]DriverType {
	result := make(map[string][]DriverType)

	for driver := range registry.databases {
		result["database"] = append(result["database"], driver)
	}
	for driver := range registry.caches {
		result["cache"] = append(result["cache"], driver)
	}
	for driver := range registry.eventPublishers {
		result["event_publisher"] = append(result["event_publisher"], driver)
	}
	for driver := range registry.eventSubscribers {
		result["event_subscriber"] = append(result["event_subscriber"], driver)
	}
	for driver := range registry.fileStorages {
		result["file_storage"] = append(result["file_storage"], driver)
	}

	return result
}
