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
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockValidator is a mock implementation of the Validator interface.
type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(data any) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockValidator) ValidateVar(field any, tag string) error {
	args := m.Called(field, tag)
	return args.Error(0)
}

// MockUserRepository is a mock implementation of the UserRepository interface.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Save(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindAll(ctx context.Context, filter output.UserFilter, pagination output.Pagination) ([]*entity.User, *output.PaginatedResult, error) {
	args := m.Called(ctx, filter, pagination)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*entity.User), args.Get(1).(*output.PaginatedResult), args.Error(2)
}

func (m *MockUserRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

// MockRoleRepository is a mock implementation of the RoleRepository interface.
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Save(ctx context.Context, role *entity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *entity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) FindByName(ctx context.Context, name string) (*entity.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) FindAll(ctx context.Context) ([]*entity.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockRoleRepository) DeleteByNames(ctx context.Context, names []string) error {
	args := m.Called(ctx, names)
	return args.Error(0)
}

func (m *MockRoleRepository) RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error {
	args := m.Called(ctx, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) DeleteByName(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockRoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

// MockCacheRepository is a mock implementation of the CacheRepository interface.
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockCacheRepository) Increment(ctx context.Context, key string, value int64) (int64, error) {
	args := m.Called(ctx, key, value)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *MockCacheRepository) SetMultiple(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	args := m.Called(ctx, values, expiration)
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of the EventPublisher interface.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, evt event.Event) error {
	args := m.Called(ctx, topic, evt)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishBatch(ctx context.Context, topic string, events []event.Event) error {
	args := m.Called(ctx, topic, events)
	return args.Error(0)
}

func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockTransactionManager is a mock implementation of the TransactionManager interface.
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) RunInTransaction(ctx context.Context, fn func(output.UnitOfWork) error) error {
	args := m.Called(ctx, mock.AnythingOfType("func(output.UnitOfWork) error"))
	// Execute the function with a mock unit of work
	if fn != nil {
		mockUow := &MockUnitOfWork{}
		return fn(mockUow)
	}
	return args.Error(0)
}

// MockUnitOfWork is a mock implementation of the UnitOfWork interface.
type MockUnitOfWork struct {
	mock.Mock
	UserRepositoryFn  func() output.UserRepository
	RoleRepositoryFn  func() output.RoleRepository
	TaskRepositoryFn  func() output.TaskRepository
	LabelRepositoryFn func() output.LabelRepository
	ACLRepositoryFn   func() output.ACLRepository
}

func (m *MockUnitOfWork) UserRepository() output.UserRepository {
	if m.UserRepositoryFn != nil {
		return m.UserRepositoryFn()
	}
	return &MockUserRepository{}
}

func (m *MockUnitOfWork) RoleRepository() output.RoleRepository {
	if m.RoleRepositoryFn != nil {
		return m.RoleRepositoryFn()
	}
	return &MockRoleRepository{}
}

func (m *MockUnitOfWork) TaskRepository() output.TaskRepository {
	if m.TaskRepositoryFn != nil {
		return m.TaskRepositoryFn()
	}
	return &MockTaskRepository{}
}

func (m *MockUnitOfWork) LabelRepository() output.LabelRepository {
	if m.LabelRepositoryFn != nil {
		return m.LabelRepositoryFn()
	}
	return &MockLabelRepository{}
}

func (m *MockUnitOfWork) ACLRepository() output.ACLRepository {
	if m.ACLRepositoryFn != nil {
		return m.ACLRepositoryFn()
	}
	return nil
}

func (m *MockUnitOfWork) PermissionRepository() output.PermissionRepository {
	return nil
}

func (m *MockUnitOfWork) Begin(ctx context.Context) (output.UnitOfWork, error) {
	return m, nil
}

func (m *MockUnitOfWork) Commit(ctx context.Context) error {
	return nil
}

func (m *MockUnitOfWork) Rollback(ctx context.Context) error {
	return nil
}

// MockTaskRepository is a mock for TaskRepository (needed for UnitOfWork)
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Save(ctx context.Context, task *entity.Task) error { return nil }
func (m *MockTaskRepository) Update(ctx context.Context, task *entity.Task) error {
	return nil
}
func (m *MockTaskRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	return nil, nil
}
func (m *MockTaskRepository) FindAll(ctx context.Context, filter output.TaskFilter, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	return nil, nil, nil
}
func (m *MockTaskRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}
func (m *MockTaskRepository) Search(ctx context.Context, query string, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	return nil, nil, nil
}
func (m *MockTaskRepository) FindOverdue(ctx context.Context, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	return nil, nil, nil
}

func (m *MockTaskRepository) CountByStatus(ctx context.Context, status valueobject.TaskStatus) (int64, error) {
	return 0, nil
}

func (m *MockTaskRepository) FindByAssignee(ctx context.Context, assigneeID uuid.UUID, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	return nil, nil, nil
}

func (m *MockTaskRepository) FindByCreator(ctx context.Context, creatorID uuid.UUID, pagination output.Pagination) ([]*entity.Task, *output.PaginatedResult, error) {
	return nil, nil, nil
}

// MockLabelRepository is a mock for LabelRepository (needed for UnitOfWork)
type MockLabelRepository struct {
	mock.Mock
}

func (m *MockLabelRepository) Save(ctx context.Context, label *entity.Label) error   { return nil }
func (m *MockLabelRepository) Update(ctx context.Context, label *entity.Label) error { return nil }
func (m *MockLabelRepository) Delete(ctx context.Context, id uuid.UUID) error        { return nil }
func (m *MockLabelRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Label, error) {
	return nil, nil
}
func (m *MockLabelRepository) FindByName(ctx context.Context, name string) (*entity.Label, error) {
	return nil, nil
}
func (m *MockLabelRepository) FindAll(ctx context.Context) ([]*entity.Label, error) { return nil, nil }
func (m *MockLabelRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}
func (m *MockLabelRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]*entity.Label, error) {
	return nil, nil
}

// setupTestAuthUseCase creates a new AuthUseCase with mock dependencies for testing.
func setupTestAuthUseCase(
	mockUserRepo *MockUserRepository,
	mockRoleRepo *MockRoleRepository,
	mockCache *MockCacheRepository,
	mockEventPublisher *MockEventPublisher,
	mockTM *MockTransactionManager,
	tokenService *jwt.TokenService,
	mockValidator *MockValidator,
) *auth.AuthUseCase {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors during tests
	}))
	return auth.NewAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator, logger)
}

