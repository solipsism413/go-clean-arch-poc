package input

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
)

// RoleService defines the input port for role-related use cases.
type RoleService interface {
	// CreateRole creates a new role.
	CreateRole(ctx context.Context, input dto.CreateRoleInput) (*dto.RoleOutput, error)

	// UpdateRole updates an existing role.
	UpdateRole(ctx context.Context, id uuid.UUID, input dto.UpdateRoleInput) (*dto.RoleOutput, error)

	// DeleteRole deletes a role by ID.
	DeleteRole(ctx context.Context, id uuid.UUID) error

	// GetRole retrieves a role by ID.
	GetRole(ctx context.Context, id uuid.UUID) (*dto.RoleOutput, error)

	// ListRoles retrieves all roles.
	ListRoles(ctx context.Context) ([]*dto.RoleOutput, error)

	// AddPermission adds a permission to a role.
	AddPermission(ctx context.Context, roleID, permissionID uuid.UUID) (*dto.RoleOutput, error)

	// RemovePermission removes a permission from a role.
	RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) (*dto.RoleOutput, error)
}

// PermissionService defines the input port for permission-related use cases.
type PermissionService interface {
	// CreatePermission creates a new permission.
	CreatePermission(ctx context.Context, input dto.CreatePermissionInput) (*dto.PermissionOutput, error)

	// GetPermission retrieves a permission by ID.
	GetPermission(ctx context.Context, id uuid.UUID) (*dto.PermissionOutput, error)

	// ListPermissions retrieves all permissions.
	ListPermissions(ctx context.Context) ([]*dto.PermissionOutput, error)

	// DeletePermission deletes a permission by ID.
	DeletePermission(ctx context.Context, id uuid.UUID) error
}

// LabelService defines the input port for label-related use cases.
type LabelService interface {
	// CreateLabel creates a new label.
	CreateLabel(ctx context.Context, input dto.CreateLabelInput) (*dto.LabelOutput, error)

	// UpdateLabel updates an existing label.
	UpdateLabel(ctx context.Context, id uuid.UUID, input dto.UpdateLabelInput) (*dto.LabelOutput, error)

	// DeleteLabel deletes a label by ID.
	DeleteLabel(ctx context.Context, id uuid.UUID) error

	// GetLabel retrieves a label by ID.
	GetLabel(ctx context.Context, id uuid.UUID) (*dto.LabelOutput, error)

	// ListLabels retrieves all labels.
	ListLabels(ctx context.Context) ([]*dto.LabelOutput, error)
}
