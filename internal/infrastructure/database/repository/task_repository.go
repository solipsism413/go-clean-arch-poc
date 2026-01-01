// Package repository provides database repository implementations.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/querybuilder"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Ensure TaskRepository implements the output.TaskRepository interface.
var _ output.TaskRepository = (*TaskRepository)(nil)

// TaskRepository implements the task repository using PostgreSQL.
type TaskRepository struct {
	db               sqlc.DBTX
	queries          *sqlc.Queries
	taskQueryBuilder *querybuilder.TaskQueryBuilder
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(db sqlc.DBTX) *TaskRepository {
	return &TaskRepository{
		db:               db,
		queries:          sqlc.New(db),
		taskQueryBuilder: querybuilder.NewTaskQueryBuilder(db),
	}
}

// Save creates a new task.
func (r *TaskRepository) Save(ctx context.Context, task *entity.Task) error {
	var dueDate pgtype.Timestamptz
	if task.DueDate != nil {
		dueDate = pgtype.Timestamptz{Time: *task.DueDate, Valid: true}
	}

	var assigneeID pgtype.UUID
	if task.AssigneeID != nil {
		assigneeID = pgtype.UUID{Bytes: *task.AssigneeID, Valid: true}
	}

	var description *string
	if task.Description != "" {
		description = &task.Description
	}

	_, err := r.queries.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:          task.ID,
		Title:       task.Title,
		Description: description,
		Status:      string(task.Status),
		Priority:    string(task.Priority),
		DueDate:     dueDate,
		AssigneeID:  assigneeID,
		CreatorID:   task.CreatorID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	})

	return err
}

// Update updates an existing task.
func (r *TaskRepository) Update(ctx context.Context, task *entity.Task) error {
	var dueDate pgtype.Timestamptz
	if task.DueDate != nil {
		dueDate = pgtype.Timestamptz{Time: *task.DueDate, Valid: true}
	}

	var assigneeID pgtype.UUID
	if task.AssigneeID != nil {
		assigneeID = pgtype.UUID{Bytes: *task.AssigneeID, Valid: true}
	}

	var description *string
	if task.Description != "" {
		description = &task.Description
	}

	_, err := r.queries.UpdateTask(ctx, sqlc.UpdateTaskParams{
		ID:          task.ID,
		Title:       task.Title,
		Description: description,
		Status:      string(task.Status),
		Priority:    string(task.Priority),
		DueDate:     dueDate,
		AssigneeID:  assigneeID,
	})

	return err
}

// Delete removes a task by ID.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteTask(ctx, id)
}

// FindByID retrieves a task by ID.
func (r *TaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	row, err := r.queries.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return sqlcTaskToEntity(row), nil
}

// FindAll retrieves tasks with filtering and pagination.
func (r *TaskRepository) FindAll(ctx context.Context, filter output.TaskFilter, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	offset := int32((pagination.Page - 1) * pagination.PageSize)
	limit := int32(pagination.PageSize)

	var rows []sqlc.Task
	var err error

	if filter.Status != nil {
		rows, err = r.queries.ListTasksByStatus(ctx, sqlc.ListTasksByStatusParams{
			Status: string(*filter.Status),
			Limit:  limit,
			Offset: offset,
		})
	} else if filter.AssigneeID != nil {
		rows, err = r.queries.ListTasksByAssignee(ctx, sqlc.ListTasksByAssigneeParams{
			AssigneeID: pgtype.UUID{Bytes: *filter.AssigneeID, Valid: true},
			Limit:      limit,
			Offset:     offset,
		})
	} else if filter.CreatorID != nil {
		rows, err = r.queries.ListTasksByCreator(ctx, sqlc.ListTasksByCreatorParams{
			CreatorID: *filter.CreatorID,
			Limit:     limit,
			Offset:    offset,
		})
	} else if filter.Search != "" {
		searchParam := filter.Search
		rows, err = r.queries.SearchTasks(ctx, sqlc.SearchTasksParams{
			Column1: &searchParam,
			Limit:   limit,
			Offset:  offset,
		})
	} else {
		rows, err = r.queries.ListTasks(ctx, sqlc.ListTasksParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, nil, err
	}

	// Get total count
	total, err := r.queries.CountTasks(ctx)
	if err != nil {
		return nil, nil, err
	}

	tasks := make([]*entity.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, sqlcTaskToEntity(row))
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return tasks, &output.PaginatedResult{
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// FindByAssignee retrieves tasks assigned to a specific user.
func (r *TaskRepository) FindByAssignee(ctx context.Context, assigneeID uuid.UUID, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	filter := output.TaskFilter{AssigneeID: &assigneeID}
	return r.FindAll(ctx, filter, pagination)
}

// FindByCreator retrieves tasks created by a specific user.
func (r *TaskRepository) FindByCreator(ctx context.Context, creatorID uuid.UUID, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	filter := output.TaskFilter{CreatorID: &creatorID}
	return r.FindAll(ctx, filter, pagination)
}

// CountByStatus counts tasks by status.
func (r *TaskRepository) CountByStatus(ctx context.Context, status valueobject.TaskStatus) (int64, error) {
	return r.queries.CountTasksByStatus(ctx, string(status))
}

// ExistsByID checks if a task exists.
func (r *TaskRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.queries.TaskExists(ctx, id)
}

// Search performs a full-text search on tasks.
func (r *TaskRepository) Search(ctx context.Context, query string, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	dtoPagination := dto.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	tasks, total, err := r.taskQueryBuilder.Search(ctx, query, dtoPagination)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return tasks, &output.PaginatedResult{
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// FindOverdue finds tasks that are past their due date.
func (r *TaskRepository) FindOverdue(ctx context.Context, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	dtoPagination := dto.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	tasks, total, err := r.taskQueryBuilder.FindOverdue(ctx, dtoPagination)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return tasks, &output.PaginatedResult{
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// sqlcTaskToEntity converts SQLC model to domain entity.
func sqlcTaskToEntity(row sqlc.Task) *entity.Task {
	task := &entity.Task{
		ID:        row.ID,
		Title:     row.Title,
		Status:    valueobject.TaskStatus(row.Status),
		Priority:  valueobject.Priority(row.Priority),
		CreatorID: row.CreatorID,
		Labels:    make([]uuid.UUID, 0),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.Description != nil {
		task.Description = *row.Description
	}

	if row.DueDate.Valid {
		task.DueDate = &row.DueDate.Time
	}

	if row.AssigneeID.Valid {
		assigneeID := uuid.UUID(row.AssigneeID.Bytes)
		task.AssigneeID = &assigneeID
	}

	return task
}
