package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/middleware"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userService input.UserService
	logger      *slog.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService input.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// List handles GET /users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := dto.UserFilter{
		Search: r.URL.Query().Get("search"),
	}

	pagination := parsePagination(r)

	users, err := h.userService.ListUsers(r.Context(), filter, pagination)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, users)
}

// Get handles GET /users/{id}
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, user)
}

// Me handles GET /users/me
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		presenter.Error(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	user, err := h.userService.GetUser(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, user)
}

// Update handles PUT /users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var input dto.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, user)
}

// Delete handles DELETE /users/{id}
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	if err := h.userService.DeleteUser(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignRole handles POST /users/{id}/roles/{roleId}
func (h *UserHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	roleID, err := uuid.Parse(r.PathValue("roleId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid role ID", err)
		return
	}

	user, err := h.userService.AssignRole(r.Context(), id, roleID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, user)
}

// RemoveRole handles DELETE /users/{id}/roles/{roleId}
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	roleID, err := uuid.Parse(r.PathValue("roleId"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid role ID", err)
		return
	}

	user, err := h.userService.RemoveRole(r.Context(), id, roleID)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, user)
}
