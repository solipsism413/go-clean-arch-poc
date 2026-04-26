// Package handler provides HTTP handlers for the REST API.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

const maxAttachmentUploadSize = 32 << 20

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	taskService input.TaskService
	aclChecker  *acl.Checker
	logger      *slog.Logger
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService input.TaskService, aclChecker *acl.Checker, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		aclChecker:  aclChecker,
		logger:      logger,
	}
}

// Create handles POST /tasks
// @Summary Create a new task
// @Description Create a new task with the provided details
// @Tags tasks
// @Accept json
// @Produce json
// @Param task body dto.CreateTaskInput true "Task details"
// @Success 201 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusCreated, task)
}

// Get handles GET /tasks/{id}
// @Summary Get a task by ID
// @Description Get detailed information about a specific task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [get]
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionRead) {
		return
	}

	task, err := h.taskService.GetTask(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// List handles GET /tasks
// @Summary List tasks
// @Description Get a paginated list of tasks with optional filtering
// @Tags tasks
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param status query string false "Filter by status" Enums(TODO, IN_PROGRESS, IN_REVIEW, DONE, ARCHIVED)
// @Param priority query string false "Filter by priority" Enums(LOW, MEDIUM, HIGH, URGENT)
// @Param search query string false "Search in title and description"
// @Success 200 {object} presenter.Response{data=dto.TaskListOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks [get]
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := dto.TaskFilter{
		Search: r.URL.Query().Get("search"),
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		filter.Priority = &priority
	}

	pagination, err := parsePagination(r)
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	tasks, err := h.taskService.ListTasks(r.Context(), filter, pagination)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, tasks)
}

// Update handles PUT /tasks/{id}
// @Summary Update a task
// @Description Update an existing task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param task body dto.UpdateTaskInput true "Task updates"
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [put]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	var input dto.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.taskService.UpdateTask(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// Delete handles DELETE /tasks/{id}
// @Summary Delete a task
// @Description Delete a task by ID
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionDelete) {
		return
	}

	if err := h.taskService.DeleteTask(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Assign handles POST /tasks/{id}/assign
// @Summary Assign a task
// @Description Assign a task to a user
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param body body object{assigneeId=string} true "Assignee payload"
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/assign [post]
func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	var body struct {
		AssigneeID uuid.UUID `json:"assigneeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.taskService.AssignTask(r.Context(), id, body.AssigneeID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// Unassign handles POST /tasks/{id}/unassign
// @Summary Unassign a task
// @Description Remove the current assignee from a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/unassign [post]
func (h *TaskHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	task, err := h.taskService.UnassignTask(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// Complete handles POST /tasks/{id}/complete
// @Summary Complete a task
// @Description Mark a task as done
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/complete [post]
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	task, err := h.taskService.CompleteTask(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// Archive handles POST /tasks/{id}/archive
// @Summary Archive a task
// @Description Archive a task so it no longer appears as active work
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/archive [post]
func (h *TaskHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	task, err := h.taskService.ArchiveTask(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// ChangeStatus handles POST /tasks/{id}/status
// @Summary Change task status
// @Description Change a task status explicitly
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param body body object{status=string} true "Status payload"
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/status [post]
func (h *TaskHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.taskService.ChangeTaskStatus(r.Context(), id, body.Status)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// AddLabel handles POST /tasks/{id}/labels/{labelId}
// @Summary Add label to task
// @Description Attach an existing label to a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param labelId path string true "Label ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/labels/{labelId} [post]
func (h *TaskHandler) AddLabel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	labelID, err := uuid.Parse(r.PathValue("labelId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid label ID", err)
		return
	}

	task, err := h.taskService.AddLabel(r.Context(), id, labelID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// RemoveLabel handles DELETE /tasks/{id}/labels/{labelId}
// @Summary Remove label from task
// @Description Detach a label from a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param labelId path string true "Label ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/labels/{labelId} [delete]
func (h *TaskHandler) RemoveLabel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	labelID, err := uuid.Parse(r.PathValue("labelId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid label ID", err)
		return
	}

	task, err := h.taskService.RemoveLabel(r.Context(), id, labelID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, task)
}

// UploadAttachment handles POST /tasks/{id}/attachments
// @Summary Upload an attachment to a task
// @Description Upload a file attachment to a specific task
// @Tags tasks
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param file formData file true "File to upload"
// @Success 201 {object} presenter.Response{data=dto.TaskAttachmentOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments [post]
func (h *TaskHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentUploadSize)

	if err := r.ParseMultipartForm(maxAttachmentUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			presenter.Error(w, http.StatusRequestEntityTooLarge, "Attachment exceeds maximum size of 32 MB", err)
			return
		}
		presenter.Error(w, http.StatusBadRequest, "Invalid multipart form", err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Failed to read file", err)
		return
	}
	defer file.Close()

	attachment, err := h.taskService.UploadTaskAttachment(r.Context(), id, header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusCreated, attachment)
}

// DownloadAttachment handles GET /tasks/{id}/attachments/{attachmentId}
// @Summary Download a task attachment
// @Description Download a file attachment by ID
// @Tags tasks
// @Param id path string true "Task ID" format(uuid)
// @Param attachmentId path string true "Attachment ID" format(uuid)
// @Success 200 {file} binary
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments/{attachmentId} [get]
func (h *TaskHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionRead) {
		return
	}

	attachmentID, err := uuid.Parse(r.PathValue("attachmentId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid attachment ID", err)
		return
	}

	reader, attachment, err := h.taskService.DownloadTaskAttachment(r.Context(), id, attachmentID)
	if err != nil {
		handleError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", attachment.ContentType)
	if disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename}); disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	if attachment.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(attachment.SizeBytes, 10))
	}

	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("failed to stream attachment", "error", err)
	}
}

// ListAttachments handles GET /tasks/{id}/attachments
// @Summary List task attachments
// @Description Get all attachments for a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.TaskAttachmentListOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments [get]
func (h *TaskHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionRead) {
		return
	}

	attachments, err := h.taskService.ListTaskAttachments(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, attachments)
}

// DeleteAttachment handles DELETE /tasks/{id}/attachments/{attachmentId}
// @Summary Delete a task attachment
// @Description Remove a file attachment from a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param attachmentId path string true "Attachment ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments/{attachmentId} [delete]
func (h *TaskHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
		return
	}

	attachmentID, err := uuid.Parse(r.PathValue("attachmentId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid attachment ID", err)
		return
	}

	if err := h.taskService.DeleteTaskAttachment(r.Context(), id, attachmentID); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// checkACL is a helper method to perform manual ACL checks in handlers.
func (h *TaskHandler) checkACL(w http.ResponseWriter, r *http.Request, resourceID uuid.UUID, permission acl.Permission) bool {
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
		return false
	}

	hasAccess, err := h.aclChecker.CanAccess(r.Context(), claims.UserID, claims.RoleIDs, entity.ResourceTypeTask, resourceID, permission)
	if err != nil {
		h.logger.Error("failed to check ACL access", "error", err, "taskId", resourceID, "userId", claims.UserID)
		presenter.Error(w, http.StatusInternalServerError, "Internal server error", nil)
		return false
	}

	if !hasAccess {
		presenter.Error(w, http.StatusForbidden, "You do not have permission for this task", nil)
		return false
	}

	return true
}
