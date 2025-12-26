// Package dto contains Data Transfer Objects for the application layer.
// DTOs are used to transfer data between layers and do not contain business logic.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// CreateTaskInput represents the input for creating a task.
type CreateTaskInput struct {
	Title       string      `json:"title" validate:"required,min=1,max=255"`
	Description string      `json:"description" validate:"max=5000"`
	Priority    string      `json:"priority" validate:"required,oneof=LOW MEDIUM HIGH URGENT"`
	DueDate     *time.Time  `json:"dueDate,omitempty"`
	AssigneeID  *uuid.UUID  `json:"assigneeId,omitempty"`
	LabelIDs    []uuid.UUID `json:"labelIds,omitempty"`
}

// UpdateTaskInput represents the input for updating a task.
type UpdateTaskInput struct {
	Title        *string    `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Description  *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	Priority     *string    `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	DueDate      *time.Time `json:"dueDate,omitempty"`
	ClearDueDate bool       `json:"clearDueDate,omitempty"`
}

// TaskOutput represents the output for task operations.
type TaskOutput struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Status      string           `json:"status"`
	Priority    string           `json:"priority"`
	DueDate     *time.Time       `json:"dueDate,omitempty"`
	AssigneeID  *uuid.UUID       `json:"assigneeId,omitempty"`
	Assignee    *UserBasicOutput `json:"assignee,omitempty"`
	CreatorID   uuid.UUID        `json:"creatorId"`
	Creator     *UserBasicOutput `json:"creator,omitempty"`
	Labels      []LabelOutput    `json:"labels,omitempty"`
	IsOverdue   bool             `json:"isOverdue"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// TaskFromEntity converts a Task entity to TaskOutput DTO.
func TaskFromEntity(task *entity.Task) *TaskOutput {
	if task == nil {
		return nil
	}
	return &TaskOutput{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Priority:    string(task.Priority),
		DueDate:     task.DueDate,
		AssigneeID:  task.AssigneeID,
		CreatorID:   task.CreatorID,
		IsOverdue:   task.IsOverdue(),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

// TaskFilter represents filtering options for task queries.
type TaskFilter struct {
	Status     *string     `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS IN_REVIEW DONE ARCHIVED"`
	Priority   *string     `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	AssigneeID *uuid.UUID  `json:"assigneeId,omitempty"`
	CreatorID  *uuid.UUID  `json:"creatorId,omitempty"`
	LabelIDs   []uuid.UUID `json:"labelIds,omitempty"`
	Search     string      `json:"search,omitempty" validate:"max=100"`
	IsOverdue  *bool       `json:"isOverdue,omitempty"`
}

// ToOutputFilter converts DTO filter to output port filter.
func (f *TaskFilter) ToOutputFilter() *TaskOutputFilter {
	filter := &TaskOutputFilter{
		Search:     f.Search,
		AssigneeID: f.AssigneeID,
		CreatorID:  f.CreatorID,
		LabelIDs:   f.LabelIDs,
		IsOverdue:  f.IsOverdue,
	}
	if f.Status != nil {
		status := valueobject.TaskStatus(*f.Status)
		filter.Status = &status
	}
	if f.Priority != nil {
		priority := valueobject.Priority(*f.Priority)
		filter.Priority = &priority
	}
	return filter
}

// TaskOutputFilter is a type alias for use with the output port.
type TaskOutputFilter struct {
	Status     *valueobject.TaskStatus
	Priority   *valueobject.Priority
	AssigneeID *uuid.UUID
	CreatorID  *uuid.UUID
	LabelIDs   []uuid.UUID
	Search     string
	IsOverdue  *bool
}

// TaskListOutput represents a paginated list of tasks.
type TaskListOutput struct {
	Tasks      []*TaskOutput `json:"tasks"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	SortBy   string `json:"sortBy,omitempty" validate:"omitempty,oneof=createdAt updatedAt dueDate priority title"`
	SortDesc bool   `json:"sortDesc,omitempty"`
}

// DefaultPagination returns default pagination values.
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
		SortBy:   "createdAt",
		SortDesc: true,
	}
}
