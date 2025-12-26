// Package postgres provides PostgreSQL database connection using pgx.
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database wraps the pgx connection pool.
type Database struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewDatabase creates a new PostgreSQL database connection pool.
func NewDatabase(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*Database, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure pool
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Name,
	)

	return &Database{
		Pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the database connection pool.
func (db *Database) Close() {
	db.Pool.Close()
	db.logger.Info("database connection closed")
}

// GetPool returns the underlying connection pool.
func (db *Database) GetPool() any {
	return db.Pool
}

// Health checks if the database is healthy.
func (db *Database) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics.
func (db *Database) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
