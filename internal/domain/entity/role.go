package entity

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role in the RBAC system.
// Roles group permissions together and can be assigned to users.
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleViewer  = "viewer"
)

type Role struct {
	// ID is the unique identifier for the role.
	ID uuid.UUID

	// Name is the unique name of the role (e.g., "admin", "manager", "member").
	Name string

	// Description provides details about the role's purpose.
	Description string

	// Permissions contains the permissions granted by this role.
	Permissions []Permission

	// CreatedAt is the timestamp when the role was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the role was last updated.
	UpdatedAt time.Time
}

// NewRole creates a new Role with the given parameters.
func NewRole(name, description string) (*Role, error) {
	if name == "" {
		return nil, ErrEmptyRoleName
	}

	now := time.Now().UTC()
	return &Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Permissions: make([]Permission, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// AddPermission adds a permission to the role.
func (r *Role) AddPermission(permission Permission) {
	for _, p := range r.Permissions {
		if p.ID == permission.ID {
			return // Permission already exists
		}
	}
	r.Permissions = append(r.Permissions, permission)
	r.UpdatedAt = time.Now().UTC()
}

// RemovePermission removes a permission from the role.
func (r *Role) RemovePermission(permissionID uuid.UUID) {
	for i, p := range r.Permissions {
		if p.ID == permissionID {
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			r.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

// HasPermission checks if the role has the specified permission.
func (r *Role) HasPermission(resource ResourceType, action PermissionAction) bool {
	for _, p := range r.Permissions {
		if p.Resource == resource && p.Action == action {
			return true
		}
		// Wildcard support
		if p.Resource == ResourceTypeAll || (p.Resource == resource && p.Action == PermissionActionAll) {
			return true
		}
	}
	return false
}

// UpdateName updates the role name.
func (r *Role) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyRoleName
	}
	r.Name = name
	r.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDescription updates the role description.
func (r *Role) UpdateDescription(description string) {
	r.Description = description
	r.UpdatedAt = time.Now().UTC()
}
