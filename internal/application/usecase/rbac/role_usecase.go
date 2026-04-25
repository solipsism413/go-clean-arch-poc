// Package rbac contains role and permission related use cases.
package rbac

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
)

// Ensure RoleUseCase implements input.RoleService.
var _ input.RoleService = (*RoleUseCase)(nil)

// RoleUseCase implements role-related use cases.
type RoleUseCase struct {
	roleRepo output.RoleRepository
	logger   *slog.Logger
}

// NewRoleUseCase creates a new RoleUseCase.
func NewRoleUseCase(roleRepo output.RoleRepository, logger *slog.Logger) *RoleUseCase {
	return &RoleUseCase{
		roleRepo: roleRepo,
		logger:   logger,
	}
}

// CreateRole creates a new role.
func (uc *RoleUseCase) CreateRole(ctx context.Context, input dto.CreateRoleInput) (*dto.RoleOutput, error) {
	return nil, domainerror.NewDomainError(domainerror.CodeInvalidOperation, "role creation not supported via graphql")
}

// UpdateRole updates an existing role.
func (uc *RoleUseCase) UpdateRole(ctx context.Context, id uuid.UUID, input dto.UpdateRoleInput) (*dto.RoleOutput, error) {
	return nil, domainerror.NewDomainError(domainerror.CodeInvalidOperation, "role update not supported via graphql")
}

// DeleteRole deletes a role by ID.
func (uc *RoleUseCase) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return domainerror.NewDomainError(domainerror.CodeInvalidOperation, "role deletion not supported via graphql")
}

// GetRole retrieves a role by ID.
func (uc *RoleUseCase) GetRole(ctx context.Context, id uuid.UUID) (*dto.RoleOutput, error) {
	role, err := uc.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domainerror.ErrRoleNotFound
	}

	return dto.RoleFromEntity(role), nil
}

// ListRoles retrieves all roles.
func (uc *RoleUseCase) ListRoles(ctx context.Context) ([]*dto.RoleOutput, error) {
	roles, err := uc.roleRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	outputs := make([]*dto.RoleOutput, 0, len(roles))
	for _, role := range roles {
		outputs = append(outputs, dto.RoleFromEntity(role))
	}

	return outputs, nil
}

// AddPermission adds a permission to a role.
func (uc *RoleUseCase) AddPermission(ctx context.Context, roleID, permissionID uuid.UUID) (*dto.RoleOutput, error) {
	return nil, domainerror.NewDomainError(domainerror.CodeInvalidOperation, "permission management not supported via graphql")
}

// RemovePermission removes a permission from a role.
func (uc *RoleUseCase) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) (*dto.RoleOutput, error) {
	return nil, domainerror.NewDomainError(domainerror.CodeInvalidOperation, "permission management not supported via graphql")
}
