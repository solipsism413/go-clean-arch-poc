// Package rest provides the REST API transport layer using standard net/http.
package rest

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/handler"
	customMiddleware "github.com/handiism/go-clean-arch-poc/internal/transport/rest/middleware"

	_ "github.com/handiism/go-clean-arch-poc/docs" // Swagger docs
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Router holds the HTTP router and handlers.
type Router struct {
	mux              *http.ServeMux
	taskHandler      *handler.TaskHandler
	taskQueryHandler *handler.TaskQueryHandler
	userHandler      *handler.UserHandler
	authHandler      *handler.AuthHandler
	authService      input.AuthService
	authMiddleware   *auth.Middleware
	logger           *slog.Logger
}

// NewRouter creates a new REST API router using standard library.
func NewRouter(
	taskService input.TaskService,
	userService input.UserService,
	authService input.AuthService,
	authMiddleware *auth.Middleware,
	aclChecker *acl.Checker,
	logger *slog.Logger,
) *Router {
	mux := http.NewServeMux()

	r := &Router{
		mux:              mux,
		taskHandler:      handler.NewTaskHandler(taskService, aclChecker, logger),
		taskQueryHandler: handler.NewTaskQueryHandler(taskService, logger),
		userHandler:      handler.NewUserHandler(userService, aclChecker, logger),
		authHandler:      handler.NewAuthHandler(authService, logger),
		authService:      authService,
		authMiddleware:   authMiddleware,
		logger:           logger,
	}

	r.registerRoutes()

	return r
}

// registerRoutes registers all API routes.
func (r *Router) registerRoutes() {
	// Health check
	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger documentation
	r.mux.Handle("/swagger/", http.StripPrefix("/swagger", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	)))

	// Public auth routes
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.Login)
	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.Register)
	r.mux.HandleFunc("POST /api/v1/auth/refresh", r.authHandler.RefreshToken)

	// Protected auth routes
	r.mux.HandleFunc("POST /api/v1/auth/logout", r.withAuth(r.authHandler.Logout))
	r.mux.HandleFunc("POST /api/v1/auth/change-password", r.withAuth(r.authHandler.ChangePassword))

	// Task routes (protected)
	r.mux.HandleFunc("GET /api/v1/tasks/search", r.withAuth(r.taskQueryHandler.Search))
	r.mux.HandleFunc("GET /api/v1/tasks/overdue", r.withAuth(r.taskQueryHandler.Overdue))
	r.mux.HandleFunc("GET /api/v1/tasks", r.withAuth(r.taskHandler.List))
	r.mux.HandleFunc("POST /api/v1/tasks", r.withPermission(entity.ResourceTypeTasks, entity.PermissionActionCreate, r.taskHandler.Create))
	r.mux.HandleFunc("GET /api/v1/tasks/{id}", r.withAuth(r.taskHandler.Get))
	r.mux.HandleFunc("PUT /api/v1/tasks/{id}", r.withAuth(r.taskHandler.Update))
	r.mux.HandleFunc("DELETE /api/v1/tasks/{id}", r.withAuth(r.taskHandler.Delete))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/assign", r.withAuth(r.taskHandler.Assign))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/unassign", r.withAuth(r.taskHandler.Unassign))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/complete", r.withAuth(r.taskHandler.Complete))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/archive", r.withAuth(r.taskHandler.Archive))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/status", r.withAuth(r.taskHandler.ChangeStatus))
	r.mux.HandleFunc("POST /api/v1/tasks/{id}/labels/{labelId}", r.withAuth(r.taskHandler.AddLabel))
	r.mux.HandleFunc("DELETE /api/v1/tasks/{id}/labels/{labelId}", r.withAuth(r.taskHandler.RemoveLabel))

	// User routes (protected)
	r.mux.HandleFunc("GET /api/v1/users", r.withPermission(entity.ResourceTypeUsers, entity.PermissionActionRead, r.userHandler.List))
	r.mux.HandleFunc("GET /api/v1/users/me", r.withAuth(r.userHandler.Me))
	r.mux.HandleFunc("GET /api/v1/users/{id}", r.withAuth(r.userHandler.Get))
	r.mux.HandleFunc("PUT /api/v1/users/{id}", r.withAuth(r.userHandler.Update))
	r.mux.HandleFunc("DELETE /api/v1/users/{id}", r.withAuth(r.userHandler.Delete))
	r.mux.HandleFunc("POST /api/v1/users/{id}/roles/{roleId}", r.withPermission(entity.ResourceTypeUsers, entity.PermissionActionUpdate, r.userHandler.AssignRole))
	r.mux.HandleFunc("DELETE /api/v1/users/{id}/roles/{roleId}", r.withPermission(entity.ResourceTypeUsers, entity.PermissionActionUpdate, r.userHandler.RemoveRole))
}

// withAuth wraps a handler with authentication middleware.
func (r *Router) withAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Apply CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Get token from Authorization header
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"Missing authorization header"}}`, http.StatusUnauthorized)
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"Invalid authorization header format"}}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		claims, err := r.authService.ValidateToken(req.Context(), token)
		if err != nil {
			http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"Invalid or expired token"}}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := customMiddleware.SetUserContext(req.Context(), claims)

		// Add claims to context using auth package key for its middleware
		ctx = context.WithValue(ctx, auth.ClaimsContextKey, claims)

		h(w, req.WithContext(ctx))
	}
}

// withPermission wraps a handler with RBAC permission middleware.
func (r *Router) withPermission(resource entity.ResourceType, action entity.PermissionAction, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		permMiddleware := r.authMiddleware.RequirePermission(resource, action)
		handler := permMiddleware(http.HandlerFunc(h))
		r.withAuth(handler.ServeHTTP)(w, req)
	}
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Apply global middleware
	handler := r.applyMiddleware(r.mux)
	handler.ServeHTTP(w, req)
}

// applyMiddleware applies global middleware to the handler.
func (r *Router) applyMiddleware(h http.Handler) http.Handler {
	// Apply in reverse order (last applied is first executed)
	h = customMiddleware.Recover(h)
	h = customMiddleware.LoggerMiddleware(r.logger)(h)
	h = customMiddleware.RequestIDMiddleware(h)
	h = customMiddleware.CORSMiddleware(h)

	return h
}

// Handler returns the underlying http.Handler.
func (r *Router) Handler() http.Handler {
	return r
}

// Handle registers a handler for the given pattern.
func (r *Router) Handle(pattern string, h http.Handler) {
	r.mux.Handle(pattern, h)
}

// HandleFunc registers a handler function for the given pattern.
func (r *Router) HandleFunc(pattern string, h http.HandlerFunc) {
	r.mux.HandleFunc(pattern, h)
}
