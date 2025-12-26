// Package task contains the task-related use cases.
package task

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

// Ensure TaskUseCase implements input.TaskService.
var _ input.TaskService = (*TaskUseCase)(nil)

// TaskUseCase implements the task-related use cases.
type TaskUseCase struct {
	taskRepo       output.TaskRepository
	userRepo       output.UserRepository
	labelRepo      output.LabelRepository
	cache          output.CacheRepository
	eventPublisher output.EventPublisher
	validator      *validation.Validator
	logger         *slog.Logger
}

// NewTaskUseCase creates a new TaskUseCase.
func NewTaskUseCase(
	taskRepo output.TaskRepository,
	userRepo output.UserRepository,
	labelRepo output.LabelRepository,
	cache output.CacheRepository,
	eventPublisher output.EventPublisher,
	logger *slog.Logger,
) *TaskUseCase {
	return &TaskUseCase{
		taskRepo:       taskRepo,
		userRepo:       userRepo,
		labelRepo:      labelRepo,
		cache:          cache,
		eventPublisher: eventPublisher,
		validator:      validation.GetValidator(),
		logger:         logger,
	}
}

// CreateTask creates a new task.
func (uc *TaskUseCase) CreateTask(ctx context.Context, input dto.CreateTaskInput) (*dto.TaskOutput, error) {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	// Parse priority
	priority, err := valueobject.ParsePriority(input.Priority)
	if err != nil {
		return nil, err
	}

	// Get creator ID from context (would be set by auth middleware)
	creatorID := getCreatorIDFromContext(ctx)

	// Create domain entity
	task, err := entity.NewTask(input.Title, input.Description, priority, creatorID)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if input.DueDate != nil {
		task.SetDueDate(*input.DueDate)
	}

	if input.AssigneeID != nil {
		// Verify assignee exists
		exists, err := uc.userRepo.ExistsByID(ctx, *input.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domainerror.ErrUserNotFound
		}
		if err := task.Assign(*input.AssigneeID); err != nil {
			return nil, err
		}
	}

	// Add labels
	for _, labelID := range input.LabelIDs {
		exists, err := uc.labelRepo.ExistsByID(ctx, labelID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domainerror.ErrLabelNotFound
		}
		task.AddLabel(labelID)
	}

	// Save to repository
	if err := uc.taskRepo.Save(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskCreated(task.ID, task.Title, task.Description, task.Priority, task.CreatorID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task created event", "taskId", task.ID, "error", err)
	}

	uc.logger.Info("task created", "taskId", task.ID, "title", task.Title)

	return dto.TaskFromEntity(task), nil
}

// UpdateTask updates an existing task.
func (uc *TaskUseCase) UpdateTask(ctx context.Context, id uuid.UUID, input dto.UpdateTaskInput) (*dto.TaskOutput, error) {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	// Fetch existing task
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Apply updates
	if input.Title != nil {
		if err := task.UpdateTitle(*input.Title); err != nil {
			return nil, err
		}
	}

	if input.Description != nil {
		task.UpdateDescription(*input.Description)
	}

	if input.Priority != nil {
		priority, err := valueobject.ParsePriority(*input.Priority)
		if err != nil {
			return nil, err
		}
		if err := task.UpdatePriority(priority); err != nil {
			return nil, err
		}
	}

	if input.DueDate != nil {
		task.SetDueDate(*input.DueDate)
	} else if input.ClearDueDate {
		task.ClearDueDate()
	}

	// Save updates
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Invalidate cache
	if uc.cache != nil {
		cacheKey := output.NewCacheKeyBuilder("app").Task(id.String())
		_ = uc.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	evt := event.NewTaskUpdated(task.ID, getCreatorIDFromContext(ctx))
	if input.Title != nil {
		evt = evt.WithTitle(*input.Title)
	}
	if input.Priority != nil {
		priority, _ := valueobject.ParsePriority(*input.Priority)
		evt = evt.WithPriority(priority)
	}
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task updated event", "taskId", task.ID, "error", err)
	}

	uc.logger.Info("task updated", "taskId", task.ID)

	return dto.TaskFromEntity(task), nil
}

// DeleteTask deletes a task by ID.
func (uc *TaskUseCase) DeleteTask(ctx context.Context, id uuid.UUID) error {
	// Check if task exists
	exists, err := uc.taskRepo.ExistsByID(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domainerror.ErrTaskNotFound
	}

	// Delete task
	if err := uc.taskRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if uc.cache != nil {
		cacheKey := output.NewCacheKeyBuilder("app").Task(id.String())
		_ = uc.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	evt := event.NewTaskDeleted(id, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task deleted event", "taskId", id, "error", err)
	}

	uc.logger.Info("task deleted", "taskId", id)

	return nil
}

// GetTask retrieves a task by ID.
func (uc *TaskUseCase) GetTask(ctx context.Context, id uuid.UUID) (*dto.TaskOutput, error) {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	output := dto.TaskFromEntity(task)

	// Enrich with labels
	labels, err := uc.labelRepo.FindByTaskID(ctx, id)
	if err == nil && len(labels) > 0 {
		output.Labels = make([]dto.LabelOutput, 0, len(labels))
		for _, label := range labels {
			output.Labels = append(output.Labels, *dto.LabelFromEntity(label))
		}
	}

	return output, nil
}

// ListTasks retrieves tasks with filtering and pagination.
func (uc *TaskUseCase) ListTasks(ctx context.Context, filter dto.TaskFilter, pagination dto.Pagination) (*dto.TaskListOutput, error) {
	// Validate input
	if err := uc.validator.Validate(filter); err != nil {
		return nil, err
	}
	if err := uc.validator.Validate(pagination); err != nil {
		return nil, err
	}

	// Convert filter
	outputFilter := output.TaskFilter{
		Search:     filter.Search,
		AssigneeID: filter.AssigneeID,
		CreatorID:  filter.CreatorID,
		LabelIDs:   filter.LabelIDs,
		IsOverdue:  filter.IsOverdue,
	}
	if filter.Status != nil {
		status := valueobject.TaskStatus(*filter.Status)
		outputFilter.Status = &status
	}
	if filter.Priority != nil {
		priority := valueobject.Priority(*filter.Priority)
		outputFilter.Priority = &priority
	}

	// Convert pagination
	outputPagination := output.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	// Fetch tasks
	tasks, paginatedResult, err := uc.taskRepo.FindAll(ctx, outputFilter, outputPagination)
	if err != nil {
		return nil, err
	}

	// Convert to output
	taskOutputs := make([]*dto.TaskOutput, 0, len(tasks))
	for _, task := range tasks {
		taskOutputs = append(taskOutputs, dto.TaskFromEntity(task))
	}

	return &dto.TaskListOutput{
		Tasks:      taskOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}, nil
}

// AssignTask assigns a task to a user.
func (uc *TaskUseCase) AssignTask(ctx context.Context, taskID, assigneeID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Verify assignee exists
	exists, err := uc.userRepo.ExistsByID(ctx, assigneeID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domainerror.ErrUserNotFound
	}

	// Assign
	if err := task.Assign(assigneeID); err != nil {
		return nil, err
	}

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskAssigned(taskID, assigneeID, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task assigned event", "taskId", taskID, "error", err)
	}

	uc.logger.Info("task assigned", "taskId", taskID, "assigneeId", assigneeID)

	return dto.TaskFromEntity(task), nil
}

// UnassignTask removes the assignee from a task.
func (uc *TaskUseCase) UnassignTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	previousAssignee := task.AssigneeID

	// Unassign
	task.Unassign()

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	if previousAssignee != nil {
		evt := event.NewTaskUnassigned(taskID, *previousAssignee, getCreatorIDFromContext(ctx))
		if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
			uc.logger.Error("failed to publish task unassigned event", "taskId", taskID, "error", err)
		}
	}

	uc.logger.Info("task unassigned", "taskId", taskID)

	return dto.TaskFromEntity(task), nil
}

