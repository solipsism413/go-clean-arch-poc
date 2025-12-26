// Package entity contains the core domain entities for the task management application.
// These entities encapsulate business logic and invariants that are independent of
// infrastructure concerns like databases, APIs, or external services.
package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// Task represents a task in the task management system.
// It is an aggregate root that encapsulates all task-related business logic.
type Task struct {
	// ID is the unique identifier for the task.
	ID uuid.UUID

	// Title is the short description of the task.
	Title string

	// Description provides detailed information about the task.
	Description string

	// Status represents the current state of the task.
	Status valueobject.TaskStatus

	// Priority indicates the urgency of the task.
	Priority valueobject.Priority

	// DueDate is the deadline for the task. Can be nil if no deadline is set.
	DueDate *time.Time

	// AssigneeID is the ID of the user assigned to this task. Can be nil if unassigned.
	AssigneeID *uuid.UUID

	// CreatorID is the ID of the user who created this task.
	CreatorID uuid.UUID

	// Labels contains the IDs of labels associated with this task.
	Labels []uuid.UUID

	// CreatedAt is the timestamp when the task was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the task was last updated.
	UpdatedAt time.Time
}

// NewTask creates a new Task with the given parameters.
// It initializes the task with TODO status and sets creation timestamps.
func NewTask(title, description string, priority valueobject.Priority, creatorID uuid.UUID) (*Task, error) {
	if title == "" {
		return nil, ErrEmptyTitle
	}

	if !priority.IsValid() {
		return nil, ErrInvalidPriority
	}

	now := time.Now().UTC()
	return &Task{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Status:      valueobject.TaskStatusTodo,
		Priority:    priority,
		CreatorID:   creatorID,
		Labels:      make([]uuid.UUID, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Complete marks the task as done.
// Returns an error if the task is already archived.
func (t *Task) Complete() error {
	if t.Status == valueobject.TaskStatusArchived {
		return ErrTaskArchived
	}
	t.Status = valueobject.TaskStatusDone
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Archive archives the task.
// Returns an error if the task is not in DONE status.
func (t *Task) Archive() error {
	if t.Status != valueobject.TaskStatusDone {
		return ErrTaskNotDone
	}
	t.Status = valueobject.TaskStatusArchived
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Assign assigns the task to a user.
// Returns an error if the task is archived.
func (t *Task) Assign(assigneeID uuid.UUID) error {
	if t.Status == valueobject.TaskStatusArchived {
		return ErrTaskArchived
	}
	t.AssigneeID = &assigneeID
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Unassign removes the assignee from the task.
func (t *Task) Unassign() {
	t.AssigneeID = nil
	t.UpdatedAt = time.Now().UTC()
}

// ChangeStatus changes the task status to the given status.
// Validates the status transition.
func (t *Task) ChangeStatus(status valueobject.TaskStatus) error {
	if t.Status == valueobject.TaskStatusArchived {
		return ErrTaskArchived
	}

	if !status.IsValid() {
		return ErrInvalidStatus
	}

	// Cannot transition from DONE to anything other than ARCHIVED
	if t.Status == valueobject.TaskStatusDone && status != valueobject.TaskStatusArchived {
		return ErrInvalidStatusTransition
	}

	t.Status = status
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateTitle updates the task title.
func (t *Task) UpdateTitle(title string) error {
	if title == "" {
		return ErrEmptyTitle
	}
	t.Title = title
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDescription updates the task description.
func (t *Task) UpdateDescription(description string) {
	t.Description = description
	t.UpdatedAt = time.Now().UTC()
}

// UpdatePriority updates the task priority.
func (t *Task) UpdatePriority(priority valueobject.Priority) error {
	if !priority.IsValid() {
		return ErrInvalidPriority
	}
	t.Priority = priority
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// SetDueDate sets the due date for the task.
func (t *Task) SetDueDate(dueDate time.Time) {
	t.DueDate = &dueDate
	t.UpdatedAt = time.Now().UTC()
}

// ClearDueDate removes the due date from the task.
func (t *Task) ClearDueDate() {
	t.DueDate = nil
	t.UpdatedAt = time.Now().UTC()
}

// AddLabel adds a label to the task.
func (t *Task) AddLabel(labelID uuid.UUID) {
	for _, id := range t.Labels {
		if id == labelID {
			return // Label already exists
		}
	}
	t.Labels = append(t.Labels, labelID)
	t.UpdatedAt = time.Now().UTC()
}

// RemoveLabel removes a label from the task.
func (t *Task) RemoveLabel(labelID uuid.UUID) {
	for i, id := range t.Labels {
		if id == labelID {
			t.Labels = append(t.Labels[:i], t.Labels[i+1:]...)
			t.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

// IsOverdue returns true if the task is past its due date and not completed.
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	if t.Status == valueobject.TaskStatusDone || t.Status == valueobject.TaskStatusArchived {
		return false
	}
	return time.Now().UTC().After(*t.DueDate)
}

// IsAssigned returns true if the task is assigned to someone.
func (t *Task) IsAssigned() bool {
	return t.AssigneeID != nil
}

// CanBeModifiedBy checks if the user can modify this task.
// A user can modify a task if they are the creator or the assignee.
func (t *Task) CanBeModifiedBy(userID uuid.UUID) bool {
	if t.CreatorID == userID {
		return true
	}
	if t.AssigneeID != nil && *t.AssigneeID == userID {
		return true
	}
	return false
}
