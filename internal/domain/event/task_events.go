// Package event contains domain events that represent important changes
// in the domain. These events can be published to external systems for
// event-driven architecture patterns.
package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// Event is the base interface for all domain events.
type Event interface {
	// EventType returns the type/name of the event.
	EventType() string
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// AggregateID returns the ID of the aggregate that produced the event.
	AggregateID() uuid.UUID
}

// BaseEvent contains common fields for all events.
type BaseEvent struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	AggregateId uuid.UUID `json:"aggregate_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// EventType returns the type of the event.
func (e BaseEvent) EventType() string {
	return e.Type
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the aggregate ID.
func (e BaseEvent) AggregateID() uuid.UUID {
	return e.AggregateId
}

// NewBaseEvent creates a new base event.
func NewBaseEvent(eventType string, aggregateID uuid.UUID) BaseEvent {
	return BaseEvent{
		ID:          uuid.New(),
		Type:        eventType,
		AggregateId: aggregateID,
		Timestamp:   time.Now().UTC(),
	}
}

// TaskCreated is emitted when a new task is created.
type TaskCreated struct {
	BaseEvent
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    valueobject.Priority   `json:"priority"`
	Status      valueobject.TaskStatus `json:"status"`
	CreatorID   uuid.UUID              `json:"creator_id"`
}

// NewTaskCreated creates a new TaskCreated event.
func NewTaskCreated(taskID uuid.UUID, title, description string, priority valueobject.Priority, creatorID uuid.UUID) TaskCreated {
	return TaskCreated{
		BaseEvent:   NewBaseEvent("task.created", taskID),
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      valueobject.TaskStatusTodo,
		CreatorID:   creatorID,
	}
}

// TaskUpdated is emitted when a task is updated.
type TaskUpdated struct {
	BaseEvent
	Title       *string                 `json:"title,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Priority    *valueobject.Priority   `json:"priority,omitempty"`
	Status      *valueobject.TaskStatus `json:"status,omitempty"`
	UpdatedBy   uuid.UUID               `json:"updated_by"`
}

// NewTaskUpdated creates a new TaskUpdated event.
func NewTaskUpdated(taskID uuid.UUID, updatedBy uuid.UUID) TaskUpdated {
	return TaskUpdated{
		BaseEvent: NewBaseEvent("task.updated", taskID),
		UpdatedBy: updatedBy,
	}
}

// WithTitle sets the title field for the update event.
func (e TaskUpdated) WithTitle(title string) TaskUpdated {
	e.Title = &title
	return e
}

// WithDescription sets the description field for the update event.
func (e TaskUpdated) WithDescription(description string) TaskUpdated {
	e.Description = &description
	return e
}

// WithPriority sets the priority field for the update event.
func (e TaskUpdated) WithPriority(priority valueobject.Priority) TaskUpdated {
	e.Priority = &priority
	return e
}

// WithStatus sets the status field for the update event.
func (e TaskUpdated) WithStatus(status valueobject.TaskStatus) TaskUpdated {
	e.Status = &status
	return e
}

// TaskDeleted is emitted when a task is deleted.
type TaskDeleted struct {
	BaseEvent
	DeletedBy uuid.UUID `json:"deleted_by"`
}

// NewTaskDeleted creates a new TaskDeleted event.
func NewTaskDeleted(taskID uuid.UUID, deletedBy uuid.UUID) TaskDeleted {
	return TaskDeleted{
		BaseEvent: NewBaseEvent("task.deleted", taskID),
		DeletedBy: deletedBy,
	}
}

// TaskAssigned is emitted when a task is assigned to a user.
type TaskAssigned struct {
	BaseEvent
	AssigneeID uuid.UUID `json:"assignee_id"`
	AssignedBy uuid.UUID `json:"assigned_by"`
}

// NewTaskAssigned creates a new TaskAssigned event.
func NewTaskAssigned(taskID, assigneeID, assignedBy uuid.UUID) TaskAssigned {
	return TaskAssigned{
		BaseEvent:  NewBaseEvent("task.assigned", taskID),
		AssigneeID: assigneeID,
		AssignedBy: assignedBy,
	}
}

// TaskUnassigned is emitted when a task is unassigned.
type TaskUnassigned struct {
	BaseEvent
	PreviousAssigneeID uuid.UUID `json:"previous_assignee_id"`
	UnassignedBy       uuid.UUID `json:"unassigned_by"`
}

// NewTaskUnassigned creates a new TaskUnassigned event.
func NewTaskUnassigned(taskID, previousAssigneeID, unassignedBy uuid.UUID) TaskUnassigned {
	return TaskUnassigned{
		BaseEvent:          NewBaseEvent("task.unassigned", taskID),
		PreviousAssigneeID: previousAssigneeID,
		UnassignedBy:       unassignedBy,
	}
}

// TaskCompleted is emitted when a task is marked as done.
type TaskCompleted struct {
	BaseEvent
	CompletedBy uuid.UUID `json:"completed_by"`
}

// NewTaskCompleted creates a new TaskCompleted event.
func NewTaskCompleted(taskID, completedBy uuid.UUID) TaskCompleted {
	return TaskCompleted{
		BaseEvent:   NewBaseEvent("task.completed", taskID),
		CompletedBy: completedBy,
	}
}

// TaskArchived is emitted when a task is archived.
type TaskArchived struct {
	BaseEvent
	ArchivedBy uuid.UUID `json:"archived_by"`
}

// NewTaskArchived creates a new TaskArchived event.
func NewTaskArchived(taskID, archivedBy uuid.UUID) TaskArchived {
	return TaskArchived{
		BaseEvent:  NewBaseEvent("task.archived", taskID),
		ArchivedBy: archivedBy,
	}
}

// TaskStatusChanged is emitted when a task status changes.
type TaskStatusChanged struct {
	BaseEvent
	OldStatus valueobject.TaskStatus `json:"old_status"`
	NewStatus valueobject.TaskStatus `json:"new_status"`
	ChangedBy uuid.UUID              `json:"changed_by"`
}

// NewTaskStatusChanged creates a new TaskStatusChanged event.
func NewTaskStatusChanged(taskID uuid.UUID, oldStatus, newStatus valueobject.TaskStatus, changedBy uuid.UUID) TaskStatusChanged {
	return TaskStatusChanged{
		BaseEvent: NewBaseEvent("task.status_changed", taskID),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
	}
}

// TaskLabelAdded is emitted when a label is added to a task.
type TaskLabelAdded struct {
	BaseEvent
	LabelID uuid.UUID `json:"label_id"`
	AddedBy uuid.UUID `json:"added_by"`
}

// NewTaskLabelAdded creates a new TaskLabelAdded event.
func NewTaskLabelAdded(taskID, labelID, addedBy uuid.UUID) TaskLabelAdded {
	return TaskLabelAdded{
		BaseEvent: NewBaseEvent("task.label_added", taskID),
		LabelID:   labelID,
		AddedBy:   addedBy,
	}
}

// TaskLabelRemoved is emitted when a label is removed from a task.
type TaskLabelRemoved struct {
	BaseEvent
	LabelID   uuid.UUID `json:"label_id"`
	RemovedBy uuid.UUID `json:"removed_by"`
}

// NewTaskLabelRemoved creates a new TaskLabelRemoved event.
func NewTaskLabelRemoved(taskID, labelID, removedBy uuid.UUID) TaskLabelRemoved {
	return TaskLabelRemoved{
		BaseEvent: NewBaseEvent("task.label_removed", taskID),
		LabelID:   labelID,
		RemovedBy: removedBy,
	}
}
