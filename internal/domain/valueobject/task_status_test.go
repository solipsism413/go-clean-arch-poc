package valueobject_test

import (
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status valueobject.TaskStatus
		valid  bool
	}{
		{valueobject.TaskStatusTodo, true},
		{valueobject.TaskStatusInProgress, true},
		{valueobject.TaskStatusInReview, true},
		{valueobject.TaskStatusDone, true},
		{valueobject.TaskStatusArchived, true},
		{valueobject.TaskStatus("INVALID"), false},
		{valueobject.TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		input   string
		want    valueobject.TaskStatus
		wantErr bool
	}{
		{"TODO", valueobject.TaskStatusTodo, false},
		{"IN_PROGRESS", valueobject.TaskStatusInProgress, false},
		{"IN_REVIEW", valueobject.TaskStatusInReview, false},
		{"DONE", valueobject.TaskStatusDone, false},
		{"ARCHIVED", valueobject.TaskStatusArchived, false},
		{"INVALID", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := valueobject.ParseTaskStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTaskStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   valueobject.TaskStatus
		terminal bool
	}{
		{valueobject.TaskStatusTodo, false},
		{valueobject.TaskStatusInProgress, false},
		{valueobject.TaskStatusInReview, false},
		{valueobject.TaskStatusDone, true},
		{valueobject.TaskStatusArchived, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

func TestTaskStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name  string
		from  valueobject.TaskStatus
		to    valueobject.TaskStatus
		canDo bool
	}{
		{"todo to in_progress", valueobject.TaskStatusTodo, valueobject.TaskStatusInProgress, true},
		{"todo to done", valueobject.TaskStatusTodo, valueobject.TaskStatusDone, true},
		{"done to archived", valueobject.TaskStatusDone, valueobject.TaskStatusArchived, true},
		{"done to todo", valueobject.TaskStatusDone, valueobject.TaskStatusTodo, false},
		{"archived to todo", valueobject.TaskStatusArchived, valueobject.TaskStatusTodo, false},
		{"archived to done", valueobject.TaskStatusArchived, valueobject.TaskStatusDone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.canDo {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.canDo)
			}
		})
	}
}

func TestAllTaskStatuses(t *testing.T) {
	statuses := valueobject.AllTaskStatuses()

	if len(statuses) != 5 {
		t.Errorf("AllTaskStatuses() returned %d statuses, want 5", len(statuses))
	}
}
