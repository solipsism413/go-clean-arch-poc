// Package rbac provides Role-Based Access Control functionality.
package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// Authorizer provides RBAC authorization checks.
type Authorizer struct {
	// Predefined permission mappings
	rolePermissions map[string][]string
}

// NewAuthorizer creates a new RBAC authorizer.
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		rolePermissions: defaultRolePermissions(),
	}
}

// defaultRolePermissions returns default role-permission mappings.
func defaultRolePermissions() map[string][]string {
	return map[string][]string{
		"admin": {
			"tasks:*",
			"users:*",
			"roles:*",
			"permissions:*",
			"labels:*",
		},
		"manager": {
			"tasks:create",
			"tasks:read",
			"tasks:update",
			"tasks:delete",
			"tasks:assign",
			"users:read",
			"labels:*",
		},
		"member": {
			"tasks:create",
			"tasks:read",
			"tasks:update",
			"labels:read",
		},
		"viewer": {
			"tasks:read",
			"users:read",
			"labels:read",
		},
	}
}

// HasPermission checks if a user has a specific permission.
func (a *Authorizer) HasPermission(ctx context.Context, user *entity.User, resource, action string) bool {
	if user == nil {
		return false
	}

	requiredPermission := resource + ":" + action

	// Check user's roles
	for _, role := range user.Roles {
		// Check direct permissions
		for _, perm := range role.Permissions {
			if a.matchPermission(string(perm.Resource)+":"+string(perm.Action), requiredPermission) {
				return true
			}
		}

		// Check predefined role permissions
		if perms, ok := a.rolePermissions[role.Name]; ok {
			for _, perm := range perms {
				if a.matchPermission(perm, requiredPermission) {
					return true
				}
			}
		}
	}

	return false
}

// HasRole checks if a user has a specific role.
func (a *Authorizer) HasRole(ctx context.Context, user *entity.User, roleName string) bool {
	if user == nil {
		return false
	}

	for _, role := range user.Roles {
		if role.Name == roleName {
			return true
		}
	}

	return false
}

// HasAnyRole checks if a user has any of the specified roles.
func (a *Authorizer) HasAnyRole(ctx context.Context, user *entity.User, roleNames ...string) bool {
	for _, roleName := range roleNames {
		if a.HasRole(ctx, user, roleName) {
			return true
		}
	}
	return false
}

// CanManageTask checks if a user can manage a specific task.
func (a *Authorizer) CanManageTask(ctx context.Context, user *entity.User, task *entity.Task) bool {
	if user == nil || task == nil {
		return false
	}

	// Admin can manage all tasks
	if a.HasRole(ctx, user, "admin") {
		return true
	}

	// Task creator can manage their task
	if task.CreatorID == user.ID {
		return true
	}

	// Task assignee can manage their assigned task
	if task.AssigneeID != nil && *task.AssigneeID == user.ID {
		return true
	}

	// Managers can manage all tasks
	if a.HasRole(ctx, user, "manager") {
		return true
	}

	return false
}

// CanViewTask checks if a user can view a specific task.
func (a *Authorizer) CanViewTask(ctx context.Context, user *entity.User, task *entity.Task) bool {
	if user == nil || task == nil {
		return false
	}

	// Anyone with tasks:read permission can view
	return a.HasPermission(ctx, user, "tasks", "read")
}

// CanManageUser checks if a user can manage another user.
func (a *Authorizer) CanManageUser(ctx context.Context, actor *entity.User, targetUserID uuid.UUID) bool {
	if actor == nil {
		return false
	}

	// Admin can manage all users
	if a.HasRole(ctx, actor, "admin") {
		return true
	}

	// Users can manage themselves
	if actor.ID == targetUserID {
		return true
	}

	return false
}

// matchPermission checks if a permission matches, supporting wildcards.
func (a *Authorizer) matchPermission(permission, required string) bool {
	// Exact match
	if permission == required {
		return true
	}

	// Wildcard match (e.g., "tasks:*" matches "tasks:read")
	if len(permission) > 0 && permission[len(permission)-1] == '*' {
		prefix := permission[:len(permission)-1]
		if len(required) >= len(prefix) && required[:len(prefix)] == prefix {
			return true
		}
	}

	// Full wildcard
	if permission == "*" {
		return true
	}

	return false
}

// Permission represents a permission in the format "resource:action".
type Permission struct {
	Resource string
	Action   string
}

// ParsePermission parses a permission string.
func ParsePermission(perm string) Permission {
	for i, c := range perm {
		if c == ':' {
			return Permission{
				Resource: perm[:i],
				Action:   perm[i+1:],
			}
		}
	}
	return Permission{Resource: perm, Action: "*"}
}
