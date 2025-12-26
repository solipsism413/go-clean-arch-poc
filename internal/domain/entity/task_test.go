package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

func TestNewTask(t *testing.T) {
	creatorID := uuid.New()

	tests := []struct {
		name        string
		title       string
		description string
		priority    valueobject.Priority
		wantErr     error
	}{
		{
			name:        "valid task",
			title:       "Test Task",
			description: "Test Description",
			priority:    valueobject.PriorityMedium,
			wantErr:     nil,
		},
		{
			name:        "empty title",
			title:       "",
			description: "Test Description",
			priority:    valueobject.PriorityMedium,
			wantErr:     entity.ErrEmptyTitle,
		},
		{
			name:        "invalid priority",
			title:       "Test Task",
			description: "Test Description",
			priority:    valueobject.Priority("INVALID"),
			wantErr:     entity.ErrInvalidPriority,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := entity.NewTask(tt.title, tt.description, tt.priority, creatorID)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewTask() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewTask() unexpected error = %v", err)
				return
			}

			if task.Title != tt.title {
				t.Errorf("NewTask() title = %v, want %v", task.Title, tt.title)
			}
			if task.Description != tt.description {
				t.Errorf("NewTask() description = %v, want %v", task.Description, tt.description)
			}
			if task.Priority != tt.priority {
				t.Errorf("NewTask() priority = %v, want %v", task.Priority, tt.priority)
			}
			if task.Status != valueobject.TaskStatusTodo {
				t.Errorf("NewTask() status = %v, want %v", task.Status, valueobject.TaskStatusTodo)
			}
			if task.CreatorID != creatorID {
				t.Errorf("NewTask() creatorID = %v, want %v", task.CreatorID, creatorID)
			}
			if task.ID == uuid.Nil {
				t.Error("NewTask() ID should not be nil")
			}
		})
	}
}

func TestTask_Complete(t *testing.T) {
	creatorID := uuid.New()

	t.Run("complete todo task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)

		err := task.Complete()

		if err != nil {
			t.Errorf("Complete() error = %v", err)
		}
		if task.Status != valueobject.TaskStatusDone {
			t.Errorf("Complete() status = %v, want %v", task.Status, valueobject.TaskStatusDone)
		}
	})

	t.Run("cannot complete archived task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		task.Status = valueobject.TaskStatusArchived

		err := task.Complete()

		if err != entity.ErrTaskArchived {
			t.Errorf("Complete() error = %v, want %v", err, entity.ErrTaskArchived)
		}
	})
}

func TestTask_Archive(t *testing.T) {
	creatorID := uuid.New()

	t.Run("archive done task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		task.Status = valueobject.TaskStatusDone

		err := task.Archive()

		if err != nil {
			t.Errorf("Archive() error = %v", err)
		}
		if task.Status != valueobject.TaskStatusArchived {
			t.Errorf("Archive() status = %v, want %v", task.Status, valueobject.TaskStatusArchived)
		}
	})

	t.Run("cannot archive todo task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)

		err := task.Archive()

		if err != entity.ErrTaskNotDone {
			t.Errorf("Archive() error = %v, want %v", err, entity.ErrTaskNotDone)
		}
	})
}

func TestTask_Assign(t *testing.T) {
	creatorID := uuid.New()
	assigneeID := uuid.New()

	t.Run("assign task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)

		err := task.Assign(assigneeID)

		if err != nil {
			t.Errorf("Assign() error = %v", err)
		}
		if task.AssigneeID == nil || *task.AssigneeID != assigneeID {
			t.Errorf("Assign() assigneeID = %v, want %v", task.AssigneeID, assigneeID)
		}
	})

	t.Run("cannot assign archived task", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		task.Status = valueobject.TaskStatusArchived

		err := task.Assign(assigneeID)

		if err != entity.ErrTaskArchived {
			t.Errorf("Assign() error = %v, want %v", err, entity.ErrTaskArchived)
		}
	})
}

func TestTask_Unassign(t *testing.T) {
	creatorID := uuid.New()
	assigneeID := uuid.New()

	task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
	_ = task.Assign(assigneeID)

	task.Unassign()

	if task.AssigneeID != nil {
		t.Error("Unassign() should set assigneeID to nil")
	}
}

