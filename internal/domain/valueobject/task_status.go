// Package valueobject contains immutable value objects for the domain.
// Value objects represent concepts that are identified by their attributes
// rather than a unique identity.
package valueobject

// TaskStatus represents the status of a task.
// It is an enumeration of possible task states.
type TaskStatus string

// Task status constants.
const (
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusInReview   TaskStatus = "IN_REVIEW"
	TaskStatusDone       TaskStatus = "DONE"
	TaskStatusArchived   TaskStatus = "ARCHIVED"
)

// validTaskStatuses contains all valid task status values.
var validTaskStatuses = map[TaskStatus]bool{
	TaskStatusTodo:       true,
	TaskStatusInProgress: true,
	TaskStatusInReview:   true,
	TaskStatusDone:       true,
	TaskStatusArchived:   true,
}

// IsValid returns true if the status is a valid TaskStatus.
func (s TaskStatus) IsValid() bool {
	return validTaskStatuses[s]
}

// String returns the string representation of the status.
func (s TaskStatus) String() string {
	return string(s)
}

// ParseTaskStatus parses a string into a TaskStatus.
// Returns an error if the string is not a valid status.
func ParseTaskStatus(s string) (TaskStatus, error) {
	status := TaskStatus(s)
	if !status.IsValid() {
		return "", ErrInvalidTaskStatus
	}
	return status, nil
}

// AllTaskStatuses returns all valid task statuses.
func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusTodo,
		TaskStatusInProgress,
		TaskStatusInReview,
		TaskStatusDone,
		TaskStatusArchived,
	}
}

// IsTerminal returns true if the status is a terminal state (DONE or ARCHIVED).
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusDone || s == TaskStatusArchived
}

// CanTransitionTo checks if a transition from this status to target is allowed.
func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	// Cannot transition from ARCHIVED
	if s == TaskStatusArchived {
		return false
	}

	// DONE can only transition to ARCHIVED
	if s == TaskStatusDone {
		return target == TaskStatusArchived
	}

	// Other statuses can transition to any non-archived status
	return target != TaskStatusArchived || s == TaskStatusDone
}
