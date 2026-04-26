// Package task contains the task-related use cases.
package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

const (
	cachePrefix = "app"

	// Cache TTLs.
	entityCacheTTL = 5 * time.Minute
	listCacheTTL   = 2 * time.Minute
)

// Ensure TaskUseCase implements input.TaskService.
var _ input.TaskService = (*TaskUseCase)(nil)

// TaskUseCase implements the task-related use cases.
type TaskUseCase struct {
	taskRepo       output.TaskRepository
	attachmentRepo output.TaskAttachmentRepository
	userRepo       output.UserRepository
	labelRepo      output.LabelRepository
	fileStorage    output.FileStorage
	cache          output.CacheRepository
	eventPublisher output.EventPublisher
	tm             output.TransactionManager
	validator      validation.Validator
	logger         *slog.Logger
}

// NewTaskUseCase creates a new TaskUseCase.
func NewTaskUseCase(
	taskRepo output.TaskRepository,
	attachmentRepo output.TaskAttachmentRepository,
	userRepo output.UserRepository,
	labelRepo output.LabelRepository,
	fileStorage output.FileStorage,
	cache output.CacheRepository,
	eventPublisher output.EventPublisher,
	tm output.TransactionManager,
	validator validation.Validator,
	logger *slog.Logger,
) *TaskUseCase {
	return &TaskUseCase{
		taskRepo:       taskRepo,
		attachmentRepo: attachmentRepo,
		userRepo:       userRepo,
		labelRepo:      labelRepo,
		fileStorage:    fileStorage,
		cache:          cache,
		eventPublisher: eventPublisher,
		tm:             tm,
		validator:      validator,
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

	// Invalidate list caches
	uc.invalidateTaskListCaches(ctx)

	// Publish event
	evt := event.NewTaskCreated(task.ID, task.Title, task.Description, task.Priority, task.CreatorID)
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, id)

	// Publish event
	evt := event.NewTaskUpdated(task.ID, getCreatorIDFromContext(ctx))
	if input.Title != nil {
		evt = evt.WithTitle(*input.Title)
	}
	if input.Priority != nil {
		priority, _ := valueobject.ParsePriority(*input.Priority)
		evt = evt.WithPriority(priority)
	}
	uc.publishTaskEvent(ctx, evt)

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

	attachments, err := uc.attachmentRepo.FindAttachmentsByTaskID(ctx, id)
	if err != nil {
		return err
	}
	for _, attachment := range attachments {
		if err := uc.deleteAttachmentObject(ctx, id, attachment.ID, attachment.S3Key); err != nil {
			return err
		}
	}

	// Delete task
	if err := uc.taskRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, id)

	// Publish event
	evt := event.NewTaskDeleted(id, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

	uc.logger.Info("task deleted", "taskId", id)

	return nil
}

// GetTask retrieves a task by ID with cache-aside.
func (uc *TaskUseCase) GetTask(ctx context.Context, id uuid.UUID) (*dto.TaskOutput, error) {
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).Task(id.String())

	// Try cache first
	if uc.cache != nil {
		var cached dto.TaskOutput
		if err := uc.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
			uc.logger.Debug("task cache hit", "taskId", id)
			return &cached, nil
		}
	}

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

	// Store in cache
	if uc.cache != nil {
		_ = uc.cache.SetJSON(ctx, cacheKey, output, entityCacheTTL)
	}

	return output, nil
}

