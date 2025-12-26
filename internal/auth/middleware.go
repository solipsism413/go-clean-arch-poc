// Package auth provides authentication and authorization middleware.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth/rbac"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// contextKey is a type for context keys.
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey contextKey = "user"
	// ClaimsContextKey is the context key for JWT claims.
	ClaimsContextKey contextKey = "claims"
)

// Middleware provides authentication and authorization middleware.
type Middleware struct {
	authService input.AuthService
	userService input.UserService
	authorizer  *rbac.Authorizer
}

// NewMiddleware creates a new auth middleware.
func NewMiddleware(authService input.AuthService, userService input.UserService, authorizer *rbac.Authorizer) *Middleware {
	return &Middleware{
		authService: authService,
		userService: userService,
		authorizer:  authorizer,
	}
}

// Authenticate validates the JWT token and sets user context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateOptional validates the JWT token if present, but doesn't fail if missing.
func (m *Middleware) AuthenticateOptional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]

		claims, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission creates middleware that requires a specific permission.
func (m *Middleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			if !m.authorizer.HasPermission(r.Context(), user, resource, action) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that requires a specific role.
func (m *Middleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			if !m.authorizer.HasRole(r.Context(), user, roleName) {
				http.Error(w, `{"error":"insufficient role"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles.
func (m *Middleware) RequireAnyRole(roleNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			if !m.authorizer.HasAnyRole(r.Context(), user, roleNames...) {
				http.Error(w, `{"error":"insufficient role"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaimsFromContext retrieves JWT claims from context.
func GetClaimsFromContext(ctx context.Context) *dto.TokenClaims {
	claims, ok := ctx.Value(ClaimsContextKey).(*dto.TokenClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserFromContext retrieves the authenticated user from context.
func GetUserFromContext(ctx context.Context) *entity.User {
	user, ok := ctx.Value(UserContextKey).(*entity.User)
	if !ok {
		return nil
	}
	return user
}

// SetUserInContext sets the user in context.
func SetUserInContext(ctx context.Context, user *entity.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}
