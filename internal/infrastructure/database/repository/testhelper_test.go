package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB holds the database connection and container for testing.
type TestDB struct {
	Pool      *pgxpool.Pool
	Container testcontainers.Container
}

// SetupTestDatabase creates a PostgreSQL testcontainer and returns a connection pool.
func SetupTestDatabase(t *testing.T) *TestDB {
	t.Helper()
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return &TestDB{
		Pool:      pool,
		Container: postgresContainer,
	}
}

// Cleanup closes the connection and stops the container.
func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()
	if tdb.Pool != nil {
		tdb.Pool.Close()
	}
	if tdb.Container != nil {
		if err := tdb.Container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}
}

// runMigrations executes the SQL migration file.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Find the migrations directory
	migrationPath := findMigrationsPath()
	if migrationPath == "" {
		return fmt.Errorf("migrations directory not found")
	}

	migrationSQL, err := os.ReadFile(filepath.Join(migrationPath, "000001_init.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// findMigrationsPath searches for the migrations directory.
func findMigrationsPath() string {
	// Try relative paths from the test location
	paths := []string{
		"../../../../migrations",
		"../../../migrations",
		"../../migrations",
		"../migrations",
		"migrations",
	}

	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "000001_init.up.sql")); err == nil {
			return p
		}
	}

	return ""
}

// CreateTestUser creates a user in the database for testing (required due to foreign key constraints).
func CreateTestUser(ctx context.Context, pool *pgxpool.Pool, t *testing.T) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, userID, fmt.Sprintf("test-%s@example.com", userID.String()[:8]), "hashedpassword", "Test User")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return userID
}

// CleanupTasks removes all tasks from the database.
func CleanupTasks(ctx context.Context, pool *pgxpool.Pool, t *testing.T) {
	t.Helper()
	_, err := pool.Exec(ctx, "DELETE FROM tasks")
	if err != nil {
		t.Logf("failed to cleanup tasks: %v", err)
	}
}
