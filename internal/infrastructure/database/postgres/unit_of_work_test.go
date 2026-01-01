package postgres_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager_Integration(t *testing.T) {
	// Skip if not in a CI environment or manually enabled
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg, err := config.Load()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := postgres.NewDatabase(ctx, cfg.Database, logger)
	require.NoError(t, err)
	defer db.Close()

	tm := postgres.NewTransactionManager(db.Pool)

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

			// Save another entity or update something...
			return nil
		})

		assert.NoError(t, err)

		// Verify label exists
		labelRepo := repository.NewLabelRepository(db.Pool)
		label, err := labelRepo.FindByID(ctx, labelID)
		assert.NoError(t, err)
		assert.NotNil(t, label)
		assert.Equal(t, labelName, label.Name)

		// Cleanup
		_ = labelRepo.Delete(ctx, labelID)
	})

	t.Run("Rollback on error", func(t *testing.T) {
		labelID := uuid.New()
		labelName := "Rollback Label " + labelID.String()

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
		labelRepo := repository.NewLabelRepository(db.Pool)
		label, err := labelRepo.FindByID(ctx, labelID)
		assert.NoError(t, err)
		assert.Nil(t, label)
	})
}
