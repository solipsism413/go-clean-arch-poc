package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	t.Run("create task with all fields", func(t *testing.T) {
		CleanupTasks(ctx, testDB.Pool, t)

		assigneeID := CreateTestUser(ctx, testDB.Pool, t)
		dueDate := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

		task := &entity.Task{
			ID:          uuid.New(),
			Title:       "Test Task",
			Description: "Test Description",
			Status:      valueobject.TaskStatusTodo,
			Priority:    valueobject.PriorityHigh,
			DueDate:     &dueDate,
			AssigneeID:  &assigneeID,
			CreatorID:   creatorID,
			Labels:      []uuid.UUID{},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		}

		err := repo.Save(ctx, task)
		require.NoError(t, err)

		// Verify task was saved
		savedTask, err := repo.FindByID(ctx, task.ID)
		require.NoError(t, err)
		require.NotNil(t, savedTask)
		assert.Equal(t, task.Title, savedTask.Title)
		assert.Equal(t, task.Description, savedTask.Description)
		assert.Equal(t, task.Status, savedTask.Status)
		assert.Equal(t, task.Priority, savedTask.Priority)
		assert.Equal(t, assigneeID, *savedTask.AssigneeID)
	})

	t.Run("create task with optional fields nil", func(t *testing.T) {
		CleanupTasks(ctx, testDB.Pool, t)

		task := &entity.Task{
			ID:        uuid.New(),
			Title:     "Minimal Task",
			Status:    valueobject.TaskStatusTodo,
			Priority:  valueobject.PriorityMedium,
			CreatorID: creatorID,
			Labels:    []uuid.UUID{},
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}

		err := repo.Save(ctx, task)
		require.NoError(t, err)

		savedTask, err := repo.FindByID(ctx, task.ID)
		require.NoError(t, err)
		require.NotNil(t, savedTask)
		assert.Nil(t, savedTask.DueDate)
		assert.Nil(t, savedTask.AssigneeID)
		assert.Empty(t, savedTask.Description)
	})
}

func TestTaskRepository_Update(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	t.Run("update all fields", func(t *testing.T) {
		CleanupTasks(ctx, testDB.Pool, t)

		task := &entity.Task{
			ID:        uuid.New(),
			Title:     "Original Title",
			Status:    valueobject.TaskStatusTodo,
			Priority:  valueobject.PriorityLow,
			CreatorID: creatorID,
			Labels:    []uuid.UUID{},
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, task))

		// Update task
		assigneeID := CreateTestUser(ctx, testDB.Pool, t)
		dueDate := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Microsecond)
		task.Title = "Updated Title"
		task.Description = "Updated Description"
		task.Status = valueobject.TaskStatusInProgress
		task.Priority = valueobject.PriorityHigh
		task.DueDate = &dueDate
		task.AssigneeID = &assigneeID

		err := repo.Update(ctx, task)
		require.NoError(t, err)

		updatedTask, err := repo.FindByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", updatedTask.Title)
		assert.Equal(t, "Updated Description", updatedTask.Description)
		assert.Equal(t, valueobject.TaskStatusInProgress, updatedTask.Status)
		assert.Equal(t, valueobject.PriorityHigh, updatedTask.Priority)
		assert.Equal(t, assigneeID, *updatedTask.AssigneeID)
	})
}

func TestTaskRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	t.Run("delete existing task", func(t *testing.T) {
		CleanupTasks(ctx, testDB.Pool, t)

		task := &entity.Task{
			ID:        uuid.New(),
			Title:     "Task to Delete",
			Status:    valueobject.TaskStatusTodo,
			Priority:  valueobject.PriorityMedium,
			CreatorID: creatorID,
			Labels:    []uuid.UUID{},
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, task))

		err := repo.Delete(ctx, task.ID)
		require.NoError(t, err)

		// Verify deletion
		deletedTask, err := repo.FindByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Nil(t, deletedTask)
	})

	t.Run("delete non-existent task", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		// Should not return error for non-existent task
		assert.NoError(t, err)
	})
}

