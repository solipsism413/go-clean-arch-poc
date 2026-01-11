// Package querybuilder_test contains tests for the task query builder.
package querybuilder_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/querybuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskQueryBuilder(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("should create task query builder", func(t *testing.T) {
		qb := querybuilder.NewTaskQueryBuilder(testDB.Pool)
		assert.NotNil(t, qb)
	})
}

func TestTaskQueryBuilder_FindWithFilter(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Create test user
	userID := CreateTestUser(ctx, testDB.Pool, t)

	// Create test tasks
	CleanupTasks(ctx, testDB.Pool, t)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Task 1", "TODO", "LOW", nil)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Task 2", "IN_PROGRESS", "MEDIUM", nil)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Task 3", "DONE", "HIGH", nil)

	qb := querybuilder.NewTaskQueryBuilder(testDB.Pool)

	t.Run("should find all tasks without filter", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 3)
	})

	t.Run("should filter by status", func(t *testing.T) {
		status := "TODO"
		filter := dto.TaskFilter{Status: &status}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Task 1", tasks[0].Title)
	})

	t.Run("should filter by priority", func(t *testing.T) {
		priority := "HIGH"
		filter := dto.TaskFilter{Priority: &priority}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Task 3", tasks[0].Title)
	})

	t.Run("should filter by creator ID", func(t *testing.T) {
		filter := dto.TaskFilter{CreatorID: &userID}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 3)
	})

	t.Run("should filter by non-existent creator ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		filter := dto.TaskFilter{CreatorID: &nonExistentID}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, tasks, 0)
	})

	t.Run("should search by title", func(t *testing.T) {
		filter := dto.TaskFilter{Search: "Task 2"}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Task 2", tasks[0].Title)
	})

	t.Run("should apply pagination", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 2}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 2)
	})

	t.Run("should apply sorting by title ASC", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "title", SortDesc: false}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Task 1", tasks[0].Title)
	})

	t.Run("should apply sorting by title DESC", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "title", SortDesc: true}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Task 3", tasks[0].Title)
	})

	t.Run("should handle invalid sort column", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "invalid_column"}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 3)
	})

	t.Run("should handle zero page defaults to 1", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 0, PageSize: 20}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 3)
	})

	t.Run("should handle zero page size defaults to 20", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 0}

		tasks, _, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})

	t.Run("should cap page size at 100", func(t *testing.T) {
		filter := dto.TaskFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 200}

		tasks, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.LessOrEqual(t, len(tasks), 100)
	})
}

func TestTaskQueryBuilder_Search(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Create test user
	userID := CreateTestUser(ctx, testDB.Pool, t)

	// Create test tasks
	CleanupTasks(ctx, testDB.Pool, t)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Important Meeting", "TODO", "HIGH", nil)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Review Code", "IN_PROGRESS", "MEDIUM", nil)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Meeting Notes", "DONE", "LOW", nil)

	qb := querybuilder.NewTaskQueryBuilder(testDB.Pool)

	t.Run("should search tasks by title", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.Search(ctx, "Meeting", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, tasks, 2)
	})

	t.Run("should return all tasks with empty query", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.Search(ctx, "", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tasks, 3)
	})

	t.Run("should return empty result for non-matching query", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.Search(ctx, "NonExistent", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, tasks, 0)
	})

	t.Run("should apply sorting in search", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "title", SortDesc: false}

		tasks, total, err := qb.Search(ctx, "Meeting", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Equal(t, "Important Meeting", tasks[0].Title)
	})

	t.Run("should apply pagination in search", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 1}

		tasks, total, err := qb.Search(ctx, "Meeting", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, tasks, 1)
	})
}

func TestTaskQueryBuilder_FindOverdue(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Create test user
	userID := CreateTestUser(ctx, testDB.Pool, t)

	// Create test tasks
	CleanupTasks(ctx, testDB.Pool, t)

	pastDate := time.Now().Add(-24 * time.Hour)
	futureDate := time.Now().Add(24 * time.Hour)

	CreateTestTask(ctx, testDB.Pool, t, userID, "Overdue Task", "TODO", "HIGH", &pastDate)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Future Task", "IN_PROGRESS", "MEDIUM", &futureDate)
	CreateTestTask(ctx, testDB.Pool, t, userID, "No Due Date Task", "TODO", "LOW", nil)
	CreateTestTask(ctx, testDB.Pool, t, userID, "Completed Overdue", "DONE", "HIGH", &pastDate)

	qb := querybuilder.NewTaskQueryBuilder(testDB.Pool)

	t.Run("should find overdue tasks", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		tasks, total, err := qb.FindOverdue(ctx, pagination)

		require.NoError(t, err)
		// Only "Overdue Task" should be returned (past due date and not DONE/ARCHIVED)
		assert.Equal(t, int64(1), total)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Overdue Task", tasks[0].Title)
	})

	t.Run("should apply pagination to overdue tasks", func(t *testing.T) {
		// Create more overdue tasks
		pastDate2 := time.Now().Add(-48 * time.Hour)
		CreateTestTask(ctx, testDB.Pool, t, userID, "Another Overdue", "IN_PROGRESS", "URGENT", &pastDate2)

		pagination := dto.Pagination{Page: 1, PageSize: 1}

		tasks, total, err := qb.FindOverdue(ctx, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, tasks, 1)
	})
}

func TestTaskAllowedSortColumns(t *testing.T) {
	t.Run("should have correct allowed sort columns", func(t *testing.T) {
		expected := []string{
			"title",
			"status",
			"priority",
			"due_date",
			"created_at",
			"updated_at",
		}

		assert.Equal(t, expected, querybuilder.TaskAllowedSortColumns)
	})
}

func TestTaskRow(t *testing.T) {
	t.Run("should create TaskRow struct", func(t *testing.T) {
		now := time.Now()
		dueDate := now.Add(24 * time.Hour)
		assigneeID := uuid.New()
		creatorID := uuid.New()

		row := querybuilder.TaskRow{
			ID:          uuid.New(),
			Title:       "Test Task",
			Description: "Test Description",
			Status:      "TODO",
			Priority:    "HIGH",
			DueDate:     &dueDate,
			AssigneeID:  &assigneeID,
			CreatorID:   creatorID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		assert.Equal(t, "Test Task", row.Title)
		assert.Equal(t, "Test Description", row.Description)
		assert.Equal(t, "TODO", row.Status)
		assert.Equal(t, "HIGH", row.Priority)
		assert.Equal(t, &dueDate, row.DueDate)
		assert.Equal(t, &assigneeID, row.AssigneeID)
		assert.Equal(t, creatorID, row.CreatorID)
	})
}
