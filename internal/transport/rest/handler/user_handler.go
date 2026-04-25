package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/middleware"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userService input.UserService
	aclChecker  *acl.Checker
	logger      *slog.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService input.UserService, aclChecker *acl.Checker, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		aclChecker:  aclChecker,
		logger:      logger,
	}
}

// List handles GET /users
// @Summary List users
// @Description Get a paginated list of users with optional search
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param search query string false "Search by name or email"
// @Success 200 {object} presenter.Response{data=dto.UserListOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := dto.UserFilter{
		Search: r.URL.Query().Get("search"),
	}

	pagination, err := parsePagination(r)
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	users, err := h.userService.ListUsers(r.Context(), filter, pagination)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, users)
}

// Get handles GET /users/{id}
// @Summary Get user by ID
// @Description Get detailed information about a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.UserOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionRead) {
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
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} presenter.Response{data=dto.UserOutput}
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/me [get]
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
// @Summary Update user
// @Description Update an existing user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param user body dto.UpdateUserInput true "User updates"
// @Success 200 {object} presenter.Response{data=dto.UserOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionWrite) {
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
// @Summary Delete user
// @Description Delete a user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	if !h.checkACL(w, r, id, acl.PermissionDelete) {
		return
	}

	if err := h.userService.DeleteUser(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignRole handles POST /users/{id}/roles/{roleId}
// @Summary Assign role to user
// @Description Attach an existing role to a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param roleId path string true "Role ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.UserOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/roles/{roleId} [post]
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
// @Summary Remove role from user
// @Description Detach a role from a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param roleId path string true "Role ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.UserOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/roles/{roleId} [delete]
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

// checkACL is a helper method to perform manual ACL checks in handlers.
func (h *UserHandler) checkACL(w http.ResponseWriter, r *http.Request, resourceID uuid.UUID, permission acl.Permission) bool {
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
		return false
	}

	hasAccess, err := h.aclChecker.CanAccess(r.Context(), claims.UserID, claims.RoleIDs, entity.ResourceTypeUser, resourceID, permission)
	if err != nil {
		h.logger.Error("failed to check ACL access", "error", err, "userId", resourceID, "actorId", claims.UserID)
		presenter.Error(w, http.StatusInternalServerError, "Internal server error", nil)
		return false
	}

	if !hasAccess {
		presenter.Error(w, http.StatusForbidden, "You do not have permission for this user", nil)
		return false
	}

	return true
}