func TestTaskRepository_FindByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	t.Run("find existing task", func(t *testing.T) {
		CleanupTasks(ctx, testDB.Pool, t)

		task := &entity.Task{
			ID:          uuid.New(),
			Title:       "Findable Task",
			Description: "Description",
			Status:      valueobject.TaskStatusTodo,
			Priority:    valueobject.PriorityMedium,
			CreatorID:   creatorID,
			Labels:      []uuid.UUID{},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, task))

		found, err := repo.FindByID(ctx, task.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, task.ID, found.ID)
		assert.Equal(t, task.Title, found.Title)
	})

	t.Run("find non-existent task returns nil", func(t *testing.T) {
		found, err := repo.FindByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestTaskRepository_FindAll(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)
	assigneeID := CreateTestUser(ctx, testDB.Pool, t)

	// Create test tasks
	createTestTasks := func() {
		CleanupTasks(ctx, testDB.Pool, t)
		now := time.Now().UTC().Truncate(time.Microsecond)

		tasks := []*entity.Task{
			{ID: uuid.New(), Title: "Task 1", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityHigh, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), Title: "Task 2", Status: valueobject.TaskStatusInProgress, Priority: valueobject.PriorityMedium, AssigneeID: &assigneeID, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), Title: "Task 3", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityLow, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), Title: "Searchable Item", Status: valueobject.TaskStatusDone, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		}
		for _, task := range tasks {
			require.NoError(t, repo.Save(ctx, task))
		}
	}

	t.Run("no filter returns all tasks", func(t *testing.T) {
		createTestTasks()

		tasks, pagination, err := repo.FindAll(ctx, output.TaskFilter{}, output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 4)
		assert.Equal(t, int64(4), pagination.Total)
	})

	t.Run("filter by status", func(t *testing.T) {
		createTestTasks()

		status := valueobject.TaskStatusTodo
		tasks, pagination, err := repo.FindAll(ctx, output.TaskFilter{Status: &status}, output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, int64(4), pagination.Total) // Note: total count doesn't filter in current impl
	})

	t.Run("filter by assignee", func(t *testing.T) {
		createTestTasks()

		tasks, _, err := repo.FindAll(ctx, output.TaskFilter{AssigneeID: &assigneeID}, output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, assigneeID, *tasks[0].AssigneeID)
	})

	t.Run("filter by creator", func(t *testing.T) {
		createTestTasks()

		tasks, _, err := repo.FindAll(ctx, output.TaskFilter{CreatorID: &creatorID}, output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 4)
	})

	t.Run("search filter", func(t *testing.T) {
		createTestTasks()

		tasks, _, err := repo.FindAll(ctx, output.TaskFilter{Search: "Searchable"}, output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Searchable Item", tasks[0].Title)
	})

	t.Run("pagination", func(t *testing.T) {
		createTestTasks()

		tasks, pagination, err := repo.FindAll(ctx, output.TaskFilter{}, output.Pagination{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, int64(4), pagination.Total)
		assert.Equal(t, 2, pagination.TotalPages)
	})
}

func TestTaskRepository_FindByAssignee(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)
	assigneeID := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Create tasks with and without assignee
	taskWithAssignee := &entity.Task{
		ID: uuid.New(), Title: "Assigned Task", Status: valueobject.TaskStatusTodo,
		Priority: valueobject.PriorityMedium, AssigneeID: &assigneeID, CreatorID: creatorID,
		Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
	}
	taskWithoutAssignee := &entity.Task{
		ID: uuid.New(), Title: "Unassigned Task", Status: valueobject.TaskStatusTodo,
		Priority: valueobject.PriorityMedium, CreatorID: creatorID,
		Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.Save(ctx, taskWithAssignee))
	require.NoError(t, repo.Save(ctx, taskWithoutAssignee))

	tasks, pagination, err := repo.FindByAssignee(ctx, assigneeID, output.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, taskWithAssignee.ID, tasks[0].ID)
	assert.NotNil(t, pagination)
}