func TestTask_ChangeStatus(t *testing.T) {
	creatorID := uuid.New()

	tests := []struct {
		name    string
		initial valueobject.TaskStatus
		target  valueobject.TaskStatus
		wantErr error
	}{
		{
			name:    "todo to in_progress",
			initial: valueobject.TaskStatusTodo,
			target:  valueobject.TaskStatusInProgress,
			wantErr: nil,
		},
		{
			name:    "in_progress to in_review",
			initial: valueobject.TaskStatusInProgress,
			target:  valueobject.TaskStatusInReview,
			wantErr: nil,
		},
		{
			name:    "in_review to done",
			initial: valueobject.TaskStatusInReview,
			target:  valueobject.TaskStatusDone,
			wantErr: nil,
		},
		{
			name:    "done to archived",
			initial: valueobject.TaskStatusDone,
			target:  valueobject.TaskStatusArchived,
			wantErr: nil,
		},
		{
			name:    "cannot change archived",
			initial: valueobject.TaskStatusArchived,
			target:  valueobject.TaskStatusTodo,
			wantErr: entity.ErrTaskArchived,
		},
		{
			name:    "done cannot go back to todo",
			initial: valueobject.TaskStatusDone,
			target:  valueobject.TaskStatusTodo,
			wantErr: entity.ErrInvalidStatusTransition,
		},
		{
			name:    "invalid status",
			initial: valueobject.TaskStatusTodo,
			target:  valueobject.TaskStatus("INVALID"),
			wantErr: entity.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
			task.Status = tt.initial

			err := task.ChangeStatus(tt.target)

			if err != tt.wantErr {
				t.Errorf("ChangeStatus() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_Labels(t *testing.T) {
	creatorID := uuid.New()
	labelID := uuid.New()

	t.Run("add label", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)

		task.AddLabel(labelID)

		if len(task.Labels) != 1 || task.Labels[0] != labelID {
			t.Errorf("AddLabel() labels = %v, want [%v]", task.Labels, labelID)
		}
	})

	t.Run("add duplicate label", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		task.AddLabel(labelID)
		task.AddLabel(labelID)

		if len(task.Labels) != 1 {
			t.Errorf("AddLabel() should not add duplicate, got %d labels", len(task.Labels))
		}
	})

	t.Run("remove label", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		task.AddLabel(labelID)

		task.RemoveLabel(labelID)

		if len(task.Labels) != 0 {
			t.Error("RemoveLabel() should remove the label")
		}
	})
}

func TestTask_IsOverdue(t *testing.T) {
	creatorID := uuid.New()

	t.Run("no due date", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)

		if task.IsOverdue() {
			t.Error("IsOverdue() should return false when no due date")
		}
	})

	t.Run("past due date", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		pastDate := time.Now().Add(-24 * time.Hour)
		task.SetDueDate(pastDate)

		if !task.IsOverdue() {
			t.Error("IsOverdue() should return true for past due date")
		}
	})

	t.Run("future due date", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		futureDate := time.Now().Add(24 * time.Hour)
		task.SetDueDate(futureDate)

		if task.IsOverdue() {
			t.Error("IsOverdue() should return false for future due date")
		}
	})

	t.Run("completed task not overdue", func(t *testing.T) {
		task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
		pastDate := time.Now().Add(-24 * time.Hour)
		task.SetDueDate(pastDate)
		task.Status = valueobject.TaskStatusDone

		if task.IsOverdue() {
			t.Error("IsOverdue() should return false for completed task")
		}
	})
}

func TestTask_CanBeModifiedBy(t *testing.T) {
	creatorID := uuid.New()
	assigneeID := uuid.New()
	otherUserID := uuid.New()

	task, _ := entity.NewTask("Test", "Desc", valueobject.PriorityMedium, creatorID)
	_ = task.Assign(assigneeID)

	if !task.CanBeModifiedBy(creatorID) {
		t.Error("Creator should be able to modify the task")
	}

	if !task.CanBeModifiedBy(assigneeID) {
		t.Error("Assignee should be able to modify the task")
	}

	if task.CanBeModifiedBy(otherUserID) {
		t.Error("Other user should not be able to modify the task")
	}
}
