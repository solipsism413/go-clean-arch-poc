// Package middleware provides HTTP middleware for the REST API.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/logger"
)

// ContextKey is the type for context keys.
type ContextKey string

// Context keys.
const (
	UserIDKey      ContextKey = "userID"
	EmailKey       ContextKey = "email"
	RolesKey       ContextKey = "roles"
	RoleIDsKey     ContextKey = "roleIDs"
	PermissionsKey ContextKey = "permissions"
)

// ResponseWriter wraps http.ResponseWriter to capture status code.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

// NewResponseWriter creates a new ResponseWriter.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the status code.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written.
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Status returns the captured status code.
func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}

// BytesWritten returns the number of bytes written.
func (rw *ResponseWriter) BytesWritten() int {
	return rw.bytesWritten
}

// RequestIDMiddleware adds a request ID to each request.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := logger.SetRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerMiddleware logs HTTP requests.
func LoggerMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := logger.GetRequestID(r.Context())

			// Wrap response writer
			rw := NewResponseWriter(w)

			// Process request
			next.ServeHTTP(rw, r)

			// Log request
			duration := time.Since(start)
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.Status(),
				"duration", duration.String(),
				"bytes", rw.BytesWritten(),
				"requestId", requestID,
				"userAgent", r.UserAgent(),
				"remoteAddr", r.RemoteAddr,
			)
		})
	}
}

// Recover recovers from panics and returns 500 error.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS preflight requests.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "Link, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetUserContext adds user claims to the context.
func SetUserContext(ctx context.Context, claims *dto.TokenClaims) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, EmailKey, claims.Email)
	ctx = context.WithValue(ctx, RolesKey, claims.Roles)
	ctx = context.WithValue(ctx, RoleIDsKey, claims.RoleIDs)
	ctx = context.WithValue(ctx, PermissionsKey, claims.Permissions)
	ctx = logger.SetUserID(ctx, claims.UserID.String())
	return ctx
}

// GetUserID retrieves user ID from context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// GetUserRoles retrieves user roles from context.
func GetUserRoles(ctx context.Context) []string {
	roles, ok := ctx.Value(RolesKey).([]string)
	if !ok {
		return nil
	}
	return roles
}

// GetUserRoleIDs retrieves user role IDs from context.
func GetUserRoleIDs(ctx context.Context) []uuid.UUID {
	roleIDs, ok := ctx.Value(RoleIDsKey).([]uuid.UUID)
	if !ok {
		return nil
	}
	return roleIDs
}

// GetUserPermissions retrieves user permissions from context.
func GetUserPermissions(ctx context.Context) []string {
	permissions, ok := ctx.Value(PermissionsKey).([]string)
	if !ok {
		return nil
	}
	return permissions
}

// HasRole checks if the user has a specific role.
func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRoles(ctx)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission.
func HasPermission(ctx context.Context, resource, action string) bool {
	permissions := GetUserPermissions(ctx)
	requiredPerm := resource + ":" + action
	for _, perm := range permissions {
		if perm == requiredPerm || perm == resource+":*" || perm == "*:*" {
			return true
		}
	}
	return false
}
