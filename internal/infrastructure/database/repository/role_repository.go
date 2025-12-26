package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ensure RoleRepository implements the output.RoleRepository interface.
var _ output.RoleRepository = (*RoleRepository)(nil)

// RoleRepository implements the role repository using PostgreSQL.
type RoleRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Save creates a new role.
func (r *RoleRepository) Save(ctx context.Context, role *entity.Role) error {
	var desc *string
	if role.Description != "" {
		desc = &role.Description
	}

	_, err := r.queries.CreateRole(ctx, sqlc.CreateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: desc,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
	return err
}

// Update updates an existing role.
func (r *RoleRepository) Update(ctx context.Context, role *entity.Role) error {
	var desc *string
	if role.Description != "" {
		desc = &role.Description
	}

	_, err := r.queries.UpdateRole(ctx, sqlc.UpdateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: desc,
	})
	return err
}

// Delete removes a role by ID.
func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteRole(ctx, id)
}

// FindByID retrieves a role by ID.
func (r *RoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	row, err := r.queries.GetRole(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	role := sqlcRoleToEntity(row)

	// Load permissions
	permRows, err := r.queries.GetRolePermissions(ctx, id)
	if err != nil {
		return nil, err
	}

	for _, permRow := range permRows {
		role.Permissions = append(role.Permissions, entity.Permission{
			ID:        permRow.ID,
			Name:      permRow.Name,
			Resource:  permRow.Resource,
			Action:    permRow.Action,
			CreatedAt: permRow.CreatedAt,
		})
	}

	return role, nil
}

// FindByName retrieves a role by name.
func (r *RoleRepository) FindByName(ctx context.Context, name string) (*entity.Role, error) {
	row, err := r.queries.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return sqlcRoleToEntity(row), nil
}

// FindAll retrieves all roles.
func (r *RoleRepository) FindAll(ctx context.Context) ([]*entity.Role, error) {
	rows, err := r.queries.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	roles := make([]*entity.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, sqlcRoleToEntity(row))
	}

	return roles, nil
}

// FindByUserID retrieves roles for a specific user.
func (r *RoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	rows, err := r.queries.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles := make([]*entity.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, sqlcRoleToEntity(row))
	}

	return roles, nil
}

// ExistsByID checks if a role exists.
func (r *RoleRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.queries.RoleExists(ctx, id)
}

// ExistsByName checks if a role with the given name exists.
func (r *RoleRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	return r.queries.RoleExistsByName(ctx, name)
}

func sqlcRoleToEntity(row sqlc.Role) *entity.Role {
	role := &entity.Role{
		ID:          row.ID,
		Name:        row.Name,
		Permissions: make([]entity.Permission, 0),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Description != nil {
		role.Description = *row.Description
	}
	return role
}

// Ensure PermissionRepository implements the output.PermissionRepository interface.
var _ output.PermissionRepository = (*PermissionRepository)(nil)

// PermissionRepository implements the permission repository using PostgreSQL.
type PermissionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewPermissionRepository creates a new PermissionRepository.
func NewPermissionRepository(pool *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Save creates a new permission.
func (r *PermissionRepository) Save(ctx context.Context, permission *entity.Permission) error {
	_, err := r.queries.CreatePermission(ctx, sqlc.CreatePermissionParams{
		ID:        permission.ID,
		Name:      permission.Name,
		Resource:  permission.Resource,
		Action:    permission.Action,
		CreatedAt: permission.CreatedAt,
	})
	return err
}

// Delete removes a permission by ID.
func (r *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeletePermission(ctx, id)
}

// FindByID retrieves a permission by ID.
func (r *PermissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	row, err := r.queries.GetPermission(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entity.Permission{
		ID:        row.ID,
		Name:      row.Name,
		Resource:  row.Resource,
		Action:    row.Action,
		CreatedAt: row.CreatedAt,
	}, nil
}

// FindAll retrieves all permissions.
func (r *PermissionRepository) FindAll(ctx context.Context) ([]*entity.Permission, error) {
	rows, err := r.queries.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	permissions := make([]*entity.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, &entity.Permission{
			ID:        row.ID,
			Name:      row.Name,
			Resource:  row.Resource,
			Action:    row.Action,
			CreatedAt: row.CreatedAt,
		})
	}

	return permissions, nil
}

// FindByRoleID retrieves permissions for a specific role.
func (r *PermissionRepository) FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	rows, err := r.queries.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	permissions := make([]*entity.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, &entity.Permission{
			ID:        row.ID,
			Name:      row.Name,
			Resource:  row.Resource,
			Action:    row.Action,
			CreatedAt: row.CreatedAt,
		})
	}

	return permissions, nil
}

// ExistsByID checks if a permission exists.
func (r *PermissionRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.queries.PermissionExists(ctx, id)
}
