// Package user contains user-related use cases.
package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// Ensure UserUseCase implements input.UserService.
var _ input.UserService = (*UserUseCase)(nil)

// UserUseCase implements user-related use cases.
type UserUseCase struct {
	userRepo       output.UserRepository
	roleRepo       output.RoleRepository
	cache          output.CacheRepository
	eventPublisher output.EventPublisher
	validator      validation.Validator
	logger         *slog.Logger
}

// NewUserUseCase creates a new UserUseCase.
func NewUserUseCase(
	userRepo output.UserRepository,
	roleRepo output.RoleRepository,
	cache output.CacheRepository,
	eventPublisher output.EventPublisher,
	validator validation.Validator,
	logger *slog.Logger,
) *UserUseCase {
	return &UserUseCase{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		cache:          cache,
		eventPublisher: eventPublisher,
		validator:      validator,
		logger:         logger,
	}
}

// CreateUser creates a new user.
func (uc *UserUseCase) CreateUser(ctx context.Context, input dto.CreateUserInput) (*dto.UserOutput, error) {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	// Check if email already exists
	exists, err := uc.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domainerror.ErrEmailAlreadyExists
	}

	// Create user entity
	user, err := entity.NewUser(input.Email, input.Password, input.Name)
	if err != nil {
		return nil, err
	}

	// Save user
	if err := uc.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewUserCreated(user.ID, user.Email, user.Name)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user created event", "userId", user.ID, "error", err)
	}

	uc.logger.Info("user created", "userId", user.ID, "email", user.Email)

	return dto.UserFromEntity(user), nil
}

// UpdateUser updates an existing user.
func (uc *UserUseCase) UpdateUser(ctx context.Context, id uuid.UUID, input dto.UpdateUserInput) (*dto.UserOutput, error) {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	// Get user
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	// Apply updates
	if input.Email != nil {
		// Check if new email already exists
		if *input.Email != user.Email {
			exists, err := uc.userRepo.ExistsByEmail(ctx, *input.Email)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, domainerror.ErrEmailAlreadyExists
			}
			user.Email = *input.Email
		}
	}

	if input.Name != nil {
		user.Name = *input.Name
	}

	user.UpdatedAt = time.Now().UTC()

	// Save user
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Publish event
	evt := event.NewUserUpdated(user.ID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user updated event", "userId", user.ID, "error", err)
	}

	uc.logger.Info("user updated", "userId", user.ID)

	return dto.UserFromEntity(user), nil
}

// DeleteUser deletes a user by ID.
func (uc *UserUseCase) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Check if user exists
	exists, err := uc.userRepo.ExistsByID(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domainerror.ErrUserNotFound
	}

	// Delete user
	if err := uc.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Publish event
	evt := event.NewUserDeleted(id)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user deleted event", "userId", id, "error", err)
	}

	uc.logger.Info("user deleted", "userId", id)

	return nil
}

// GetUser retrieves a user by ID.
func (uc *UserUseCase) GetUser(ctx context.Context, id uuid.UUID) (*dto.UserOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	return dto.UserFromEntity(user), nil
}

// GetUserByEmail retrieves a user by email.
func (uc *UserUseCase) GetUserByEmail(ctx context.Context, email string) (*dto.UserOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	return dto.UserFromEntity(user), nil
}

// ListUsers retrieves users with filtering and pagination.
func (uc *UserUseCase) ListUsers(ctx context.Context, filter dto.UserFilter, pagination dto.Pagination) (*dto.UserListOutput, error) {
	// Validate input
	if err := uc.validator.Validate(filter); err != nil {
		return nil, err
	}
	if err := uc.validator.Validate(pagination); err != nil {
		return nil, err
	}

	// Convert filter
	outputFilter := output.UserFilter{
		Search: filter.Search,
		RoleID: filter.RoleID,
	}

	// Convert pagination
	outputPagination := output.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
	}

	// Fetch users
	users, paginatedResult, err := uc.userRepo.FindAll(ctx, outputFilter, outputPagination)
	if err != nil {
		return nil, err
	}

	// Convert to output
	userOutputs := make([]*dto.UserOutput, 0, len(users))
	for _, user := range users {
		userOutputs = append(userOutputs, dto.UserFromEntity(user))
	}

	return &dto.UserListOutput{
		Users:      userOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}, nil
}

// AssignRole assigns a role to a user.
func (uc *UserUseCase) AssignRole(ctx context.Context, userID, roleID uuid.UUID) (*dto.UserOutput, error) {
	// Get user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	// Get role
	role, err := uc.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domainerror.ErrRoleNotFound
	}

	// Assign role
	user.AssignRole(*role)

	// Save user
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Publish event (using userID as assignedBy for now - should be the actor performing the action)
	evt := event.NewUserRoleAssigned(userID, roleID, role.Name, userID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish role assigned event", "userId", userID, "error", err)
	}

	uc.logger.Info("role assigned to user", "userId", userID, "roleId", roleID)

	return dto.UserFromEntity(user), nil
}

