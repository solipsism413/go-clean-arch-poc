// Package handler provides HTTP handlers for the REST API.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// TaskQueryHandler handles task-related query HTTP requests.
type TaskQueryHandler struct {
	taskService input.TaskService
	logger      *slog.Logger
}

// NewTaskQueryHandler creates a new TaskQueryHandler.
func NewTaskQueryHandler(taskService input.TaskService, logger *slog.Logger) *TaskQueryHandler {
	return &TaskQueryHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// Search handles GET /tasks/search
// @Summary Search tasks
// @Description Search tasks using full-text search
// @Tags tasks
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Success 200 {object} presenter.Response{data=dto.TaskListOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/search [get]
func (h *TaskQueryHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		presenter.Error(w, http.StatusBadRequest, "Missing search query", nil)
		return
	}

	pagination := parsePagination(r)

	tasks, err := h.taskService.SearchTasks(r.Context(), query, pagination)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, tasks)
}

// Overdue handles GET /tasks/overdue
// @Summary Get overdue tasks
// @Description Get a paginated list of overdue tasks
// @Tags tasks
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Success 200 {object} presenter.Response{data=dto.TaskListOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /tasks/overdue [get]
func (h *TaskQueryHandler) Overdue(w http.ResponseWriter, r *http.Request) {
	pagination := parsePagination(r)

	tasks, err := h.taskService.GetOverdueTasks(r.Context(), pagination)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, tasks)
}
