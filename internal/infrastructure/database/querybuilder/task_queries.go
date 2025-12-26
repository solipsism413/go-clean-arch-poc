// Package querybuilder provides a task-specific query builder.
package querybuilder

import (
	"context"
	"fmt"
	"strings"
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
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	countQuery := psql.Select("COUNT(*)").From("tasks")

	statusStr := stringPtrToString(filter.Status)
	if statusStr != "" {
		countQuery = countQuery.Where(sq.Eq{"status": statusStr})
	}

	priorityStr := stringPtrToString(filter.Priority)
	if priorityStr != "" {
		countQuery = countQuery.Where(sq.Eq{"priority": priorityStr})
	}

	if filter.AssigneeID != nil {
		countQuery = countQuery.Where(sq.Eq{"assignee_id": *filter.AssigneeID})
	}
	if filter.CreatorID != nil {
		countQuery = countQuery.Where(sq.Eq{"creator_id": *filter.CreatorID})
	}
	if filter.Search != "" {
		countQuery = countQuery.Where(sq.ILike{"title": "%" + filter.Search + "%"})
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
	selectQuery := psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).From("tasks")

	if statusStr != "" {
		selectQuery = selectQuery.Where(sq.Eq{"status": statusStr})
	}
	if priorityStr != "" {
		selectQuery = selectQuery.Where(sq.Eq{"priority": priorityStr})
	}

	if filter.AssigneeID != nil {
		selectQuery = selectQuery.Where(sq.Eq{"assignee_id": *filter.AssigneeID})
	}
	if filter.CreatorID != nil {
		selectQuery = selectQuery.Where(sq.Eq{"creator_id": *filter.CreatorID})
	}
	if filter.Search != "" {
		selectQuery = selectQuery.Where(sq.ILike{"title": "%" + filter.Search + "%"})
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	// Validate sort column
	isAllowedSort := false
	for _, allowed := range TaskAllowedSortColumns {
		if strings.EqualFold(sortBy, allowed) {
			sortBy = allowed
			isAllowedSort = true
			break
		}
	}

	if isAllowedSort {
		direction := "ASC"
		if pagination.SortDesc {
			direction = "DESC"
		}
		selectQuery = selectQuery.OrderBy(fmt.Sprintf("%s %s", sortBy, direction))
	}

	// Apply pagination
	page := pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := pagination.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	selectQuery = selectQuery.Limit(uint64(pageSize)).Offset(uint64(offset))

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

	// Build count query with search
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	countQuery := psql.Select("COUNT(*)").From("tasks")
	if query != "" {
		countQuery = countQuery.Where(sq.ILike{"title": "%" + query + "%"})
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
	selectQuery := psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).From("tasks")

	if query != "" {
		or := sq.Or{
			sq.ILike{"title": "%" + query + "%"},
			sq.ILike{"description": "%" + query + "%"},
		}
		selectQuery = selectQuery.Where(or)
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy != "" {
		isAllowedSort := false
		for _, allowed := range TaskAllowedSortColumns {
			if strings.EqualFold(sortBy, allowed) {
				sortBy = allowed
				isAllowedSort = true
				break
			}
		}
		if isAllowedSort {
			direction := "ASC"
			if pagination.SortDesc {
				direction = "DESC"
			}
			selectQuery = selectQuery.OrderBy(fmt.Sprintf("%s %s", sortBy, direction))
		}
	}

	// Apply pagination
	page := pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := pagination.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	selectQuery = selectQuery.Limit(uint64(pageSize)).Offset(uint64(offset))

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
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	countQuery := psql.Select("COUNT(*)").
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
	// Build select query
	selectQuery := psql.Select(
		"id", "title", "description", "status", "priority",
		"due_date", "assignee_id", "creator_id", "created_at", "updated_at",
	).
		From("tasks").
		Where(sq.Lt{"due_date": now}).
		Where(sq.NotEq{"status": []string{"DONE", "ARCHIVED"}}).
		OrderBy("due_date ASC")

	// Apply pagination
	page := pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := pagination.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	selectQuery = selectQuery.Limit(uint64(pageSize)).Offset(uint64(offset))

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
