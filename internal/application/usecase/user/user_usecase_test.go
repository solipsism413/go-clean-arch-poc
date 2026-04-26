// Package user_test contains tests for the user use case.
package user_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupTestUserUseCase creates a new UserUseCase with mock dependencies for testing.
func setupTestUserUseCase(
	mockUserRepo *mocks.MockUserRepository,
	mockRoleRepo *mocks.MockRoleRepository,
	mockCache *mocks.MockCacheRepository,
	mockEventPublisher *mocks.MockEventPublisher,
	mockTM *mocks.MockTransactionManager,
	mockValidator *mocks.MockValidator,
) *user.UserUseCase {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors during tests
	}))
	return user.NewUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator, logger)
}

// TestUserUseCase_CreateUser tests the CreateUser method.
func TestUserUseCase_CreateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		input := dto.CreateUserInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockUserRepo.On("Save", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := uc.CreateUser(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Email, result.Email)
		assert.Equal(t, input.Name, result.Name)
		assert.NotEqual(t, uuid.Nil, result.ID)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		input := dto.CreateUserInput{
			Email:    "invalid-email",
			Password: "123",
			Name:     "",
		}

		validationErr := errors.New("validation failed")
		mockValidator.On("Validate", input).Return(validationErr)

		result, err := uc.CreateUser(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, validationErr, err)

		mockValidator.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		input := dto.CreateUserInput{
			Email:    "existing@example.com",
			Password: "password123",
			Name:     "Test User",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(true, nil)

		result, err := uc.CreateUser(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrEmailAlreadyExists)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("repository error on exists", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		input := dto.CreateUserInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		}

		repoErr := errors.New("database error")
		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(false, repoErr)

		result, err := uc.CreateUser(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoErr, err)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_UpdateUser tests the UpdateUser method.
func TestUserUseCase_UpdateUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		existingUser, _ := entity.NewUser("old@example.com", "password", "Old Name")
		existingUser.ID = userID

		newEmail := "new@example.com"
		newName := "New Name"

		input := dto.UpdateUserInput{
			Email: &newEmail,
			Name:  &newName,
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
		mockUserRepo.On("ExistsByEmail", ctx, newEmail).Return(false, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := uc.UpdateUser(ctx, userID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, newEmail, result.Email)
		assert.Equal(t, newName, result.Name)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		newName := "New Name"
		input := dto.UpdateUserInput{
			Name: &newName,
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.UpdateUser(ctx, userID, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("new email already exists", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		existingUser, _ := entity.NewUser("old@example.com", "password", "Old Name")
		existingUser.ID = userID

		newEmail := "existing@example.com"
		input := dto.UpdateUserInput{
			Email: &newEmail,
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
		mockUserRepo.On("ExistsByEmail", ctx, newEmail).Return(true, nil)

		result, err := uc.UpdateUser(ctx, userID, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrEmailAlreadyExists)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_DeleteUser tests the DeleteUser method.
func TestUserUseCase_DeleteUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser("delete@example.com", "password", "Delete User")
		testUser.ID = userID

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockUserRepo.On("Delete", ctx, userID).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		err := uc.DeleteUser(ctx, userID)

		assert.NoError(t, err)

		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		err := uc.DeleteUser(ctx, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_GetUser tests the GetUser method.
func TestUserUseCase_GetUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser("test@example.com", "password", "Test User")
		testUser.ID = userID

		mockCache.On("GetJSON", ctx, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("cache miss"))
		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockCache.On("SetJSON", ctx, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		result, err := uc.GetUser(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, userID, result.ID)
		assert.Equal(t, "test@example.com", result.Email)
		assert.Equal(t, "Test User", result.Name)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		mockCache.On("GetJSON", ctx, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("cache miss"))
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_GetUserByEmail tests the GetUserByEmail method.
func TestUserUseCase_GetUserByEmail(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser(email, "password", "Test User")

		mockCache.On("GetJSON", ctx, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("cache miss"))
		mockUserRepo.On("FindByEmail", ctx, email).Return(testUser, nil)
		mockCache.On("SetJSON", ctx, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		result, err := uc.GetUserByEmail(ctx, email)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, email, result.Email)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		mockCache.On("GetJSON", ctx, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("cache miss"))
		mockUserRepo.On("FindByEmail", ctx, email).Return(nil, nil)

		result, err := uc.GetUserByEmail(ctx, email)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_ListUsers tests the ListUsers method.
func TestUserUseCase_ListUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		filter := dto.UserFilter{Search: "test"}
		pagination := dto.Pagination{Page: 1, PageSize: 10}

		user1, _ := entity.NewUser("user1@example.com", "password", "User 1")
		user2, _ := entity.NewUser("user2@example.com", "password", "User 2")
		users := []*entity.User{user1, user2}

		paginatedResult := &output.PaginatedResult{
			Total:      2,
			Page:       1,
			PageSize:   10,
			TotalPages: 1,
		}

		mockValidator.On("Validate", filter).Return(nil)
		mockValidator.On("Validate", pagination).Return(nil)
		mockCache.On("GetJSON", ctx, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("cache miss"))
		mockUserRepo.On("FindAll", ctx, mock.AnythingOfType("output.UserFilter"), mock.AnythingOfType("output.Pagination")).
			Return(users, paginatedResult, nil)
		mockCache.On("SetJSON", ctx, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		result, err := uc.ListUsers(ctx, filter, pagination)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Users, 2)
		assert.Equal(t, int64(2), result.Total)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 0, PageSize: 0}

		validationErr := errors.New("invalid pagination")
		mockValidator.On("Validate", filter).Return(nil)
		mockValidator.On("Validate", pagination).Return(validationErr)

		result, err := uc.ListUsers(ctx, filter, pagination)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, validationErr, err)

		mockValidator.AssertExpectations(t)
	})
}

// TestUserUseCase_AssignRole tests the AssignRole method.
func TestUserUseCase_AssignRole(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	roleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser("test@example.com", "password", "Test User")
		testUser.ID = userID

		testRole, _ := entity.NewRole("admin", "Administrator")

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockRoleRepo.On("FindByID", ctx, roleID).Return(testRole, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := uc.AssignRole(ctx, userID, roleID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Roles, 1)
		assert.Equal(t, "admin", result.Roles[0].Name)

		mockUserRepo.AssertExpectations(t)
		mockRoleRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.AssignRole(ctx, userID, roleID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("role not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser("test@example.com", "password", "Test User")
		testUser.ID = userID

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockRoleRepo.On("FindByID", ctx, roleID).Return(nil, nil)

		result, err := uc.AssignRole(ctx, userID, roleID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrRoleNotFound)

		mockUserRepo.AssertExpectations(t)
		mockRoleRepo.AssertExpectations(t)
	})
}

// TestUserUseCase_RemoveRole tests the RemoveRole method.
func TestUserUseCase_RemoveRole(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	roleID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		testUser, _ := entity.NewUser("test@example.com", "password", "Test User")
		testUser.ID = userID
		testRole, _ := entity.NewRole("admin", "Administrator")
		testRole.ID = roleID
		testUser.AssignRole(*testRole)

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockRoleRepo.On("FindByID", ctx, roleID).Return(testRole, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := uc.RemoveRole(ctx, userID, roleID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Roles, 0)

		mockUserRepo.AssertExpectations(t)
		mockRoleRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// TestUserUseCase_SeedSystemRoles tests the SeedSystemRoles method.
func TestUserUseCase_SeedSystemRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("success - new roles", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		mockValidator := new(mocks.MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		// No existing roles
		mockRoleRepo.On("FindAll", ctx).Return([]*entity.Role{}, nil)
		// Role not found for each role name
		mockRoleRepo.On("FindByName", ctx, mock.AnythingOfType("string")).Return(nil, nil)
		mockRoleRepo.On("RemoveAllPermissions", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
		mockRoleRepo.On("Save", ctx, mock.AnythingOfType("*entity.Role")).Return(nil)
		mockTM.On("RunInTransaction", ctx, mock.Anything).Return(func(ctx context.Context, fn func(uow output.UnitOfWork) error) error {
			mockUow := new(mocks.MockUnitOfWork)
			mockUow.On("RoleRepository").Return(mockRoleRepo)
			return fn(mockUow)
		})

		err := uc.SeedSystemRoles(ctx)

		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
		mockTM.AssertExpectations(t)
	})
}
