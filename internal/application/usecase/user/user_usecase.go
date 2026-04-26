// Package user contains user-related use cases.
package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

const (
	cachePrefix = "app"

	// Cache TTLs.
	entityCacheTTL = 5 * time.Minute
	listCacheTTL   = 2 * time.Minute
)

// Ensure UserUseCase implements input.UserService.
var _ input.UserService = (*UserUseCase)(nil)

// UserUseCase implements user-related use cases.
type UserUseCase struct {
	userRepo       output.UserRepository
	roleRepo       output.RoleRepository
	cache          output.CacheRepository
	eventPublisher output.EventPublisher
	tm             output.TransactionManager
	validator      validation.Validator
	logger         *slog.Logger
}

// NewUserUseCase creates a new UserUseCase.
func NewUserUseCase(
	userRepo output.UserRepository,
	roleRepo output.RoleRepository,
	cache output.CacheRepository,
	eventPublisher output.EventPublisher,
	tm output.TransactionManager,
	validator validation.Validator,
	logger *slog.Logger,
) *UserUseCase {
	return &UserUseCase{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		cache:          cache,
		eventPublisher: eventPublisher,
		tm:             tm,
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

	// Invalidate list caches so the new user appears in lists
	uc.invalidateUserCaches(ctx, user.ID, "", user.Email)

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

	oldEmail := user.Email

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

	// Invalidate caches (both old and new email keys)
	uc.invalidateUserCaches(ctx, id, oldEmail, user.Email)

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
	// Fetch user to get email for cache invalidation
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerror.ErrUserNotFound
	}

	// Delete user
	if err := uc.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate caches
	uc.invalidateUserCaches(ctx, id, user.Email, "")

	// Publish event
	evt := event.NewUserDeleted(id)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user deleted event", "userId", id, "error", err)
	}

	uc.logger.Info("user deleted", "userId", id)

	return nil
}

// GetUser retrieves a user by ID with cache-aside.
func (uc *UserUseCase) GetUser(ctx context.Context, id uuid.UUID) (*dto.UserOutput, error) {
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).User(id.String())

	// Try cache first
	if uc.cache != nil {
		var cached dto.UserOutput
		if err := uc.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
			uc.logger.Debug("user cache hit", "userId", id)
			return &cached, nil
		}
	}

	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	output := dto.UserFromEntity(user)

	// Store in cache
	if uc.cache != nil {
		_ = uc.cache.SetJSON(ctx, cacheKey, output, entityCacheTTL)
	}

	return output, nil
}

// GetUserByEmail retrieves a user by email with cache-aside.
func (uc *UserUseCase) GetUserByEmail(ctx context.Context, email string) (*dto.UserOutput, error) {
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).UserByEmail(email)

	// Try cache first
	if uc.cache != nil {
		var cached dto.UserOutput
		if err := uc.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
			uc.logger.Debug("user email cache hit", "email", email)
			return &cached, nil
		}
	}

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	output := dto.UserFromEntity(user)

	// Store in cache
	if uc.cache != nil {
		_ = uc.cache.SetJSON(ctx, cacheKey, output, entityCacheTTL)
	}

	return output, nil
}

// ListUsers retrieves users with filtering and pagination with cache-aside.
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

	// Build cache key
	filterHash := uc.buildUserListCacheKey(outputFilter, outputPagination)
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).UserList(filterHash)

	// Try cache first
	if uc.cache != nil {
		var cached dto.UserListOutput
		if err := uc.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
			uc.logger.Debug("user list cache hit", "filterHash", filterHash)
			return &cached, nil
		}
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

	result := &dto.UserListOutput{
		Users:      userOutputs,
		Total:      paginatedResult.Total,
		Page:       paginatedResult.Page,
		PageSize:   paginatedResult.PageSize,
		TotalPages: paginatedResult.TotalPages,
	}

	// Store in cache
	if uc.cache != nil {
		_ = uc.cache.SetJSON(ctx, cacheKey, result, listCacheTTL)
	}

	return result, nil
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

	// Invalidate user caches and sessions
	uc.invalidateUserCaches(ctx, userID, user.Email, user.Email)
	uc.invalidateUserSessions(ctx, userID)

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

	// Invalidate user caches and sessions
	uc.invalidateUserCaches(ctx, userID, user.Email, user.Email)
	uc.invalidateUserSessions(ctx, userID)

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
	return uc.tm.RunInTransaction(ctx, func(uow output.UnitOfWork) error {
		uc.logger.Info("seeding system roles...")

		roleRepo := uow.RoleRepository()

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
		existingRoles, err := roleRepo.FindAll(ctx)
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
			if err := roleRepo.DeleteByNames(ctx, obsoleteRoles); err != nil {
				return err
			}
		}

		// 5. Upsert desired roles and sync permissions
		for _, r := range desiredRoles {
			// Check if role exists to get its ID or create new one
			role, err := roleRepo.FindByName(ctx, r.Name)
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
			if err := roleRepo.RemoveAllPermissions(ctx, role.ID); err != nil {
				return err
			}

			// Save/Upsert role and its permissions
			if err := roleRepo.Save(ctx, role); err != nil {
				return err
			}

			uc.logger.Info("synchronized system role", "role", r.Name)
		}

		return nil
	})
}

// invalidateUserCaches removes user entity caches and all list caches.
func (uc *UserUseCase) invalidateUserCaches(ctx context.Context, userID uuid.UUID, oldEmail, newEmail string) {
	if uc.cache == nil {
		return
	}
	// Invalidate by ID
	cacheKey := output.NewCacheKeyBuilder(cachePrefix).User(userID.String())
	_ = uc.cache.Delete(ctx, cacheKey)

	// Invalidate by old email if known
	if oldEmail != "" {
		emailKey := output.NewCacheKeyBuilder(cachePrefix).UserByEmail(oldEmail)
		_ = uc.cache.Delete(ctx, emailKey)
	}

	// Invalidate by new email if known
	if newEmail != "" && newEmail != oldEmail {
		emailKey := output.NewCacheKeyBuilder(cachePrefix).UserByEmail(newEmail)
		_ = uc.cache.Delete(ctx, emailKey)
	}

	// Invalidate all list caches
	listPattern := output.NewCacheKeyBuilder(cachePrefix).UserList("") + "*"
	_ = uc.cache.DeletePattern(ctx, listPattern)
}

// invalidateUserSessions revokes all active sessions for a user.
func (uc *UserUseCase) invalidateUserSessions(ctx context.Context, userID uuid.UUID) {
	if uc.cache == nil {
		return
	}
	sessionPattern := output.NewCacheKeyBuilder(cachePrefix).UserSessions(userID.String())
	_ = uc.cache.DeletePattern(ctx, sessionPattern)
	uc.logger.Info("invalidated all user sessions", "userId", userID)
}

// buildUserListCacheKey creates a deterministic hash from filter and pagination.
func (uc *UserUseCase) buildUserListCacheKey(filter output.UserFilter, pagination output.Pagination) string {
	h := sha256.New()
	fmt.Fprintf(h, "search=%s|", filter.Search)
	if filter.RoleID != nil {
		fmt.Fprintf(h, "role=%s|", filter.RoleID.String())
	}
	fmt.Fprintf(h, "page=%d|size=%d|sort=%s|desc=%v",
		pagination.Page, pagination.PageSize, pagination.SortBy, pagination.SortDesc)
	return hex.EncodeToString(h.Sum(nil))
}
