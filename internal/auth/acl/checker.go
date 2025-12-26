// Package acl provides Access Control List functionality for fine-grained permissions.
package acl

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// SubjectType defines types of subjects that can be granted permissions.
type SubjectType = entity.ACLSubjectType

const (
	SubjectTypeUser = entity.ACLSubjectTypeUser
	SubjectTypeRole = entity.ACLSubjectTypeRole
)

// Permission defines ACL permission levels.
type Permission = entity.ACLPermission

const (
	PermissionRead   = entity.ACLPermissionRead
	PermissionWrite  = entity.ACLPermissionWrite
	PermissionDelete = entity.ACLPermissionDelete
	PermissionAdmin  = entity.ACLPermissionAdmin
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
func (c *Checker) CanAccess(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, resourceType entity.ResourceType, resourceID uuid.UUID, permission Permission) (bool, error) {
	if userID == uuid.Nil {
		return false, nil
	}

	// Check direct user permission
	hasPermission, err := c.aclRepo.HasPermission(
		ctx,
		resourceType,
		resourceID,
		SubjectTypeUser,
		userID,
		permission,
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
			resourceType,
			resourceID,
			SubjectTypeUser,
			userID,
			PermissionAdmin,
		)
		if err != nil {
			return false, err
		}
		if hasAdmin {
			return true, nil
		}
	}

	// Check role-based permissions
	for _, roleID := range roleIDs {
		hasRolePermission, err := c.aclRepo.HasPermission(
			ctx,
			resourceType,
			resourceID,
			SubjectTypeRole,
			roleID,
			permission,
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
				resourceType,
				resourceID,
				SubjectTypeRole,
				roleID,
				PermissionAdmin,
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
		subjectType,
		subjectID,
		permission,
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
	return c.aclRepo.DeleteByResource(ctx, resourceType, resourceID)
}

// GetResourcePermissions gets all ACL entries for a resource.
func (c *Checker) GetResourcePermissions(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID) ([]*entity.ACLEntry, error) {
	return c.aclRepo.FindByResource(ctx, resourceType, resourceID)
}

// GetUserPermissions gets all ACL entries for a user.
func (c *Checker) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*entity.ACLEntry, error) {
	return c.aclRepo.FindBySubject(ctx, SubjectTypeUser, userID)
}