// ChangeTaskStatus changes the status of a task.
func (uc *TaskUseCase) ChangeTaskStatus(ctx context.Context, taskID uuid.UUID, status string) (*dto.TaskOutput, error) {
	// Parse and validate status
	newStatus, err := valueobject.ParseTaskStatus(status)
	if err != nil {
		return nil, err
	}

	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	oldStatus := task.Status

	// Change status
	if err := task.ChangeStatus(newStatus); err != nil {
		return nil, err
	}

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskStatusChanged(taskID, oldStatus, newStatus, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task status changed event", "taskId", taskID, "error", err)
	}

	uc.logger.Info("task status changed", "taskId", taskID, "oldStatus", oldStatus, "newStatus", newStatus)

	return dto.TaskFromEntity(task), nil
}

// CompleteTask marks a task as done.
func (uc *TaskUseCase) CompleteTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Complete
	if err := task.Complete(); err != nil {
		return nil, err
	}

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskCompleted(taskID, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task completed event", "taskId", taskID, "error", err)
	}

	uc.logger.Info("task completed", "taskId", taskID)

	return dto.TaskFromEntity(task), nil
}

// ArchiveTask archives a completed task.
func (uc *TaskUseCase) ArchiveTask(ctx context.Context, taskID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Archive
	if err := task.Archive(); err != nil {
		return nil, err
	}

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskArchived(taskID, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task archived event", "taskId", taskID, "error", err)
	}

	uc.logger.Info("task archived", "taskId", taskID)

	return dto.TaskFromEntity(task), nil
}

