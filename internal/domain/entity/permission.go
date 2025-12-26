package entity

import (
	"time"

	"github.com/google/uuid"
)

// Permission represents a permission in the RBAC/ACL system.
// Permissions define what actions can be performed on what resources.
type Permission struct {
	// ID is the unique identifier for the permission.
	ID uuid.UUID

	// Name is a human-readable name for the permission.
	Name string

	// Resource is the resource this permission applies to (e.g., "task", "user", "*").
	Resource string

	// Action is the action this permission allows (e.g., "create", "read", "update", "delete", "*").
	Action string

	// CreatedAt is the timestamp when the permission was created.
	CreatedAt time.Time
}

// NewPermission creates a new Permission with the given parameters.
func NewPermission(name, resource, action string) (*Permission, error) {
	if name == "" {
		return nil, ErrEmptyPermissionName
	}
	if resource == "" {
		return nil, ErrEmptyResource
	}
	if action == "" {
		return nil, ErrEmptyAction
	}

	return &Permission{
		ID:        uuid.New(),
		Name:      name,
		Resource:  resource,
		Action:    action,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Matches checks if this permission matches the given resource and action.
// Supports wildcard matching with "*".
func (p *Permission) Matches(resource, action string) bool {
	resourceMatches := p.Resource == "*" || p.Resource == resource
	actionMatches := p.Action == "*" || p.Action == action
	return resourceMatches && actionMatches
}

// String returns a string representation of the permission.
func (p *Permission) String() string {
	return p.Resource + ":" + p.Action
}
