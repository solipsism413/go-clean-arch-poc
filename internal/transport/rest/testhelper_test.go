package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	authusecase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/auth"
	labelusecase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/label"
	taskusecase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/task"
	userusecase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/acl"
	"github.com/handiism/go-clean-arch-poc/internal/auth/rbac"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestApp holds all the test infrastructure
type TestApp struct {
	Server       *httptest.Server
	Pool         *pgxpool.Pool
	Container    *tcpostgres.PostgresContainer
	FileStorage  *testFileStorage
	TokenService *jwt.TokenService
	Logger       *slog.Logger
}

// TestUser represents a test user with token
type TestUser struct {
	ID          uuid.UUID
	Email       string
	Password    string
	AccessToken string
}

// SetupTestApp creates a full test application with real database
func SetupTestApp(t *testing.T) *TestApp {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	runMigrations(t, pool)

	// Create logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create JWT config
	jwtConfig := config.JWTConfig{
		SecretKey:            "test-secret-key-for-testing-only-32bytes",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test-app",
	}
	tokenService := jwt.NewTokenService(jwtConfig)

	// Create repositories
	userRepo := repository.NewUserRepository(pool)
	roleRepo := repository.NewRoleRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	labelRepo := repository.NewLabelRepository(pool)
	aclRepo := repository.NewACLRepository(pool)

	// Create transaction manager
	tm := postgres.NewTransactionManager(pool)

	// Create mock cache and event publisher using mockery-generated mocks
	cache := new(mocks.MockCacheRepository)
	eventPublisher := new(mocks.MockEventPublisher)

	// Set up default cache expectations for integration tests
	cache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
	cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	cache.On("Exists", mock.Anything, mock.Anything).Return(func(ctx context.Context, key string) (bool, error) {
		// Treat blacklist keys as "not revoked"; everything else as existing
		return !strings.Contains(key, ":token:blacklist:"), nil
	})
	cache.On("SetNX", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	cache.On("Expire", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cache.On("Increment", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	cache.On("DeletePattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("GetMultiple", mock.Anything, mock.Anything).Return(nil, nil)
	cache.On("SetMultiple", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cache.On("GetJSON", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("cache miss"))
	cache.On("SetJSON", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Set up default event publisher expectations
	eventPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	eventPublisher.On("PublishBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	eventPublisher.On("Close").Return(nil)

	// Create validator
	validator := validation.NewValidator()

	// Create use cases
	authUseCase := authusecase.NewAuthUseCase(userRepo, roleRepo, cache, eventPublisher, tm, tokenService, validator, logger)
	userUseCase := userusecase.NewUserUseCase(userRepo, roleRepo, cache, eventPublisher, tm, validator, logger)
	// Create in-memory file storage for tests
	fileStorage := &testFileStorage{files: make(map[string][]byte)}

	taskUseCase := taskusecase.NewTaskUseCase(taskRepo, taskRepo, userRepo, labelRepo, fileStorage, cache, eventPublisher, tm, validator, logger)
	labelUseCase := labelusecase.NewLabelUseCase(labelRepo, validator, logger)
	require.NoError(t, userUseCase.SeedSystemRoles(ctx))

	// Create RBAC authorizer and ACL checker
	authorizer := rbac.NewAuthorizer()
	aclChecker := acl.NewChecker(aclRepo)

	// Create auth middleware
	authMiddleware := auth.NewMiddleware(authUseCase, userUseCase, authorizer, aclChecker)

	// Create REST router
	router := rest.NewRouter(taskUseCase, userUseCase, authUseCase, labelUseCase, authMiddleware, aclChecker, logger)

	// Create test server
	server := httptest.NewServer(router)

	return &TestApp{
		Server:       server,
		Pool:         pool,
		Container:    container,
		FileStorage:  fileStorage,
		TokenService: tokenService,
		Logger:       logger,
	}
}

// Cleanup cleans up test resources
func (app *TestApp) Cleanup(t *testing.T) {
	t.Helper()
	app.Server.Close()
	app.Pool.Close()
	if err := app.Container.Terminate(context.Background()); err != nil {
		t.Logf("Failed to terminate container: %v", err)
	}
}

// CreateTestUser creates a test user and returns it with access token
func (app *TestApp) CreateTestUser(t *testing.T, email, password string) *TestUser {
	t.Helper()
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Create user entity
	user := &entity.User{
		ID:        userID,
		Email:     email,
		Name:      "Test User",
		Roles:     []entity.Role{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, user.UpdatePassword(password))

	// Save to database
	userRepo := repository.NewUserRepository(app.Pool)
	require.NoError(t, userRepo.Save(ctx, user))

	// Generate token
	authOutput, err := app.TokenService.GenerateTokenPair(ctx, userID, email, []string{}, []uuid.UUID{}, []string{})
	require.NoError(t, err)

	return &TestUser{
		ID:          userID,
		Email:       email,
		Password:    password,
		AccessToken: authOutput.AccessToken,
	}
}

// DoRequest performs an HTTP request to the test server
func (app *TestApp) DoRequest(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, app.Server.URL+path, bodyReader)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// DoMultipartRequest performs a multipart form HTTP request to the test server
func (app *TestApp) DoMultipartRequest(t *testing.T, method, path string, body io.Reader, contentType, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, app.Server.URL+path, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// ParseResponse parses JSON response into the given type
func ParseResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()

	var result T
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

// runMigrations runs database migrations
func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	migrationsPath := findMigrationsPath()
	files, err := os.ReadDir(migrationsPath)
	require.NoError(t, err)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" && filepath.Ext(file.Name()[:len(file.Name())-4]) == ".up" {
			content, err := os.ReadFile(filepath.Join(migrationsPath, file.Name()))
			require.NoError(t, err)
			_, err = pool.Exec(context.Background(), string(content))
			require.NoError(t, err, "Failed to run migration: %s", file.Name())
		}
	}
}

// findMigrationsPath finds the migrations directory
func findMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	for i := 0; i < 10; i++ {
		migrationsPath := filepath.Join(dir, "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}
		dir = filepath.Dir(dir)
	}

	panic("migrations directory not found")
}

// testFileStorage is a simple in-memory file storage for tests.
type testFileStorage struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func (fs *testFileStorage) Upload(ctx context.Context, key string, reader io.Reader, options output.UploadOptions) (*output.FileMetadata, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[key] = data
	return &output.FileMetadata{Key: key, Size: int64(len(data))}, nil
}

func (fs *testFileStorage) Download(ctx context.Context, key string) (io.ReadCloser, *output.FileMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	data, ok := fs.files[key]
	if !ok {
		return nil, nil, errors.New("file not found")
	}
	return io.NopCloser(bytes.NewReader(data)), &output.FileMetadata{Key: key, Size: int64(len(data))}, nil
}

func (fs *testFileStorage) Delete(ctx context.Context, key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.files, key)
	return nil
}

func (fs *testFileStorage) Exists(ctx context.Context, key string) (bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, ok := fs.files[key]
	return ok, nil
}

func (fs *testFileStorage) GetMetadata(ctx context.Context, key string) (*output.FileMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	data, ok := fs.files[key]
	if !ok {
		return nil, errors.New("file not found")
	}
	return &output.FileMetadata{Key: key, Size: int64(len(data))}, nil
}

func (fs *testFileStorage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return "https://example.com/" + key, nil
}

func (fs *testFileStorage) GenerateUploadURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return "https://example.com/upload/" + key, nil
}

func (fs *testFileStorage) List(ctx context.Context, prefix string, maxKeys int) ([]*output.FileMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var result []*output.FileMetadata
	for key, data := range fs.files {
		if strings.HasPrefix(key, prefix) {
			result = append(result, &output.FileMetadata{Key: key, Size: int64(len(data))})
		}
	}
	return result, nil
}

func (fs *testFileStorage) Copy(ctx context.Context, sourceKey, destKey string) (*output.FileMetadata, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	data, ok := fs.files[sourceKey]
	if !ok {
		return nil, errors.New("source file not found")
	}
	fs.files[destKey] = data
	return &output.FileMetadata{Key: destKey, Size: int64(len(data))}, nil
}