// ListTasks retrieves tasks with filtering and pagination with cache-aside.
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

	// Build cache key from filter + pagination hash
	filterHash := uc.buildTaskListCacheKey(outputFilter, outputPagination)
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).TaskList(filterHash)

	// Try cache first
	if uc.cache != nil {
		var cached dto.TaskListOutput
		if err := uc.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
			uc.logger.Debug("task list cache hit", "filterHash", filterHash)
			return &cached, nil
		}
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

	result := &dto.TaskListOutput{
		Tasks:      taskOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}

	// Store in cache
	if uc.cache != nil {
		_ = uc.cache.SetJSON(ctx, cacheKey, result, listCacheTTL)
	}

	return result, nil
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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskAssigned(taskID, assigneeID, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	if previousAssignee != nil {
		evt := event.NewTaskUnassigned(taskID, *previousAssignee, getCreatorIDFromContext(ctx))
		uc.publishTaskEvent(ctx, evt)
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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskStatusChanged(taskID, oldStatus, newStatus, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskCompleted(taskID, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskArchived(taskID, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskLabelAdded(taskID, labelID, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

	// Invalidate caches
	uc.invalidateTaskCaches(ctx, taskID)

	// Publish event
	evt := event.NewTaskLabelRemoved(taskID, labelID, getCreatorIDFromContext(ctx))
	uc.publishTaskEvent(ctx, evt)

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

// UploadTaskAttachment uploads a file attachment to a task.
func (uc *TaskUseCase) UploadTaskAttachment(ctx context.Context, taskID uuid.UUID, filename, contentType string, reader io.Reader) (*dto.TaskAttachmentOutput, error) {
	if err := uc.ensureFileStorageAvailable(); err != nil {
		return nil, err
	}

	// Verify task exists
	exists, err := uc.taskRepo.ExistsByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domainerror.ErrTaskNotFound
	}

	// Sanitize filename to prevent path traversal in storage key
	safeFilename := sanitizeAttachmentFilename(filename)
	if safeFilename == "" || safeFilename == "." || safeFilename == "/" {
		return nil, domainerror.NewDomainError(domainerror.CodeValidation, "invalid filename")
	}

	// Get uploader ID from context
	uploadedBy := getCreatorIDFromContext(ctx)

	// Build a unique S3 key so duplicate filenames do not overwrite each other.
	keyBuilder := output.NewFileKeyBuilder("attachments")
	storageName := uuid.NewString() + "-" + safeFilename
	s3Key := keyBuilder.TaskAttachment(taskID.String(), storageName)

	// Upload to S3/MinIO
	storageMeta, err := uc.fileStorage.Upload(ctx, s3Key, reader, output.UploadOptions{
		ContentType: contentType,
		Metadata: map[string]string{
			"task_id":     taskID.String(),
			"uploaded_by": uploadedBy.String(),
			"filename":    safeFilename,
		},
	})
	if err != nil {
		uc.logger.Error("failed to upload attachment to storage", "taskId", taskID, "error", err)
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create attachment entity
	attachment := entity.NewTaskAttachment(
		taskID,
		safeFilename,
		storageMeta.Key,
		contentType,
		storageMeta.Size,
		uploadedBy,
	)

	// Save attachment record
	if err := uc.attachmentRepo.SaveAttachment(ctx, attachment); err != nil {
		uc.logger.Error("failed to save attachment record", "taskId", taskID, "error", err)
		uc.cleanupUploadedAttachmentObject(ctx, taskID, attachment.ID, storageMeta.Key, err)
		return nil, fmt.Errorf("failed to save attachment: %w", err)
	}

	uc.logger.Info("task attachment uploaded", "taskId", taskID, "attachmentId", attachment.ID, "filename", safeFilename)

	return &dto.TaskAttachmentOutput{
		ID:          attachment.ID,
		TaskID:      attachment.TaskID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		SizeBytes:   attachment.SizeBytes,
		UploadedBy:  attachment.UploadedBy,
		CreatedAt:   attachment.CreatedAt,
	}, nil
}

// DownloadTaskAttachment retrieves a file attachment by ID, verifying it belongs to the given task.
func (uc *TaskUseCase) DownloadTaskAttachment(ctx context.Context, taskID, attachmentID uuid.UUID) (io.ReadCloser, *dto.TaskAttachmentOutput, error) {
	if err := uc.ensureFileStorageAvailable(); err != nil {
		return nil, nil, err
	}

	// Find attachment record
	attachment, err := uc.attachmentRepo.FindAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	if attachment == nil {
		return nil, nil, domainerror.ErrAttachmentNotFound
	}

	// Verify attachment belongs to the requested task
	if attachment.TaskID != taskID {
		return nil, nil, domainerror.ErrAttachmentNotFound
	}

	// Download from S3/MinIO
	reader, _, err := uc.fileStorage.Download(ctx, attachment.S3Key)
	if err != nil {
		uc.logger.Error("failed to download attachment from storage", "attachmentId", attachmentID, "error", err)
		return nil, nil, fmt.Errorf("failed to download file: %w", err)
	}

	output := &dto.TaskAttachmentOutput{
		ID:          attachment.ID,
		TaskID:      attachment.TaskID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		SizeBytes:   attachment.SizeBytes,
		UploadedBy:  attachment.UploadedBy,
		CreatedAt:   attachment.CreatedAt,
	}

	return reader, output, nil
}

// ListTaskAttachments lists all attachments for a task.
func (uc *TaskUseCase) ListTaskAttachments(ctx context.Context, taskID uuid.UUID) (*dto.TaskAttachmentListOutput, error) {
	// Verify task exists
	exists, err := uc.taskRepo.ExistsByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domainerror.ErrTaskNotFound
	}

	attachments, err := uc.attachmentRepo.FindAttachmentsByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	outputs := make([]dto.TaskAttachmentOutput, 0, len(attachments))
	for _, attachment := range attachments {
		outputs = append(outputs, dto.TaskAttachmentOutput{
			ID:          attachment.ID,
			TaskID:      attachment.TaskID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			SizeBytes:   attachment.SizeBytes,
			UploadedBy:  attachment.UploadedBy,
			CreatedAt:   attachment.CreatedAt,
		})
	}

	return &dto.TaskAttachmentListOutput{Attachments: outputs}, nil
}

// DeleteTaskAttachment removes an attachment by ID, verifying it belongs to the given task.
func (uc *TaskUseCase) DeleteTaskAttachment(ctx context.Context, taskID, attachmentID uuid.UUID) error {
	// Find attachment record
	attachment, err := uc.attachmentRepo.FindAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if attachment == nil {
		return domainerror.ErrAttachmentNotFound
	}

	// Verify attachment belongs to the requested task
	if attachment.TaskID != taskID {
		return domainerror.ErrAttachmentNotFound
	}

	if err := uc.deleteAttachmentObject(ctx, taskID, attachmentID, attachment.S3Key); err != nil {
		return err
	}

	// Delete attachment record
	if err := uc.attachmentRepo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}

	uc.logger.Info("task attachment deleted", "attachmentId", attachmentID)

	return nil
}

func sanitizeAttachmentFilename(filename string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")
	return path.Base(normalized)
}

func (uc *TaskUseCase) ensureFileStorageAvailable() error {
	if !uc.hasFileStorage() {
		return domainerror.NewDomainError(domainerror.CodeInternalError, "file storage is unavailable")
	}
	return nil
}

func (uc *TaskUseCase) publishAttachmentCleanupRequested(ctx context.Context, taskID, attachmentID uuid.UUID, objectKey, reason string, cause error) error {
	if !uc.hasEventPublisher() {
		uc.logger.Error("attachment cleanup retry unavailable",
			"taskId", taskID,
			"attachmentId", attachmentID,
			"objectKey", objectKey,
			"reason", reason,
			"error", cause,
		)
		return domainerror.NewDomainError(domainerror.CodeInternalError, "attachment cleanup is unavailable")
	}

	evt := event.NewTaskAttachmentCleanupRequested(taskID, attachmentID, objectKey)
	uc.logger.Warn("queueing attachment cleanup retry",
		"taskId", taskID,
		"attachmentId", attachmentID,
		"objectKey", objectKey,
		"reason", reason,
		"error", cause,
	)
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to queue attachment cleanup retry",
			"taskId", taskID,
			"attachmentId", attachmentID,
			"objectKey", objectKey,
			"reason", reason,
			"error", err,
		)
		return domainerror.Wrap(domainerror.CodeInternalError, "failed to publish attachment cleanup request", err)
	}
	uc.logger.Info("attachment cleanup retry queued",
		"taskId", taskID,
		"attachmentId", attachmentID,
		"objectKey", objectKey,
		"reason", reason,
		"eventId", evt.ID,
	)
	return nil
}

func (uc *TaskUseCase) deleteAttachmentObject(ctx context.Context, taskID, attachmentID uuid.UUID, objectKey string) error {
	if !uc.hasFileStorage() {
		return uc.publishAttachmentCleanupRequested(ctx, taskID, attachmentID, objectKey, "storage_unavailable", nil)
	}

	if err := uc.fileStorage.Delete(ctx, objectKey); err != nil {
		uc.logger.Error("failed to delete attachment from storage", "taskId", taskID, "attachmentId", attachmentID, "objectKey", objectKey, "error", err)
		return uc.publishAttachmentCleanupRequested(ctx, taskID, attachmentID, objectKey, "delete_failed", err)
	}

	return nil
}

func (uc *TaskUseCase) cleanupUploadedAttachmentObject(ctx context.Context, taskID, attachmentID uuid.UUID, objectKey string, cause error) {
	if !uc.hasFileStorage() {
		uc.logger.Error("uploaded attachment cleanup skipped because storage is unavailable",
			"taskId", taskID,
			"attachmentId", attachmentID,
			"objectKey", objectKey,
			"error", cause,
		)
		return
	}

	if err := uc.fileStorage.Delete(ctx, objectKey); err != nil {
		uc.logger.Error("failed to roll back uploaded attachment from storage",
			"taskId", taskID,
			"attachmentId", attachmentID,
			"objectKey", objectKey,
			"error", err,
		)
		if publishErr := uc.publishAttachmentCleanupRequested(ctx, taskID, attachmentID, objectKey, "save_attachment_failed", err); publishErr != nil {
			uc.logger.Error("failed to queue uploaded attachment cleanup retry",
				"taskId", taskID,
				"attachmentId", attachmentID,
				"objectKey", objectKey,
				"error", publishErr,
			)
		}
	}
}

func (uc *TaskUseCase) publishTaskEvent(ctx context.Context, evt event.Event) {
	if !uc.hasEventPublisher() {
		uc.logger.Warn("task event publisher unavailable", "eventType", evt.EventType(), "aggregateId", evt.AggregateID())
		return
	}
	if err := uc.eventPublisher.Publish(ctx, output.TopicTaskEvents, evt); err != nil {
		uc.logger.Error("failed to publish task event", "eventType", evt.EventType(), "aggregateId", evt.AggregateID(), "error", err)
	}
}

func (uc *TaskUseCase) hasFileStorage() bool {
	return !isNilDependency(uc.fileStorage)
}

func (uc *TaskUseCase) hasEventPublisher() bool {
	return !isNilDependency(uc.eventPublisher)
}

func isNilDependency(dep any) bool {
	if dep == nil {
		return true
	}
	v := reflect.ValueOf(dep)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// invalidateTaskCaches removes the task cache and all list caches.
func (uc *TaskUseCase) invalidateTaskCaches(ctx context.Context, taskID uuid.UUID) {
	if uc.cache == nil {
		return
	}
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).Task(taskID.String())
	_ = uc.cache.Delete(ctx, cacheKey)
	uc.invalidateTaskListCaches(ctx)
}

// invalidateTaskListCaches removes all task list caches.
func (uc *TaskUseCase) invalidateTaskListCaches(ctx context.Context) {
	if uc.cache == nil {
		return
	}
	listPattern := output.NewCacheKeyBuilder(cachePrefix).TaskList("") + "*"
	_ = uc.cache.DeletePattern(ctx, listPattern)
}

// buildTaskListCacheKey creates a deterministic hash from filter and pagination.
func (uc *TaskUseCase) buildTaskListCacheKey(filter output.TaskFilter, pagination output.Pagination) string {
	h := sha256.New()
	fmt.Fprintf(h, "search=%s|", filter.Search)
	if filter.AssigneeID != nil {
		fmt.Fprintf(h, "assignee=%s|", filter.AssigneeID.String())
	}
	if filter.CreatorID != nil {
		fmt.Fprintf(h, "creator=%s|", filter.CreatorID.String())
	}
	fmt.Fprintf(h, "overdue=%v|", filter.IsOverdue)
	if len(filter.LabelIDs) > 0 {
		// Sort for deterministic ordering
		sorted := make([]uuid.UUID, len(filter.LabelIDs))
		copy(sorted, filter.LabelIDs)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].String() < sorted[j].String()
		})
		for _, id := range sorted {
			fmt.Fprintf(h, "label=%s|", id.String())
		}
	}
	if filter.Status != nil {
		fmt.Fprintf(h, "status=%s|", *filter.Status)
	}
	if filter.Priority != nil {
		fmt.Fprintf(h, "priority=%s|", *filter.Priority)
	}
	fmt.Fprintf(h, "page=%d|size=%d|sort=%s|desc=%v",
		pagination.Page, pagination.PageSize, pagination.SortBy, pagination.SortDesc)
	return hex.EncodeToString(h.Sum(nil))
}

// getCreatorIDFromContext extracts the user ID from context.
func getCreatorIDFromContext(ctx context.Context) uuid.UUID {
	if claims := auth.GetClaimsFromContext(ctx); claims != nil {
		return claims.UserID
	}
	return uuid.Nil
}
