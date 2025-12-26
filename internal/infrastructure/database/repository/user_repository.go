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

// Ensure UserRepository implements the output.UserRepository interface.
var _ output.UserRepository = (*UserRepository)(nil)

// UserRepository implements the user repository using PostgreSQL.
type UserRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Save creates a new user.
func (r *UserRepository) Save(ctx context.Context, user *entity.User) error {
	_, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	})
	return err
}

// Update updates an existing user.
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	_, err := r.queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
	})
	return err
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteUser(ctx, id)
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := r.queries.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	user := sqlcUserToEntity(row)

	// Load roles
	roles, err := r.loadUserRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

// FindByEmail retrieves a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	user := sqlcUserToEntity(row)

	// Load roles
	roles, err := r.loadUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

// FindAll retrieves users with filtering and pagination.
func (r *UserRepository) FindAll(ctx context.Context, filter output.UserFilter, pagination output.Pagination) ([]*entity.User, *output.PaginatedResult, error) {
	offset := int32((pagination.Page - 1) * pagination.PageSize)
	limit := int32(pagination.PageSize)

	var rows []sqlc.User
	var err error

	if filter.Search != "" {
		searchParam := filter.Search
		rows, err = r.queries.SearchUsers(ctx, sqlc.SearchUsersParams{
			Column1: &searchParam,
			Limit:   limit,
			Offset:  offset,
		})
	} else {
		rows, err = r.queries.ListUsers(ctx, sqlc.ListUsersParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, nil, err
	}

	total, err := r.queries.CountUsers(ctx)
	if err != nil {
		return nil, nil, err
	}

	users := make([]*entity.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, sqlcUserToEntity(row))
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return users, &output.PaginatedResult{
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ExistsByID checks if a user exists.
func (r *UserRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.queries.UserExists(ctx, id)
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.queries.UserExistsByEmail(ctx, email)
}

// loadUserRoles loads roles for a user.
func (r *UserRepository) loadUserRoles(ctx context.Context, userID uuid.UUID) ([]entity.Role, error) {
	roleRows, err := r.queries.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(roleRows) == 0 {
		return make([]entity.Role, 0), nil
	}

	roleIDs := make([]uuid.UUID, 0, len(roleRows))
	roleMap := make(map[uuid.UUID]*entity.Role)
	roles := make([]entity.Role, 0, len(roleRows))

	for _, roleRow := range roleRows {
		roleIDs = append(roleIDs, roleRow.ID)
		role := &entity.Role{
			ID:          roleRow.ID,
			Name:        roleRow.Name,
			Permissions: make([]entity.Permission, 0),
			CreatedAt:   roleRow.CreatedAt,
			UpdatedAt:   roleRow.UpdatedAt,
		}
		if roleRow.Description != nil {
			role.Description = *roleRow.Description
		}
		roleMap[roleRow.ID] = role
	}

	// Load permissions for all roles at once
	permRows, err := r.queries.GetPermissionsByRoleIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	for _, permRow := range permRows {
		if role, ok := roleMap[permRow.RoleID]; ok {
			role.Permissions = append(role.Permissions, entity.Permission{
				ID:        permRow.ID,
				Name:      permRow.Name,
				Resource:  entity.ResourceType(permRow.Resource),
				Action:    entity.PermissionAction(permRow.Action),
				CreatedAt: permRow.CreatedAt,
			})
		}
	}

	for _, roleRow := range roleRows {
		if role, ok := roleMap[roleRow.ID]; ok {
			roles = append(roles, *role)
		}
	}

	return roles, nil
}

func sqlcUserToEntity(row sqlc.User) *entity.User {
	return &entity.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		Roles:        make([]entity.Role, 0),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
