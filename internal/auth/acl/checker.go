// Package acl provides Access Control List functionality for fine-grained permissions.
package acl

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// SubjectType defines types of subjects that can be granted permissions.
type SubjectType string

const (
	SubjectTypeUser SubjectType = "user"
	SubjectTypeRole SubjectType = "role"
)

// Permission defines ACL permission levels.
type Permission string

const (
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"
)

// Checker provides ACL authorization checks.
type Checker struct {
	aclRepo output.ACLRepository
}

// NewChecker creates a new ACL checker.
func NewChecker(aclRepo output.ACLRepository) *Checker {
	return &Checker{
		aclRepo: aclRepo,
	}
}

// CanAccess checks if a user has the specified permission on a resource.
func (c *Checker) CanAccess(ctx context.Context, user *entity.User, resourceType entity.ResourceType, resourceID uuid.UUID, permission Permission) (bool, error) {
	if user == nil {
		return false, nil
	}

	// Check direct user permission
	hasPermission, err := c.aclRepo.HasPermission(
		ctx,
		string(resourceType),
		resourceID,
		string(SubjectTypeUser),
		user.ID,
		string(permission),
	)
	if err != nil {
		return false, err
	}
	if hasPermission {
		return true, nil
	}

	// Check if user has admin permission (implies all other permissions)
	if permission != PermissionAdmin {
		hasAdmin, err := c.aclRepo.HasPermission(
			ctx,
			string(resourceType),
			resourceID,
			string(SubjectTypeUser),
			user.ID,
			string(PermissionAdmin),
		)
		if err != nil {
			return false, err
		}
		if hasAdmin {
			return true, nil
		}
	}

	// Check role-based permissions
	for _, role := range user.Roles {
		hasRolePermission, err := c.aclRepo.HasPermission(
			ctx,
			string(resourceType),
			resourceID,
			string(SubjectTypeRole),
			role.ID,
			string(permission),
		)
		if err != nil {
			return false, err
		}
		if hasRolePermission {
			return true, nil
		}

		// Check admin permission for role
		if permission != PermissionAdmin {
			hasRoleAdmin, err := c.aclRepo.HasPermission(
				ctx,
				string(resourceType),
				resourceID,
				string(SubjectTypeRole),
				role.ID,
				string(PermissionAdmin),
			)
			if err != nil {
				return false, err
			}
			if hasRoleAdmin {
				return true, nil
			}
		}
	}

	return false, nil
}

// GrantAccess grants a permission to a user on a resource.
func (c *Checker) GrantAccess(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID, subjectType SubjectType, subjectID uuid.UUID, permission Permission) error {
	entry, err := entity.NewACLEntry(
		resourceType,
		resourceID,
		string(subjectType),
		subjectID,
		entity.ACLPermission(permission),
	)
	if err != nil {
		return err
	}
	return c.aclRepo.Save(ctx, entry)
}

// RevokeAccess revokes a permission from a resource.
func (c *Checker) RevokeAccess(ctx context.Context, entryID uuid.UUID) error {
	return c.aclRepo.Delete(ctx, entryID)
}

// RevokeAllAccess revokes all permissions for a resource.
func (c *Checker) RevokeAllAccess(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID) error {
	return c.aclRepo.DeleteByResource(ctx, string(resourceType), resourceID)
}

// GetResourcePermissions gets all ACL entries for a resource.
func (c *Checker) GetResourcePermissions(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID) ([]*entity.ACLEntry, error) {
	return c.aclRepo.FindByResource(ctx, string(resourceType), resourceID)
}

// GetUserPermissions gets all ACL entries for a user.
func (c *Checker) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*entity.ACLEntry, error) {
	return c.aclRepo.FindBySubject(ctx, string(SubjectTypeUser), userID)
}
