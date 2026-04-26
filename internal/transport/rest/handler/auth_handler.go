package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/middleware"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService input.AuthService
	logger      *slog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService input.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Login handles POST /auth/login
// @Summary Login
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body dto.LoginInput true "Login credentials"
// @Success 200 {object} presenter.Response{data=dto.AuthOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input dto.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	auth, err := h.authService.Login(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, auth)
}

// Register handles POST /auth/register
// @Summary Register user
// @Description Create a new user account and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.CreateUserInput true "User details"
// @Success 201 {object} presenter.Response{data=dto.AuthOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 409 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	auth, err := h.authService.Register(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.Created(w, auth)
}

// RefreshToken handles POST /auth/refresh
// @Summary Refresh token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body object{refreshToken=string} true "Refresh token"
// @Success 200 {object} presenter.Response{data=dto.AuthOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	auth, err := h.authService.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.JSON(w, http.StatusOK, auth)
}

// Logout handles POST /auth/logout
// @Summary Logout
// @Description Invalidate the current session
// @Tags auth
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		presenter.Error(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	token := auth.GetTokenFromContext(r.Context())
	if err := h.authService.Logout(r.Context(), userID, token); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword handles POST /auth/change-password
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param passwords body dto.ChangePasswordInput true "Password change details"
// @Success 204 "No Content"
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		presenter.Error(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var input dto.ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.authService.ChangePassword(r.Context(), userID, input); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
