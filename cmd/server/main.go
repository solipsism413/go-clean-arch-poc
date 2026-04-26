// Package main is the entry point for the Task Manager server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	authUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/auth"
	labelUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/label"
	roleUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/rbac"
	taskUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/task"
	userUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/application/worker"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/auth/rbac"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	redisCache "github.com/handiism/go-clean-arch-poc/internal/infrastructure/cache/redis"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/kafka"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/logger"
	s3Storage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/s3"
	"github.com/handiism/go-clean-arch-poc/internal/transport/graphql"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest"
	"github.com/handiism/go-clean-arch-poc/internal/transport/socketio"
	"github.com/handiism/go-clean-arch-poc/internal/transport/sse"
	"github.com/handiism/go-clean-arch-poc/internal/transport/websocket"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

// @title Task Manager API
// @version 1.0
// @description A Task Management Application built with Hexagonal Architecture in Go.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@taskmanager.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

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

	log.Info("starting task manager server",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
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

	// Initialize Kafka event subscriber for background consumers
	var eventSubscriber output.EventSubscriber
	sub, err := kafka.NewEventSubscriber(ctx, cfg.Kafka, log)
	if err != nil {
		log.Warn("failed to initialize kafka event subscriber, background consumers will not run", "error", err)
	} else {
		eventSubscriber = sub
		log.Info("connected to kafka event subscriber")
		defer sub.Close()
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
	aclRepo := repository.NewACLRepository(db.Pool)

	// Initialize authorizer and ACL checker
	authorizer := rbac.NewAuthorizer()
	aclChecker := acl.NewChecker(aclRepo)

	// Initialize validator
	v := validation.NewValidator()

	// Initialize use cases
	taskService := taskUseCase.NewTaskUseCase(taskRepo, taskRepo, userRepo, labelRepo, fileStorage, cacheRepo, eventPublisher, tm, v, log)
	userService := userUseCase.NewUserUseCase(userRepo, roleRepo, cacheRepo, eventPublisher, tm, v, log)
	authService := authUseCase.NewAuthUseCase(userRepo, roleRepo, cacheRepo, eventPublisher, tm, tokenService, v, log)
	labelService := labelUseCase.NewLabelUseCase(labelRepo, v, log)
	roleService := roleUseCase.NewRoleUseCase(roleRepo, log)

	// Initialize background event consumer
	eventConsumer := worker.NewEventConsumer(log)
	eventConsumer.RegisterHandler("task.created", func(ctx context.Context, evt event.Event) error {
		log.Info("background handler: task created", "taskID", evt.AggregateID())
		return nil
	})
	eventConsumer.RegisterHandler("task.updated", func(ctx context.Context, evt event.Event) error {
		log.Info("background handler: task updated", "taskID", evt.AggregateID())
		return nil
	})
	eventConsumer.RegisterHandler("task.attachment_cleanup_requested", func(ctx context.Context, evt event.Event) error {
		cleanupEvent, ok := evt.(*event.TaskAttachmentCleanupRequested)
		if !ok {
			return fmt.Errorf("unexpected event type %T", evt)
		}
		if fileStorage == nil {
			return fmt.Errorf("file storage unavailable for cleanup retry")
		}
		if err := fileStorage.Delete(ctx, cleanupEvent.ObjectKey); err != nil {
			return err
		}
		log.Info("background handler: attachment cleanup completed", "taskID", cleanupEvent.AggregateID(), "attachmentId", cleanupEvent.AttachmentID)
		return nil
	})
	eventConsumer.RegisterHandler("user.created", func(ctx context.Context, evt event.Event) error {
		log.Info("background handler: user created", "userID", evt.AggregateID())
		return nil
	})
	eventConsumer.RegisterHandler("user.logged_in", func(ctx context.Context, evt event.Event) error {
		log.Info("background handler: user logged in", "userID", evt.AggregateID())
		return nil
	})

	if eventSubscriber != nil {
		if err := eventConsumer.Start(ctx, eventSubscriber, []string{output.TopicTaskEvents, output.TopicUserEvents}); err != nil {
			log.Warn("failed to start event consumer", "error", err)
		} else {
			log.Info("background event consumer started")
			defer eventConsumer.Stop()
		}
	}

	// Initialize auth middleware
	authMiddleware := auth.NewMiddleware(authService, userService, authorizer, aclChecker)

	// Seed system roles
	if err := userService.SeedSystemRoles(ctx); err != nil {
		log.Error("failed to seed system roles", "error", err)
		os.Exit(1)
	}
	log.Info("system roles seeded successfully")

	// =====================
	// Initialize WebSocket
	// =====================
	wsHub := websocket.NewHub(log)
	go wsHub.Run(ctx)
	log.Info("websocket hub started")

	// =====================
	// Initialize SSE
	// =====================
	sseBroker := sse.NewBroker(log)
	go sseBroker.Run(ctx)
	log.Info("sse broker started")

	// =====================
	// Initialize Socket.IO
	// =====================
	socketIOHandler, err := socketio.NewHandler(taskService, log)
	if err != nil {
		log.Warn("failed to initialize socket.io", "error", err)
	} else {
		if err := socketIOHandler.Start(ctx); err != nil {
			log.Warn("failed to start socket.io", "error", err)
		} else {
			log.Info("socket.io started")
		}
	}

	// =====================
	// Initialize REST Router
	// =====================
	router := rest.NewRouter(taskService, userService, authService, labelService, authMiddleware, aclChecker, log)

	// =====================
	// Initialize GraphQL
	// =====================
	graphQLResolver := graphql.NewResolver(taskService, userService, authService, labelService, roleService, authorizer)
	graphQLHandler := graphql.NewHandler(graphQLResolver, authService)
	router.Handle("POST /graphql", graphQLHandler)
	router.Handle("GET /graphql", graphQLHandler)
	router.Handle("GET /graphql/playground", graphql.NewPlaygroundHandler("/graphql"))

	// Register WebSocket handler
	router.Handle("GET /ws", websocket.NewHandler(wsHub, taskService, authService, log))

	// Register SSE handler
	router.Handle("GET /events", sse.NewHandler(sseBroker, log))

	// Register Socket.IO handler
	if socketIOHandler != nil {
		router.Handle("/socket.io/", socketIOHandler)
	}

	// =====================
	// Start HTTP Server
	// =====================
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Print transport endpoints
	log.Info("=== Transport Endpoints ===")
	log.Info("REST API", "url", fmt.Sprintf("http://%s/api/v1", server.Addr))
	log.Info("GraphQL", "url", fmt.Sprintf("http://%s/graphql", server.Addr))
	log.Info("GraphQL Playground", "url", fmt.Sprintf("http://%s/graphql/playground", server.Addr))
	log.Info("WebSocket", "url", fmt.Sprintf("ws://%s/ws", server.Addr))
	log.Info("SSE", "url", fmt.Sprintf("http://%s/events", server.Addr))
	log.Info("Socket.IO", "url", fmt.Sprintf("http://%s/socket.io/", server.Addr))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down servers...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("servers exited gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
