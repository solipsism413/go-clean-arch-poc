// Package rbac provides Role-Based Access Control functionality.
package rbac

import (
	"context"
	"strings"

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

	// Extract role names and permission strings from entity
	roleNames := make([]string, 0, len(user.Roles))
	permissions := make([]string, 0)
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
		for _, perm := range role.Permissions {
			permissions = append(permissions, string(perm.Resource)+":"+string(perm.Action))
		}
	}

	return a.HasPermissionFromClaims(roleNames, permissions, resource, action)
}

// HasPermissionFromClaims checks if a user has a specific permission based on claims.
func (a *Authorizer) HasPermissionFromClaims(roles []string, permissions []string, resource entity.ResourceType, action entity.PermissionAction) bool {
	// Check direct permissions (from database, now in claims)
	for _, p := range permissions {
		// Split permission into resource and action
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			continue
		}
		pResource := entity.ResourceType(parts[0])
		pAction := entity.PermissionAction(parts[1])

		if a.matchPermission(pResource, pAction, resource, action) {
			return true
		}
	}

	// Check predefined role permissions (from policy)
	for _, roleName := range roles {
		if perms, ok := a.rolePermissions[roleName]; ok {
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

	roleNames := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}

	return a.HasRoleFromClaims(roleNames, roleName)
}

// HasRoleFromClaims checks if a user has a specific role based on claims.
func (a *Authorizer) HasRoleFromClaims(roles []string, roleName string) bool {
	for _, r := range roles {
		if r == roleName {
			return true
		}
	}
	return false
}

// HasAnyRole checks if a user has any of the specified roles.
func (a *Authorizer) HasAnyRole(ctx context.Context, user *entity.User, roleNames ...string) bool {
	if user == nil {
		return false
	}

	userRoles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		userRoles = append(userRoles, role.Name)
	}

	return a.HasAnyRoleFromClaims(userRoles, roleNames...)
}

// HasAnyRoleFromClaims checks if a user has any of the specified roles based on claims.
func (a *Authorizer) HasAnyRoleFromClaims(userRoles []string, roleNames ...string) bool {
	for _, roleName := range roleNames {
		if a.HasRoleFromClaims(userRoles, roleName) {
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
