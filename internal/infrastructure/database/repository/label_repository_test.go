package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	t.Run("create label", func(t *testing.T) {
		label := &entity.Label{
			ID:        uuid.New(),
			Name:      "Bug",
			Color:     "#FF0000",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}

		err := repo.Save(ctx, label)
		require.NoError(t, err)

		saved, err := repo.FindByID(ctx, label.ID)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Equal(t, label.Name, saved.Name)
		assert.Equal(t, label.Color, saved.Color)
	})
}

func TestLabelRepository_Update(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	label := &entity.Label{
		ID:        uuid.New(),
		Name:      "Original",
		Color:     "#000000",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, label))

	label.Name = "Updated"
	label.Color = "#FFFFFF"
	err := repo.Update(ctx, label)
	require.NoError(t, err)

	updated, err := repo.FindByID(ctx, label.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "#FFFFFF", updated.Color)
}

func TestLabelRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	label := &entity.Label{
		ID:        uuid.New(),
		Name:      "ToDelete",
		Color:     "#FF0000",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, label))

	err := repo.Delete(ctx, label.ID)
	require.NoError(t, err)

	deleted, err := repo.FindByID(ctx, label.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestLabelRepository_FindByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	t.Run("find existing", func(t *testing.T) {
		label := &entity.Label{
			ID:        uuid.New(),
			Name:      "Feature",
			Color:     "#00FF00",
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, label))

		found, err := repo.FindByID(ctx, label.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, label.ID, found.ID)
	})

	t.Run("find non-existent returns nil", func(t *testing.T) {
		found, err := repo.FindByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestLabelRepository_FindAll(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	// Clean existing labels
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM labels")

	labels := []*entity.Label{
		{ID: uuid.New(), Name: "Bug", Color: "#FF0000", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "Feature", Color: "#00FF00", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "Enhancement", Color: "#0000FF", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	for _, label := range labels {
		require.NoError(t, repo.Save(ctx, label))
	}

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestLabelRepository_ExistsByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewLabelRepository(testDB.Pool)

	label := &entity.Label{
		ID:        uuid.New(),
		Name:      "Exists",
		Color:     "#AABBCC",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, label))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, label.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists returns false", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLabelRepository_FindByTaskID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	labelRepo := repository.NewLabelRepository(testDB.Pool)
	taskRepo := repository.NewTaskRepository(testDB.Pool)

	// Create user for task
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	// Create labels
	label1 := &entity.Label{ID: uuid.New(), Name: "Bug", Color: "#FF0000", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	label2 := &entity.Label{ID: uuid.New(), Name: "Feature", Color: "#00FF00", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, labelRepo.Save(ctx, label1))
	require.NoError(t, labelRepo.Save(ctx, label2))

	// Create task
	task := &entity.Task{
		ID:        uuid.New(),
		Title:     "Task with labels",
		Status:    "TODO",
		Priority:  "MEDIUM",
		CreatorID: creatorID,
		Labels:    []uuid.UUID{},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, taskRepo.Save(ctx, task))

	// Associate labels with task
	_, err := testDB.Pool.Exec(ctx, "INSERT INTO task_labels (task_id, label_id) VALUES ($1, $2), ($1, $3)",
		task.ID, label1.ID, label2.ID)
	require.NoError(t, err)

	// Find labels by task ID
	labels, err := labelRepo.FindByTaskID(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, labels, 2)
}
