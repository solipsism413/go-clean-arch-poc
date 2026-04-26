// Package auth_test contains tests for the auth use case.
package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/usecase/auth"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupTestAuthUseCase creates a new AuthUseCase with mock dependencies for testing.
func setupTestAuthUseCase(
	mockUserRepo *mocks.MockUserRepository,
	mockRoleRepo *mocks.MockRoleRepository,
	mockCache *mocks.MockCacheRepository,
	mockEventPublisher *mocks.MockEventPublisher,
	mockTM *mocks.MockTransactionManager,
	tokenService *jwt.TokenService,
	mockValidator *mocks.MockValidator,
) *auth.AuthUseCase {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors during tests
	}))
	return auth.NewAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator, logger)
}

// createTestTokenService creates a real TokenService for testing.
func createTestTokenService() *jwt.TokenService {
	cfg := config.JWTConfig{
		SecretKey:            "test-secret-key-for-testing-only",
		AccessTokenDuration:  1 * time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	return jwt.NewTokenService(cfg)
}

// createTestUserWithRoles creates a test user with roles for testing.
func createTestUserWithRoles(email, password, name string, roleNames []string) (*entity.User, error) {
	user, err := entity.NewUser(email, password, name)
	if err != nil {
		return nil, err
	}

	for _, roleName := range roleNames {
		role, err := entity.NewRole(roleName, roleName+" description")
		if err != nil {
			return nil, err
		}

		// Add some permissions to the role
		perm, err := entity.NewPermission("read:tasks", entity.ResourceTypeTask, entity.PermissionActionRead)
		if err != nil {
			return nil, err
		}
		role.AddPermission(*perm)

		user.AssignRole(*role)
	}
	return user, nil
}

// TestAuthUseCase_Register tests the Register method.
func TestAuthUseCase_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.CreateUserInput{
			Email:    "register@example.com",
			Password: "password123",
			Name:     "Register User",
		}

		memberRole, err := entity.NewRole(entity.RoleMember, "Member role")
		assert.NoError(t, err)
		perm, err := entity.NewPermission("tasks:read", entity.ResourceTypeTasks, entity.PermissionActionRead)
		assert.NoError(t, err)
		memberRole.AddPermission(*perm)

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockRoleRepo.On("FindByName", ctx, entity.RoleMember).Return(memberRole, nil)
		mockUserRepo.On("Save", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("SetJSON", ctx, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		result, err := uc.Register(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotNil(t, result.User)
		assert.Equal(t, input.Email, result.User.Email)
		assert.Len(t, result.User.Roles, 1)
		assert.Equal(t, entity.RoleMember, result.User.Roles[0].Name)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockRoleRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})

	t.Run("duplicate email", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.CreateUserInput{
			Email:    "register@example.com",
			Password: "password123",
			Name:     "Register User",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(true, nil)

		result, err := uc.Register(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrEmailAlreadyExists)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("default role not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.CreateUserInput{
			Email:    "register@example.com",
			Password: "password123",
			Name:     "Register User",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockRoleRepo.On("FindByName", ctx, entity.RoleMember).Return(nil, nil)

		result, err := uc.Register(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrRoleNotFound)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockRoleRepo.AssertExpectations(t)
	})
}

// TestAuthUseCase_Login tests the Login method.
func TestAuthUseCase_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.LoginInput{
			Email:    "test@example.com",
			Password: "password123",
		}

		testUser, err := createTestUserWithRoles("test@example.com", "password123", "Test User", []string{"admin"})
		assert.NoError(t, err)

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByEmail", ctx, input.Email).Return(testUser, nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("SetJSON", ctx, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		result, err := uc.Login(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotNil(t, result.User)
		assert.Equal(t, testUser.Email, result.User.Email)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.LoginInput{
			Email:    "invalid-email",
			Password: "",
		}

		validationErr := errors.New("validation failed")
		mockValidator.On("Validate", input).Return(validationErr)

		result, err := uc.Login(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, validationErr, err)

		mockValidator.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.LoginInput{
			Email:    "notfound@example.com",
			Password: "password123",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByEmail", ctx, input.Email).Return(nil, nil)

		result, err := uc.Login(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrInvalidCredentials)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.LoginInput{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		testUser, err := createTestUserWithRoles("test@example.com", "correctpassword", "Test User", []string{"admin"})
		assert.NoError(t, err)

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByEmail", ctx, input.Email).Return(testUser, nil)

		result, err := uc.Login(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrInvalidCredentials)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.LoginInput{
			Email:    "test@example.com",
			Password: "password123",
		}

		repoErr := errors.New("database error")
		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByEmail", ctx, input.Email).Return(nil, repoErr)

		result, err := uc.Login(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoErr, err)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthUseCase_Logout tests the Logout method.
func TestAuthUseCase_Logout(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		// Generate a valid token to pass to Logout
		roles := []string{"admin"}
		roleIDs := []uuid.UUID{uuid.New()}
		permissions := []string{"task:read"}
		authOutput, err := tokenService.GenerateTokenPair(ctx, userID, "test@example.com", roles, roleIDs, permissions)
		assert.NoError(t, err)

		mockCache.On("Set", mock.Anything, mock.Anything, []byte("revoked"), mock.Anything).Return(nil).Once()
		mockCache.On("GetJSON", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockCache.On("Delete", mock.Anything, mock.Anything).Return(nil).Once()
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)

		err = uc.Logout(ctx, userID, authOutput.AccessToken)

		assert.NoError(t, err)

		mockCache.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		err := uc.Logout(ctx, userID, "invalid-token")

		assert.Error(t, err)
	})
}

// TestAuthUseCase_RefreshToken tests the RefreshToken method.
func TestAuthUseCase_RefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		testUser, err := createTestUserWithRoles("test@example.com", "password123", "Test User", []string{"admin"})
		assert.NoError(t, err)
		userID := testUser.ID

		// First, generate a valid refresh token for the user
		roles := []string{"admin"}
		roleIDs := []uuid.UUID{testUser.Roles[0].ID}
		permissions := []string{"task:read"}
		authOutput, err := tokenService.GenerateTokenPair(ctx, userID, testUser.Email, roles, roleIDs, permissions)
		assert.NoError(t, err)
		refreshToken := authOutput.RefreshToken

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockCache.On("Exists", mock.Anything, mock.Anything).Return(false, nil).Once()
		mockCache.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Once()
		mockCache.On("SetJSON", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockCache.On("SetJSON", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotNil(t, result.User)

		mockUserRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		refreshToken := "invalid-refresh-token"

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrInvalidToken)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		// Create a user just to generate a valid token
		testUser, err := createTestUserWithRoles("test@example.com", "password123", "Test User", []string{"admin"})
		assert.NoError(t, err)
		userID := testUser.ID

		// Generate a valid refresh token for this user
		roles := []string{"admin"}
		roleIDs := []uuid.UUID{testUser.Roles[0].ID}
		permissions := []string{"task:read"}
		authOutput, err := tokenService.GenerateTokenPair(ctx, userID, testUser.Email, roles, roleIDs, permissions)
		assert.NoError(t, err)
		refreshToken := authOutput.RefreshToken

		mockCache.On("Exists", mock.Anything, mock.Anything).Return(false, nil).Once()
		mockCache.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Once()
		// Mock the repo to return nil (user not found)
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// TestAuthUseCase_ChangePassword tests the ChangePassword method.
func TestAuthUseCase_ChangePassword(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		testUser, err := entity.NewUser("test@example.com", "oldpassword", "Test User")
		assert.NoError(t, err)
		testUser.ID = userID

		input := dto.ChangePasswordInput{
			OldPassword: "oldpassword",
			NewPassword: "newpassword123",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)
		mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(nil)

		err = uc.ChangePassword(ctx, userID, input)

		assert.NoError(t, err)

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
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.ChangePasswordInput{
			OldPassword: "",
			NewPassword: "123",
		}

		validationErr := errors.New("validation failed")
		mockValidator.On("Validate", input).Return(validationErr)

		err := uc.ChangePassword(ctx, userID, input)

		assert.Error(t, err)
		assert.Equal(t, validationErr, err)

		mockValidator.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		input := dto.ChangePasswordInput{
			OldPassword: "oldpassword",
			NewPassword: "newpassword123",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		err := uc.ChangePassword(ctx, userID, input)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("wrong old password", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		testUser, err := entity.NewUser("test@example.com", "correctoldpassword", "Test User")
		assert.NoError(t, err)
		testUser.ID = userID

		input := dto.ChangePasswordInput{
			OldPassword: "wrongoldpassword",
			NewPassword: "newpassword123",
		}

		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)

		err = uc.ChangePassword(ctx, userID, input)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domainerror.ErrInvalidCredentials)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("repository error on update", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		testUser, err := entity.NewUser("test@example.com", "oldpassword", "Test User")
		assert.NoError(t, err)
		testUser.ID = userID

		input := dto.ChangePasswordInput{
			OldPassword: "oldpassword",
			NewPassword: "newpassword123",
		}

		repoErr := errors.New("database error")
		mockValidator.On("Validate", input).Return(nil)
		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(repoErr)

		err = uc.ChangePassword(ctx, userID, input)

		assert.Error(t, err)
		assert.Equal(t, repoErr, err)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthUseCase_ValidateToken tests the ValidateToken method.
func TestAuthUseCase_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		// Create a test user and generate a valid token
		testUser, err := createTestUserWithRoles("test@example.com", "password123", "Test User", []string{"admin"})
		assert.NoError(t, err)

		roles := []string{"admin"}
		roleIDs := []uuid.UUID{testUser.Roles[0].ID}
		permissions := []string{"task:read"}
		authOutput, err := tokenService.GenerateTokenPair(ctx, testUser.ID, testUser.Email, roles, roleIDs, permissions)
		assert.NoError(t, err)

		mockCache.On("Exists", mock.Anything, mock.Anything).Return(false, nil).Once()
		mockCache.On("Exists", mock.Anything, mock.Anything).Return(true, nil).Once()

		result, err := uc.ValidateToken(ctx, authOutput.AccessToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testUser.ID, result.UserID)
		assert.Equal(t, testUser.Email, result.Email)

		mockCache.AssertExpectations(t)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		token := "invalid-token"

		result, err := uc.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("malformed token", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockCache := new(mocks.MockCacheRepository)
		mockEventPublisher := new(mocks.MockEventPublisher)
		mockTM := new(mocks.MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(mocks.MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		token := "not-a-jwt-token"

		result, err := uc.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