// RemoveRole removes a role from a user.
func (uc *UserUseCase) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) (*dto.UserOutput, error) {
	// Get user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	// Get role
	role, err := uc.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domainerror.ErrRoleNotFound
	}

	// Remove role
	user.RemoveRole(roleID)

	// Save user
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Publish event (using userID as removedBy for now - should be the actor performing the action)
	evt := event.NewUserRoleRemoved(userID, roleID, role.Name, userID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish role removed event", "userId", userID, "error", err)
	}

	uc.logger.Info("role removed from user", "userId", userID, "roleId", roleID)

	return dto.UserFromEntity(user), nil
}

// SeedSystemRoles seeds system-defined roles and ensures they are in sync with the database.
func (uc *UserUseCase) SeedSystemRoles(ctx context.Context) error {
	uc.logger.Info("seeding system roles...")

	// 1. Define desired roles state
	desiredRoles := []struct {
		Name        string
		Description string
		Permissions []struct {
			Resource entity.ResourceType
			Action   entity.PermissionAction
		}
	}{
		{
			Name:        entity.RoleAdmin,
			Description: "Full system access",
			Permissions: []struct {
				Resource entity.ResourceType
				Action   entity.PermissionAction
			}{
				{entity.ResourceTypeTask, entity.PermissionActionAll},
				{entity.ResourceTypeUser, entity.PermissionActionAll},
				{entity.ResourceTypeRole, entity.PermissionActionAll},
				{entity.ResourceTypePermissions, entity.PermissionActionAll},
				{entity.ResourceTypeLabels, entity.PermissionActionAll},
			},
		},
		{
			Name:        entity.RoleManager,
			Description: "Management access",
			Permissions: []struct {
				Resource entity.ResourceType
				Action   entity.PermissionAction
			}{
				{entity.ResourceTypeTask, entity.PermissionActionCreate},
				{entity.ResourceTypeTask, entity.PermissionActionRead},
				{entity.ResourceTypeTask, entity.PermissionActionUpdate},
				{entity.ResourceTypeTask, entity.PermissionActionDelete},
				{entity.ResourceTypeTask, entity.PermissionActionAssign},
				{entity.ResourceTypeUser, entity.PermissionActionRead},
				{entity.ResourceTypeLabel, entity.PermissionActionAll},
			},
		},
		{
			Name:        entity.RoleMember,
			Description: "Member access",
			Permissions: []struct {
				Resource entity.ResourceType
				Action   entity.PermissionAction
			}{
				{entity.ResourceTypeTask, entity.PermissionActionCreate},
				{entity.ResourceTypeTask, entity.PermissionActionRead},
				{entity.ResourceTypeTask, entity.PermissionActionUpdate},
				{entity.ResourceTypeLabel, entity.PermissionActionRead},
			},
		},
		{
			Name:        entity.RoleViewer,
			Description: "Viewer access",
			Permissions: []struct {
				Resource entity.ResourceType
				Action   entity.PermissionAction
			}{
				{entity.ResourceTypeTask, entity.PermissionActionRead},
				{entity.ResourceTypeUser, entity.PermissionActionRead},
				{entity.ResourceTypeLabel, entity.PermissionActionRead},
			},
		},
	}

	// 2. Map desired roles for easy lookup
	desiredRolesMap := make(map[string]bool)
	for _, r := range desiredRoles {
		desiredRolesMap[r.Name] = true
	}

	// 3. Get existing roles from database
	existingRoles, err := uc.roleRepo.FindAll(ctx)
	if err != nil {
		return err
	}

	// 4. Delete roles that are no longer defined in code
	obsoleteRoles := make([]string, 0)
	for _, er := range existingRoles {
		if !desiredRolesMap[er.Name] {
			obsoleteRoles = append(obsoleteRoles, er.Name)
		}
	}

	if len(obsoleteRoles) > 0 {
		uc.logger.Info("deleting obsolete system roles", "roles", obsoleteRoles)
		if err := uc.roleRepo.DeleteByNames(ctx, obsoleteRoles); err != nil {
			return err
		}
	}

	// 5. Upsert desired roles and sync permissions
	for _, r := range desiredRoles {
		// Check if role exists to get its ID or create new one
		role, err := uc.roleRepo.FindByName(ctx, r.Name)
		if err != nil {
			return err
		}

		if role == nil {
			// Create new role entity
			role, err = entity.NewRole(r.Name, r.Description)
			if err != nil {
				return err
			}
		} else {
			// Update existing role metadata
			role.UpdateDescription(r.Description)
			role.UpdatedAt = time.Now().UTC()
		}

		// Clear permissions in the entity for sync
		role.Permissions = make([]entity.Permission, 0)

		// Create and add permissions to the entity
		for _, rp := range r.Permissions {
			pName := string(rp.Resource) + ":" + string(rp.Action)
			perm, err := entity.NewPermission(pName, rp.Resource, rp.Action)
			if err != nil {
				return err
			}
			role.AddPermission(*perm)
		}

		// Clear existing permissions in DB for this role before saving new ones
		// This ensures that removed permissions are also synced
		if err := uc.roleRepo.RemoveAllPermissions(ctx, role.ID); err != nil {
			return err
		}

		// Save/Upsert role and its permissions
		if err := uc.roleRepo.Save(ctx, role); err != nil {
			return err
		}

		uc.logger.Info("synchronized system role", "role", r.Name)
	}

	return nil
}
