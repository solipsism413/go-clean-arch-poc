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
	rolePermissions map[string][]Permission
}

// NewAuthorizer creates a new RBAC authorizer.
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		rolePermissions: defaultRolePermissions(),
	}
}

// defaultRolePermissions returns default role-permission mappings.
func defaultRolePermissions() map[string][]Permission {
	return map[string][]Permission{
		entity.RoleAdmin: {
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionAll},
			{Resource: entity.ResourceTypeUsers, Action: entity.PermissionActionAll},
			{Resource: entity.ResourceTypeRoles, Action: entity.PermissionActionAll},
			{Resource: entity.ResourceTypePermissions, Action: entity.PermissionActionAll},
			{Resource: entity.ResourceTypeLabels, Action: entity.PermissionActionAll},
		},
		entity.RoleManager: {
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionCreate},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionRead},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionUpdate},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionDelete},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionAssign},
			{Resource: entity.ResourceTypeUsers, Action: entity.PermissionActionRead},
			{Resource: entity.ResourceTypeLabels, Action: entity.PermissionActionAll},
		},
		entity.RoleMember: {
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionCreate},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionRead},
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionUpdate},
			{Resource: entity.ResourceTypeLabels, Action: entity.PermissionActionRead},
		},
		entity.RoleViewer: {
			{Resource: entity.ResourceTypeTasks, Action: entity.PermissionActionRead},
			{Resource: entity.ResourceTypeUsers, Action: entity.PermissionActionRead},
			{Resource: entity.ResourceTypeLabels, Action: entity.PermissionActionRead},
		},
	}
}

// HasPermission checks if a user has a specific permission.
func (a *Authorizer) HasPermission(ctx context.Context, user *entity.User, resource entity.ResourceType, action entity.PermissionAction) bool {
	if user == nil {
		return false
	}

	// Check user's roles
	for _, role := range user.Roles {
		// Check direct permissions (from database)
		for _, perm := range role.Permissions {
			if a.matchPermission(perm.Resource, perm.Action, resource, action) {
				return true
			}
		}

		// Check predefined role permissions (from policy)
		if perms, ok := a.rolePermissions[role.Name]; ok {
			for _, perm := range perms {
				if a.matchPermission(perm.Resource, perm.Action, resource, action) {
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
	if a.HasRole(ctx, user, entity.RoleAdmin) {
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
	if a.HasRole(ctx, user, entity.RoleManager) {
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
	return a.HasPermission(ctx, user, entity.ResourceTypeTasks, entity.PermissionActionRead)
}

// CanManageUser checks if a user can manage another user.
func (a *Authorizer) CanManageUser(ctx context.Context, actor *entity.User, targetUserID uuid.UUID) bool {
	if actor == nil {
		return false
	}

	// Admin can manage all users
	if a.HasRole(ctx, actor, entity.RoleAdmin) {
		return true
	}

	// Users can manage themselves
	if actor.ID == targetUserID {
		return true
	}

	return false
}

// matchPermission checks if a permission matches, supporting wildcards.
func (a *Authorizer) matchPermission(
	pResource entity.ResourceType,
	pAction entity.PermissionAction,
	requiredResource entity.ResourceType,
	requiredAction entity.PermissionAction,
) bool {
	resourceMatches := pResource == entity.ResourceTypeAll || pResource == requiredResource
	actionMatches := pAction == entity.PermissionActionAll || pAction == requiredAction
	return resourceMatches && actionMatches
}

// Permission represents a permission in the format "resource:action".
type Permission struct {
	Resource entity.ResourceType
	Action   entity.PermissionAction
}
