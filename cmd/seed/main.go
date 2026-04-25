package main

import (
	"context"
	"fmt"
	"os"

	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/seed"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/logger"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	})

	db, err := postgres.NewDatabase(ctx, cfg.Database, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed.SeedDevelopmentData(ctx, db, log); err != nil {
		log.Error("failed to seed database", "error", err)
		os.Exit(1)
	}

	log.Info("database seed completed successfully")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
