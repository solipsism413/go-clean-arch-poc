// Package user_test contains tests for the user use case.
package user_test

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
	"github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
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

func (m *MockCacheRepository) GetJSON(ctx context.Context, key string, dest any) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheRepository) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
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
	// Uow allows setting a custom UnitOfWork to use within transactions
	Uow *MockUnitOfWork
}

func (m *MockTransactionManager) RunInTransaction(ctx context.Context, fn func(output.UnitOfWork) error) error {
	args := m.Called(ctx, mock.Anything)
	// Execute the function with a mock unit of work
	if fn != nil {
		mockUow := m.Uow
		if mockUow == nil {
			mockUow = &MockUnitOfWork{}
		}
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

func (m *MockTaskRepository) Save(ctx context.Context, task *entity.Task) error   { return nil }
func (m *MockTaskRepository) Update(ctx context.Context, task *entity.Task) error { return nil }
func (m *MockTaskRepository) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
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

// setupTestUserUseCase creates a new UserUseCase with mock dependencies for testing.
func setupTestUserUseCase(
	mockUserRepo *MockUserRepository,
	mockRoleRepo *MockRoleRepository,
	mockCache *MockCacheRepository,
	mockEventPublisher *MockEventPublisher,
	mockTM *MockTransactionManager,
	mockValidator *MockValidator,
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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		mockUserRepo.On("FindByID", ctx, userID).Return(nil, nil)

		result, err := uc.AssignRole(ctx, userID, roleID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domainerror.ErrUserNotFound)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("role not found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

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
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		mockCache := new(MockCacheRepository)
		mockEventPublisher := new(MockEventPublisher)
		mockTM := new(MockTransactionManager)
		mockValidator := new(MockValidator)

		// Configure the mock UnitOfWork to use the same mockRoleRepo
		mockTM.Uow = &MockUnitOfWork{
			RoleRepositoryFn: func() output.RoleRepository { return mockRoleRepo },
		}

		uc := setupTestUserUseCase(mockUserRepo, mockRoleRepo, mockCache, mockEventPublisher, mockTM, mockValidator)

		// No existing roles
		mockRoleRepo.On("FindAll", ctx).Return([]*entity.Role{}, nil)
		// Role not found for each role name
		mockRoleRepo.On("FindByName", ctx, mock.AnythingOfType("string")).Return(nil, nil)
		mockRoleRepo.On("RemoveAllPermissions", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
		mockRoleRepo.On("Save", ctx, mock.AnythingOfType("*entity.Role")).Return(nil)
		mockTM.On("RunInTransaction", ctx, mock.Anything).Return(nil)

		err := uc.SeedSystemRoles(ctx)

		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
		mockTM.AssertExpectations(t)
	})
}
