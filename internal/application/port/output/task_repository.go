// Package output contains the output port interfaces for the application layer.
// These interfaces define the contracts that secondary adapters (repositories,
// external services) must fulfill.
package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// TaskFilter defines the filtering options for task queries.
type TaskFilter struct {
	Status     *valueobject.TaskStatus
	Priority   *valueobject.Priority
	AssigneeID *uuid.UUID
	CreatorID  *uuid.UUID
	LabelIDs   []uuid.UUID
	Search     string
	IsOverdue  *bool
}

// Pagination defines pagination parameters.
type Pagination struct {
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

// PaginatedResult contains pagination metadata.
type PaginatedResult struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// TaskRepository defines the output port for task persistence.
type TaskRepository interface {
	// Save creates a new task.
	Save(ctx context.Context, task *entity.Task) error

	// Update updates an existing task.
	Update(ctx context.Context, task *entity.Task) error

	// Delete removes a task by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a task by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Task, error)

	// FindAll retrieves tasks with filtering and pagination.
	FindAll(ctx context.Context, filter TaskFilter, pagination Pagination) ([]*entity.Task, *PaginatedResult, error)

	// FindByAssignee retrieves tasks assigned to a specific user.
	FindByAssignee(ctx context.Context, assigneeID uuid.UUID, pagination Pagination) ([]*entity.Task, *PaginatedResult, error)

	// FindByCreator retrieves tasks created by a specific user.
	FindByCreator(ctx context.Context, creatorID uuid.UUID, pagination Pagination) ([]*entity.Task, *PaginatedResult, error)

	// CountByStatus counts tasks by status.
	CountByStatus(ctx context.Context, status valueobject.TaskStatus) (int64, error)

	// ExistsByID checks if a task exists.
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// Search performs a full-text search on tasks.
	Search(ctx context.Context, query string, pagination Pagination) ([]*entity.Task, *PaginatedResult, error)

	// FindOverdue finds tasks that are past their due date.
	FindOverdue(ctx context.Context, pagination Pagination) ([]*entity.Task, *PaginatedResult, error)
}