// AddLabel adds a label to a task.
func (uc *TaskUseCase) AddLabel(ctx context.Context, taskID, labelID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Verify label exists
	exists, err := uc.labelRepo.ExistsByID(ctx, labelID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domainerror.ErrLabelNotFound
	}

	// Add label
	task.AddLabel(labelID)

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskLabelAdded(taskID, labelID, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task label added event", "taskId", taskID, "error", err)
	}

	return dto.TaskFromEntity(task), nil
}

// RemoveLabel removes a label from a task.
func (uc *TaskUseCase) RemoveLabel(ctx context.Context, taskID, labelID uuid.UUID) (*dto.TaskOutput, error) {
	// Fetch task
	task, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domainerror.ErrTaskNotFound
	}

	// Remove label
	task.RemoveLabel(labelID)

	// Save
	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewTaskLabelRemoved(taskID, labelID, getCreatorIDFromContext(ctx))
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task label removed event", "taskId", taskID, "error", err)
	}

	return dto.TaskFromEntity(task), nil
}

// SearchTasks performs a full-text search on tasks.
func (uc *TaskUseCase) SearchTasks(ctx context.Context, query string, pagination dto.Pagination) (*dto.TaskListOutput, error) {
	// Convert pagination
	outputPagination := output.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	tasks, paginatedResult, err := uc.taskRepo.Search(ctx, query, outputPagination)
	if err != nil {
		return nil, err
	}

	// Convert to output
	taskOutputs := make([]*dto.TaskOutput, 0, len(tasks))
	for _, task := range tasks {
		taskOutputs = append(taskOutputs, dto.TaskFromEntity(task))
	}

	return &dto.TaskListOutput{
		Tasks:      taskOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}, nil
}

// GetOverdueTasks retrieves tasks that are past their due date.
func (uc *TaskUseCase) GetOverdueTasks(ctx context.Context, pagination dto.Pagination) (*dto.TaskListOutput, error) {
	// Convert pagination
	outputPagination := output.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	tasks, paginatedResult, err := uc.taskRepo.FindOverdue(ctx, outputPagination)
	if err != nil {
		return nil, err
	}

	// Convert to output
	taskOutputs := make([]*dto.TaskOutput, 0, len(tasks))
	for _, task := range tasks {
		taskOutputs = append(taskOutputs, dto.TaskFromEntity(task))
	}

	return &dto.TaskListOutput{
		Tasks:      taskOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}, nil
}

// getCreatorIDFromContext extracts the user ID from context.
// In a real implementation, this would be set by the auth middleware.
func getCreatorIDFromContext(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("userID").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}
