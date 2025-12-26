// Package querybuilder provides a task-specific query builder.
package querybuilder

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// Allowed columns for task sorting
var TaskAllowedSortColumns = []string{
	"title",
	"status",
	"priority",
	"due_date",
	"created_at",
	"updated_at",
}

// TaskQueryBuilder provides methods for building task queries.
type TaskQueryBuilder struct {
	pool *pgxpool.Pool
}

// NewTaskQueryBuilder creates a new task query builder.
func NewTaskQueryBuilder(pool *pgxpool.Pool) *TaskQueryBuilder {
	return &TaskQueryBuilder{
		pool: pool,
	}
}

// TaskRow represents a row from the tasks table.
type TaskRow struct {
	ID          uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
	AssigneeID  *uuid.UUID
	CreatorID   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FindWithFilter finds tasks matching the given filter and pagination.
func (tqb *TaskQueryBuilder) FindWithFilter(ctx context.Context, filter dto.TaskFilter, pagination dto.Pagination) ([]*entity.Task, int64, error) {
	// Build the count query first
	countQuery := Psql.Select("COUNT(*)").From("tasks")
	countQuery = WhereNotEmpty(countQuery, "status", stringPtrToString(filter.Status))
	countQuery = WhereNotEmpty(countQuery, "priority", stringPtrToString(filter.Priority))

	if filter.AssigneeID != nil {
		countQuery = countQuery.Where(sq.Eq{"assignee_id": *filter.AssigneeID})
	}
	if filter.CreatorID != nil {
		countQuery = countQuery.Where(sq.Eq{"creator_id": *filter.CreatorID})
	}
	if filter.Search != "" {
		countQuery = WhereILike(countQuery, "title", filter.Search)
	}

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = tqb.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*entity.Task{}, 0, nil
	}

	// Build the select query
	selectQuery := Psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).From("tasks")

	selectQuery = WhereNotEmpty(selectQuery, "status", stringPtrToString(filter.Status))
	selectQuery = WhereNotEmpty(selectQuery, "priority", stringPtrToString(filter.Priority))

	if filter.AssigneeID != nil {
		selectQuery = selectQuery.Where(sq.Eq{"assignee_id": *filter.AssigneeID})
	}
	if filter.CreatorID != nil {
		selectQuery = selectQuery.Where(sq.Eq{"creator_id": *filter.CreatorID})
	}
	if filter.Search != "" {
		selectQuery = WhereILike(selectQuery, "title", filter.Search)
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	selectQuery = OrderBySafe(selectQuery, sortBy, pagination.SortDesc, TaskAllowedSortColumns)

	// Apply pagination
	selectQuery = Paginate(selectQuery, pagination.Page, pagination.PageSize)

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := tqb.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Search performs a full-text search on tasks.
func (tqb *TaskQueryBuilder) Search(ctx context.Context, query string, pagination dto.Pagination) ([]*entity.Task, int64, error) {
	searchColumns := []string{"title", "description"}

	// Build count query with search
	countQuery := Psql.Select("COUNT(*)").From("tasks")
	if query != "" {
		countQuery = WhereILike(countQuery, "title", query)
	}

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = tqb.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*entity.Task{}, 0, nil
	}

	// Build select query with search
	opts := FilterOptions{
		Search:   query,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	selectQuery := Psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).From("tasks")

	selectQuery = ApplyFilters(selectQuery, opts, searchColumns, TaskAllowedSortColumns)

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := tqb.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// FindOverdue finds tasks that are past their due date.
func (tqb *TaskQueryBuilder) FindOverdue(ctx context.Context, pagination dto.Pagination) ([]*entity.Task, int64, error) {
	now := time.Now()

	// Build count query
	countQuery := Psql.Select("COUNT(*)").
		From("tasks").
		Where(sq.Lt{"due_date": now}).
		Where(sq.NotEq{"status": []string{"DONE", "ARCHIVED"}})

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = tqb.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*entity.Task{}, 0, nil
	}

	// Build select query
	selectQuery := Psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).
		From("tasks").
		Where(sq.Lt{"due_date": now}).
		Where(sq.NotEq{"status": []string{"DONE", "ARCHIVED"}}).
		OrderBy("due_date ASC")

	selectQuery = Paginate(selectQuery, pagination.Page, pagination.PageSize)

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := tqb.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Helper functions

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func scanTasks(rows pgx.Rows) ([]*entity.Task, error) {
	var tasks []*entity.Task

	for rows.Next() {
		var row TaskRow
		err := rows.Scan(
			&row.ID,
			&row.Title,
			&row.Description,
			&row.Status,
			&row.Priority,
			&row.DueDate,
			&row.AssigneeID,
			&row.CreatorID,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		task := &entity.Task{
			ID:          row.ID,
			Title:       row.Title,
			Description: row.Description,
			Status:      valueobject.TaskStatus(row.Status),
			Priority:    valueobject.Priority(row.Priority),
			DueDate:     row.DueDate,
			AssigneeID:  row.AssigneeID,
			CreatorID:   row.CreatorID,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}
