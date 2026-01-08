package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, pool)

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}

	return pool, cleanup
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	migrationsPath := findMigrationsPath()
	files, err := os.ReadDir(migrationsPath)
	require.NoError(t, err)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" && filepath.Ext(file.Name()[:len(file.Name())-4]) == ".up" {
			content, err := os.ReadFile(filepath.Join(migrationsPath, file.Name()))
			require.NoError(t, err)
			_, err = pool.Exec(context.Background(), string(content))
			require.NoError(t, err, "Failed to run migration: %s", file.Name())
		}
	}
}

func findMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	for i := 0; i < 10; i++ {
		migrationsPath := filepath.Join(dir, "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}
		dir = filepath.Dir(dir)
	}

	panic("migrations directory not found")
}

func TestTransactionManager_Integration(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tm := postgres.NewTransactionManager(pool)

	t.Run("Commit updates from multiple repositories", func(t *testing.T) {
		labelID := uuid.New()
		labelName := "Test Label " + labelID.String()

		err := tm.RunInTransaction(ctx, func(uow output.UnitOfWork) error {
			labelRepo := uow.LabelRepository()

			// Save a label
			err := labelRepo.Save(ctx, &entity.Label{
				ID:        labelID,
				Name:      labelName,
				Color:     "#FF0000",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
			if err != nil {
				return err
			}

			return nil
		})

		assert.NoError(t, err)

		// Verify label exists
		labelRepo := repository.NewLabelRepository(pool)
		label, err := labelRepo.FindByID(ctx, labelID)
		assert.NoError(t, err)
		assert.NotNil(t, label)
		assert.Equal(t, labelName, label.Name)

		// Cleanup
		_ = labelRepo.Delete(ctx, labelID)
	})

	t.Run("Rollback on error", func(t *testing.T) {
		labelID := uuid.New()
		labelName := "Rollback Label"

		err := tm.RunInTransaction(ctx, func(uow output.UnitOfWork) error {
			labelRepo := uow.LabelRepository()

			err := labelRepo.Save(ctx, &entity.Label{
				ID:        labelID,
				Name:      labelName,
				Color:     "#FF0000",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
			if err != nil {
				return err
			}

			return assert.AnError // Force rollback
		})

		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)

		// Verify label DOES NOT exist
		labelRepo := repository.NewLabelRepository(pool)
		label, err := labelRepo.FindByID(ctx, labelID)
		assert.NoError(t, err)
		assert.Nil(t, label)
	})
}
