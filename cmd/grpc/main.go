// Package main is the entry point for the Task Manager gRPC server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	authUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/auth"
	labelUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/label"
	taskUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/task"
	userUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	redisCache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/redis"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/kafka"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/logger"
	s3Storage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/s3"
	grpcTransport "github.com/handiism/go-clean-arch-poc/internal/transport/grpc"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	log := logger.New(logger.Config{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	})
	slog.SetDefault(log)

	log.Info("starting task manager gRPC server",
		"host", cfg.Server.Host,
		"port", cfg.GRPC.Port,
	)

	// Initialize database connection
	db, err := postgres.NewDatabase(ctx, cfg.Database, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to database")

	// Initialize Redis cache
	cacheRepo, err := redisCache.NewCacheRepository(ctx, cfg.Redis, log)
	if err != nil {
		log.Warn("failed to connect to redis, cache will be unavailable", "error", err)
	} else {
		log.Info("connected to redis")
		defer cacheRepo.Close()
	}

	// Initialize Kafka event publisher
	eventPublisher, err := kafka.NewEventPublisher(ctx, cfg.Kafka, log)
	if err != nil {
		log.Warn("failed to initialize kafka, events will not be published", "error", err)
	} else {
		log.Info("connected to kafka")
		defer eventPublisher.Close()
	}

	// Initialize S3 file storage
	fileStorage, err := s3Storage.NewFileStorage(ctx, cfg.S3, log)
	if err != nil {
		log.Warn("failed to initialize s3, file storage will be unavailable", "error", err)
	} else {
		log.Info("connected to s3")
	}
	_ = fileStorage // Keep reference for future use

	// Initialize JWT token service
	tokenService := jwt.NewTokenService(cfg.JWT)

	// Initialize transaction manager
	tm := postgres.NewTransactionManager(db.Pool)

	// Initialize repositories
	taskRepo := repository.NewTaskRepository(db.Pool)
	userRepo := repository.NewUserRepository(db.Pool)
	roleRepo := repository.NewRoleRepository(db.Pool)
	labelRepo := repository.NewLabelRepository(db.Pool)

	// Initialize validator
	v := validation.NewValidator()

	// Initialize use cases
	taskService := taskUseCase.NewTaskUseCase(taskRepo, userRepo, labelRepo, cacheRepo, eventPublisher, tm, v, log)
	userService := userUseCase.NewUserUseCase(userRepo, roleRepo, cacheRepo, eventPublisher, tm, v, log)
	authService := authUseCase.NewAuthUseCase(userRepo, roleRepo, cacheRepo, eventPublisher, tm, tokenService, v, log)
	labelService := labelUseCase.NewLabelUseCase(labelRepo, v, log)

	// =====================
	// Initialize gRPC Server
	// =====================
	grpcServer := grpcTransport.NewServer(log)
	grpcTransport.RegisterApplicationServices(grpcServer.GRPCServer(), taskService, userService, authService, labelService)

	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.GRPC.Port)
	if err := grpcServer.Start(ctx, grpcAddr); err != nil {
		log.Error("failed to start gRPC server", "error", err)
		os.Exit(1)
	}
	log.Info("grpc server started", "addr", grpcAddr)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gRPC server...")

	grpcServer.Stop()

	log.Info("server exited gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
