// Package input contains the input port interfaces for the application layer.
// These interfaces define the use cases that the application exposes to the
// outside world (primary adapters like REST, GraphQL, gRPC handlers).
package input

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
)

// TaskService defines the input port for task-related use cases.
// This interface is implemented by the task use case and used by transport adapters.
type TaskService interface {
	// CreateTask creates a new task.
	CreateTask(ctx context.Context, input dto.CreateTaskInput) (*dto.TaskOutput, error)

	// UpdateTask updates an existing task.
	UpdateTask(ctx context.Context, id uuid.UUID, input dto.UpdateTaskInput) (*dto.TaskOutput, error)

	// DeleteTask deletes a task by ID.
	DeleteTask(ctx context.Context, id uuid.UUID) error

	// GetTask retrieves a task by ID.
	GetTask(ctx context.Context, id uuid.UUID) (*dto.TaskOutput, error)

	// ListTasks retrieves tasks with filtering and pagination.
	ListTasks(ctx context.Context, filter dto.TaskFilter, pagination dto.Pagination) (*dto.TaskListOutput, error)

	// AssignTask assigns a task to a user.
	AssignTask(ctx context.Context, taskID, assigneeID uuid.UUID) (*dto.TaskOutput, error)

	// UnassignTask removes the assignee from a task.
	UnassignTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error)

	// ChangeTaskStatus changes the status of a task.
	ChangeTaskStatus(ctx context.Context, taskID uuid.UUID, status string) (*dto.TaskOutput, error)

	// CompleteTask marks a task as done.
	CompleteTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error)

	// ArchiveTask archives a completed task.
	ArchiveTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error)

	// AddLabel adds a label to a task.
	AddLabel(ctx context.Context, taskID, labelID uuid.UUID) (*dto.TaskOutput, error)

	// RemoveLabel removes a label from a task.
	RemoveLabel(ctx context.Context, taskID, labelID uuid.UUID) (*dto.TaskOutput, error)

	// SearchTasks performs a full-text search on tasks.
	SearchTasks(ctx context.Context, query string, pagination dto.Pagination) (*dto.TaskListOutput, error)

	// GetOverdueTasks retrieves tasks that are past their due date.
	GetOverdueTasks(ctx context.Context, pagination dto.Pagination) (*dto.TaskListOutput, error)
}
