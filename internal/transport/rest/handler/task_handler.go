// Package handler provides HTTP handlers for the REST API.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	taskService input.TaskService
	logger      *slog.Logger
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService input.TaskService, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
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

	pagination := parsePagination(r)

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

	if err := h.taskService.DeleteTask(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Assign handles POST /tasks/{id}/assign
func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) AddLabel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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
func (h *TaskHandler) RemoveLabel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid task ID", err)
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

// handleError maps domain errors to HTTP responses.
func handleError(w http.ResponseWriter, err error) {
	if validationErr, ok := err.(*validation.ValidationError); ok {
		presenter.ValidationError(w, validationErr)
		return
	}

	if domainerror.IsNotFoundError(err) {
		presenter.Error(w, http.StatusNotFound, err.Error(), nil)
		return
	}

	if domainerror.IsValidationError(err) {
		presenter.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if domainerror.IsUnauthorizedError(err) {
		presenter.Error(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	if domainerror.IsForbiddenError(err) {
		presenter.Error(w, http.StatusForbidden, err.Error(), nil)
		return
	}

	presenter.Error(w, http.StatusInternalServerError, "Internal server error", err)
}

// parsePagination extracts pagination parameters from the request.
func parsePagination(r *http.Request) dto.Pagination {
	pagination := dto.DefaultPagination()

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := parsePositiveInt(page); err == nil {
			pagination.Page = p
		}
	}

	if pageSize := r.URL.Query().Get("pageSize"); pageSize != "" {
		if ps, err := parsePositiveInt(pageSize); err == nil && ps <= 100 {
			pagination.PageSize = ps
		}
	}

	if sortBy := r.URL.Query().Get("sortBy"); sortBy != "" {
		pagination.SortBy = sortBy
	}

	if sortDesc := r.URL.Query().Get("sortDesc"); sortDesc == "true" {
		pagination.SortDesc = true
	}

	return pagination
}

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 1, nil
	}
	return n, nil
}
