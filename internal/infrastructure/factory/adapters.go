// Package factory provides adapter registrations.
package factory

import (
	"context"
	"log/slog"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	redisCache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/redis"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/kafka"
	s3Storage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/s3"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

func init() {
	// Register PostgreSQL adapter
	RegisterDatabase(DatabaseDriverPostgres, func(ctx context.Context, cfg any, logger *slog.Logger) (Database, error) {
		dbCfg, ok := cfg.(config.DatabaseConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return postgres.NewDatabase(ctx, dbCfg, logger)
	})

	// Register Redis cache adapter
	RegisterCache(CacheDriverRedis, func(ctx context.Context, cfg any, logger *slog.Logger) (output.CacheRepository, error) {
		redisCfg, ok := cfg.(config.RedisConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return redisCache.NewCacheRepository(ctx, redisCfg, logger)
	})

	// Register Kafka event publisher adapter
	RegisterEventPublisher(MessageDriverKafka, func(ctx context.Context, cfg any, logger *slog.Logger) (output.EventPublisher, error) {
		kafkaCfg, ok := cfg.(config.KafkaConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return kafka.NewEventPublisher(ctx, kafkaCfg, logger)
	})

	// Register Kafka event subscriber adapter
	RegisterEventSubscriber(MessageDriverKafka, func(ctx context.Context, cfg any, logger *slog.Logger) (output.EventSubscriber, error) {
		kafkaCfg, ok := cfg.(config.KafkaConfig)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return kafka.NewEventSubscriber(ctx, kafkaCfg, logger)
	})

	// Register S3 file storage adapter
	RegisterFileStorage(StorageDriverS3, func(ctx context.Context, cfg any, logger *slog.Logger) (output.FileStorage, error) {
		s3Cfg, ok := cfg.(config.S3Config)
		if !ok {
			return nil, ErrInvalidConfig
		}
		return s3Storage.NewFileStorage(ctx, s3Cfg, logger)
	})
}
