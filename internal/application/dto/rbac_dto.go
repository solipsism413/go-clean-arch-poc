package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// CreateRoleInput represents the input for creating a role.
type CreateRoleInput struct {
	Name          string      `json:"name" validate:"required,min=1,max=50"`
	Description   string      `json:"description" validate:"max=255"`
	PermissionIDs []uuid.UUID `json:"permissionIds,omitempty"`
}

// UpdateRoleInput represents the input for updating a role.
type UpdateRoleInput struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=255"`
}

// RoleOutput represents the output for role operations.
type RoleOutput struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Permissions []PermissionOutput `json:"permissions,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// RoleFromEntity converts a Role entity to RoleOutput DTO.
func RoleFromEntity(role *entity.Role) *RoleOutput {
	if role == nil {
		return nil
	}
	output := &RoleOutput{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: make([]PermissionOutput, 0, len(role.Permissions)),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
	for _, perm := range role.Permissions {
		output.Permissions = append(output.Permissions, *PermissionFromEntity(&perm))
	}
	return output
}

// CreatePermissionInput represents the input for creating a permission.
type CreatePermissionInput struct {
	Name     string `json:"name" validate:"required,min=1,max=50"`
	Resource string `json:"resource" validate:"required,min=1,max=50"`
	Action   string `json:"action" validate:"required,min=1,max=50"`
}

// PermissionOutput represents the output for permission operations.
type PermissionOutput struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
}

// PermissionFromEntity converts a Permission entity to PermissionOutput DTO.
func PermissionFromEntity(permission *entity.Permission) *PermissionOutput {
	if permission == nil {
		return nil
	}
	return &PermissionOutput{
		ID:        permission.ID,
		Name:      permission.Name,
		Resource:  permission.Resource,
		Action:    permission.Action,
		CreatedAt: permission.CreatedAt,
	}
}

// CreateLabelInput represents the input for creating a label.
type CreateLabelInput struct {
	Name  string `json:"name" validate:"required,min=1,max=50"`
	Color string `json:"color" validate:"required,hexcolor"`
}

// UpdateLabelInput represents the input for updating a label.
type UpdateLabelInput struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=50"`
	Color *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
}

// LabelOutput represents the output for label operations.
type LabelOutput struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LabelFromEntity converts a Label entity to LabelOutput DTO.
func LabelFromEntity(label *entity.Label) *LabelOutput {
	if label == nil {
		return nil
	}
	return &LabelOutput{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		CreatedAt: label.CreatedAt,
		UpdatedAt: label.UpdatedAt,
	}
}

// CreateACLEntryInput represents the input for creating an ACL entry.
type CreateACLEntryInput struct {
	ResourceType string    `json:"resourceType" validate:"required,oneof=task project"`
	ResourceID   uuid.UUID `json:"resourceId" validate:"required"`
	SubjectType  string    `json:"subjectType" validate:"required,oneof=user role"`
	SubjectID    uuid.UUID `json:"subjectId" validate:"required"`
	Permission   string    `json:"permission" validate:"required,oneof=read write delete admin"`
}

// ACLEntryOutput represents the output for ACL operations.
type ACLEntryOutput struct {
	ID           uuid.UUID `json:"id"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uuid.UUID `json:"resourceId"`
	SubjectType  string    `json:"subjectType"`
	SubjectID    uuid.UUID `json:"subjectId"`
	Permission   string    `json:"permission"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ACLEntryFromEntity converts an ACLEntry entity to ACLEntryOutput DTO.
func ACLEntryFromEntity(entry *entity.ACLEntry) *ACLEntryOutput {
	if entry == nil {
		return nil
	}
	return &ACLEntryOutput{
		ID:           entry.ID,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		SubjectType:  entry.SubjectType,
		SubjectID:    entry.SubjectID,
		Permission:   entry.Permission,
		CreatedAt:    entry.CreatedAt,
	}
}
