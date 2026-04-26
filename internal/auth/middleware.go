// Package auth provides authentication and authorization middleware.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/auth/rbac"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// ContextKey is a type for context keys.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "user"
	// ClaimsContextKey is the context key for JWT claims.
	ClaimsContextKey ContextKey = "claims"
	// TokenContextKey is the context key for the raw JWT token string.
	TokenContextKey ContextKey = "token"
)

// Middleware provides authentication and authorization middleware.
type Middleware struct {
	authService input.AuthService
	userService input.UserService
	authorizer  *rbac.Authorizer
	aclChecker  *acl.Checker
}

// NewMiddleware creates a new auth middleware.
func NewMiddleware(authService input.AuthService, userService input.UserService, authorizer *rbac.Authorizer, aclChecker *acl.Checker) *Middleware {
	return &Middleware{
		authService: authService,
		userService: userService,
		authorizer:  authorizer,
		aclChecker:  aclChecker,
	}
}

// Authenticate validates the JWT token and sets user context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			presenter.Error(w, http.StatusUnauthorized, "Missing authorization header", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			presenter.Error(w, http.StatusUnauthorized, "Invalid authorization header format", nil)
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			presenter.Error(w, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		// Add claims and raw token to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		ctx = context.WithValue(ctx, TokenContextKey, token)

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
		ctx = context.WithValue(ctx, TokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission creates middleware that requires a specific permission.
func (m *Middleware) RequirePermission(resource entity.ResourceType, action entity.PermissionAction) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
				return
			}

			if !m.authorizer.HasPermissionFromClaims(claims.Roles, claims.Permissions, resource, action) {
				presenter.Error(w, http.StatusForbidden, "Insufficient permissions", nil)
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
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
				return
			}

			if !m.authorizer.HasRoleFromClaims(claims.Roles, roleName) {
				presenter.Error(w, http.StatusForbidden, "Insufficient role", nil)
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
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
				return
			}

			if !m.authorizer.HasAnyRoleFromClaims(claims.Roles, roleNames...) {
				presenter.Error(w, http.StatusForbidden, "Insufficient role", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireResourcePermission creates middleware that requires a specific ACL permission on a resource.
func (m *Middleware) RequireResourcePermission(resourceType entity.ResourceType, permission acl.Permission, idParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				presenter.Error(w, http.StatusUnauthorized, "Authentication required", nil)
				return
			}

			// Get resource ID from path parameter
			idStr := r.PathValue(idParam)
			if idStr == "" {
				presenter.Error(w, http.StatusBadRequest, "Resource ID missing", nil)
				return
			}

			resourceID, err := uuid.Parse(idStr)
			if err != nil {
				presenter.Error(w, http.StatusBadRequest, "Invalid resource ID", err)
				return
			}

			hasAccess, err := m.aclChecker.CanAccess(r.Context(), claims.UserID, claims.RoleIDs, resourceType, resourceID, permission)
			if err != nil {
				presenter.Error(w, http.StatusInternalServerError, "Error checking access", err)
				return
			}

			if !hasAccess {
				presenter.Error(w, http.StatusForbidden, "Insufficient permissions for this resource", nil)
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

// GetTokenFromContext retrieves the raw JWT token string from context.
func GetTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(TokenContextKey).(string)
	if !ok {
		return ""
	}
	return token
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