// createTestTokenService creates a real TokenService for testing.
func createTestTokenService() *jwt.TokenService {
	cfg := config.JWTConfig{
		SecretKey:           "test-secret-key-for-testing-only",
		AccessTokenDuration: 1 * time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:              "test-issuer",
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

// TestAuthUseCase_Login tests the Login method.
func TestAuthUseCase_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		testUser, err := entity.NewUser("test@example.com", "password123", "Test User")
		assert.NoError(t, err)
		testUser.ID = userID

		mockUserRepo.On("FindByID", ctx, userID).Return(testUser, nil)
		mockEventPublisher.On("Publish", ctx, output.TopicUserEvents, mock.Anything).Return(nil)

		err = uc.Logout(ctx, userID)

		assert.NoError(t, err)

		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		err := uc.Logout(ctx, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		repoErr := errors.New("database error")
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, repoErr)

		err := uc.Logout(ctx, userID)

		assert.Error(t, err)
		assert.Equal(t, repoErr, err)

		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthUseCase_RefreshToken tests the RefreshToken method.
func TestAuthUseCase_RefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotNil(t, result.User)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		refreshToken := "invalid-refresh-token"

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrInvalidToken)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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

		// Mock the repo to return nil (user not found)
		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.RefreshToken(ctx, refreshToken)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthUseCase_ChangePassword tests the ChangePassword method.
func TestAuthUseCase_ChangePassword(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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

		err = uc.ChangePassword(ctx, userID, input)

		assert.NoError(t, err)

		mockValidator.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEventPublisher.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		// Create a test user and generate a valid token
		testUser, err := createTestUserWithRoles("test@example.com", "password123", "Test User", []string{"admin"})
		assert.NoError(t, err)

		roles := []string{"admin"}
		roleIDs := []uuid.UUID{testUser.Roles[0].ID}
		permissions := []string{"task:read"}
		authOutput, err := tokenService.GenerateTokenPair(ctx, testUser.ID, testUser.Email, roles, roleIDs, permissions)
		assert.NoError(t, err)

		result, err := uc.ValidateToken(ctx, authOutput.AccessToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testUser.ID, result.UserID)
		assert.Equal(t, testUser.Email, result.Email)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		token := "invalid-token"

		result, err := uc.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("malformed token", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		tokenService := createTestTokenService()
		mockValidator := new(MockValidator)

		uc := setupTestAuthUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, tokenService, mockValidator)

		token := "not-a-jwt-token"

		result, err := uc.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