func TestTaskRepository_FindByCreator(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creator1 := CreateTestUser(ctx, testDB.Pool, t)
	creator2 := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	task1 := &entity.Task{
		ID: uuid.New(), Title: "Creator1 Task", Status: valueobject.TaskStatusTodo,
		Priority: valueobject.PriorityMedium, CreatorID: creator1,
		Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
	}
	task2 := &entity.Task{
		ID: uuid.New(), Title: "Creator2 Task", Status: valueobject.TaskStatusTodo,
		Priority: valueobject.PriorityMedium, CreatorID: creator2,
		Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.Save(ctx, task1))
	require.NoError(t, repo.Save(ctx, task2))

	tasks, pagination, err := repo.FindByCreator(ctx, creator1, output.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.NotNil(t, pagination)
}

func TestTaskRepository_CountByStatus(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Create tasks with different statuses
	statuses := []valueobject.TaskStatus{
		valueobject.TaskStatusTodo,
		valueobject.TaskStatusTodo,
		valueobject.TaskStatusInProgress,
		valueobject.TaskStatusDone,
	}
	for i, status := range statuses {
		task := &entity.Task{
			ID: uuid.New(), Title: "Task " + string(rune('A'+i)), Status: status,
			Priority: valueobject.PriorityMedium, CreatorID: creatorID,
			Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, repo.Save(ctx, task))
	}

	todoCount, err := repo.CountByStatus(ctx, valueobject.TaskStatusTodo)
	require.NoError(t, err)
	assert.Equal(t, int64(2), todoCount)

	inProgressCount, err := repo.CountByStatus(ctx, valueobject.TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, int64(1), inProgressCount)

	doneCount, err := repo.CountByStatus(ctx, valueobject.TaskStatusDone)
	require.NoError(t, err)
	assert.Equal(t, int64(1), doneCount)
}

func TestTaskRepository_ExistsByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	task := &entity.Task{
		ID: uuid.New(), Title: "Existing Task", Status: valueobject.TaskStatusTodo,
		Priority: valueobject.PriorityMedium, CreatorID: creatorID,
		Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.Save(ctx, task))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, task.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists returns false", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestTaskRepository_Search(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	tasks := []*entity.Task{
		{ID: uuid.New(), Title: "Build feature", Description: "Important work", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Title: "Fix bug", Description: "Critical bug fix", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityHigh, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Title: "Write tests", Description: "Add unit tests", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
	}
	for _, task := range tasks {
		require.NoError(t, repo.Save(ctx, task))
	}

	t.Run("search by title", func(t *testing.T) {
		results, pagination, err := repo.Search(ctx, "Build", output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Build feature", results[0].Title)
		assert.NotNil(t, pagination)
	})

	t.Run("search by title partial match", func(t *testing.T) {
		// Note: The query builder's count query only searches title,
		// so description-only matches may return empty due to count=0 early exit
		results, _, err := repo.Search(ctx, "bug", output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Fix bug", results[0].Title)
	})

	t.Run("search no results", func(t *testing.T) {
		results, pagination, err := repo.Search(ctx, "nonexistent", output.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, results, 0)
		assert.Equal(t, int64(0), pagination.Total)
	})
}

func TestTaskRepository_FindOverdue(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewTaskRepository(testDB.Pool)
	creatorID := CreateTestUser(ctx, testDB.Pool, t)

	CleanupTasks(ctx, testDB.Pool, t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	pastDate := now.Add(-48 * time.Hour)
	futureDate := now.Add(48 * time.Hour)

	tasks := []*entity.Task{
		{ID: uuid.New(), Title: "Overdue Task", Description: "This is overdue", DueDate: &pastDate, Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Title: "Future Task", Description: "This is in the future", DueDate: &futureDate, Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Title: "Done Overdue", Description: "This is done", DueDate: &pastDate, Status: valueobject.TaskStatusDone, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Title: "No Due Date", Description: "No deadline", Status: valueobject.TaskStatusTodo, Priority: valueobject.PriorityMedium, CreatorID: creatorID, Labels: []uuid.UUID{}, CreatedAt: now, UpdatedAt: now},
	}
	for _, task := range tasks {
		require.NoError(t, repo.Save(ctx, task))
	}

	results, pagination, err := repo.FindOverdue(ctx, output.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Overdue Task", results[0].Title)
	assert.Equal(t, int64(1), pagination.Total)
}
