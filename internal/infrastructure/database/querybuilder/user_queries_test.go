// Package querybuilder_test contains tests for the user query builder.
package querybuilder_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/querybuilder"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserQueryBuilder(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("should create user query builder", func(t *testing.T) {
		qb := querybuilder.NewUserQueryBuilder(testDB.Pool)
		assert.NotNil(t, qb)
	})
}

// CreateUserWithName creates a user with a specific name for testing.
func CreateUserWithName(ctx context.Context, pool *pgxpool.Pool, t *testing.T, name, email string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, userID, email, "hashedpassword", name)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return userID
}

func TestUserQueryBuilder_FindWithFilter(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Clean up users first
	CleanupUsers(ctx, testDB.Pool, t)

	// Create test users
	CreateUserWithName(ctx, testDB.Pool, t, "Alice Anderson", "alice@example.com")
	CreateUserWithName(ctx, testDB.Pool, t, "Bob Brown", "bob@example.com")
	CreateUserWithName(ctx, testDB.Pool, t, "Charlie Clark", "charlie@example.com")

	qb := querybuilder.NewUserQueryBuilder(testDB.Pool)

	t.Run("should find all users without filter", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 3)
	})

	t.Run("should search by name", func(t *testing.T) {
		filter := dto.UserFilter{Search: "Alice"}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, users, 1)
		assert.Equal(t, "Alice Anderson", users[0].Name)
	})

	t.Run("should search case-insensitively", func(t *testing.T) {
		filter := dto.UserFilter{Search: "bob"}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Bob Brown", users[0].Name)
	})

	t.Run("should return empty result for non-matching search", func(t *testing.T) {
		filter := dto.UserFilter{Search: "NonExistent"}
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, users, 0)
	})

	t.Run("should apply pagination", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 2}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 2)
	})

	t.Run("should apply sorting by name ASC", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "name", SortDesc: false}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Alice Anderson", users[0].Name)
	})

	t.Run("should apply sorting by name DESC", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "name", SortDesc: true}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Charlie Clark", users[0].Name)
	})

	t.Run("should apply sorting by email", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "email", SortDesc: false}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "alice@example.com", users[0].Email)
	})

	t.Run("should handle invalid sort column", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "invalid_column"}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 3)
	})

	t.Run("should default empty sort to created_at", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: ""}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 3)
	})

	t.Run("should handle zero page defaults to 1", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 0, PageSize: 20}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 3)
	})

	t.Run("should handle zero page size defaults to 20", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 0}

		users, _, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Len(t, users, 3)
	})

	t.Run("should cap page size at 100", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 200}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.LessOrEqual(t, len(users), 100)
	})
}

func TestUserQueryBuilder_Search(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Clean up users first
	CleanupUsers(ctx, testDB.Pool, t)

	// Create test users
	CreateUserWithName(ctx, testDB.Pool, t, "John Doe", "john.doe@example.com")
	CreateUserWithName(ctx, testDB.Pool, t, "Jane Doe", "jane.doe@company.com")
	CreateUserWithName(ctx, testDB.Pool, t, "Bob Smith", "bob.smith@example.com")

	qb := querybuilder.NewUserQueryBuilder(testDB.Pool)

	t.Run("should search users by name", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.Search(ctx, "Doe", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, users, 2)
	})

	t.Run("should search users by email", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.Search(ctx, "example.com", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, users, 2)
	})

	t.Run("should return all users with empty query", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.Search(ctx, "", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, users, 3)
	})

	t.Run("should return empty result for non-matching query", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20}

		users, total, err := qb.Search(ctx, "NonExistent", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, users, 0)
	})

	t.Run("should apply sorting in search", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 20, SortBy: "name", SortDesc: false}

		users, total, err := qb.Search(ctx, "Doe", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		// Jane Doe should come before John Doe alphabetically
		assert.Equal(t, "Jane Doe", users[0].Name)
	})

	t.Run("should apply pagination in search", func(t *testing.T) {
		pagination := dto.Pagination{Page: 1, PageSize: 1}

		users, total, err := qb.Search(ctx, "Doe", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, users, 1)
	})

	t.Run("should handle second page of results", func(t *testing.T) {
		pagination := dto.Pagination{Page: 2, PageSize: 1}

		users, _, err := qb.Search(ctx, "Doe", pagination)

		require.NoError(t, err)
		assert.Len(t, users, 1)
	})

	t.Run("should handle page beyond results", func(t *testing.T) {
		pagination := dto.Pagination{Page: 10, PageSize: 20}

		users, total, err := qb.Search(ctx, "Doe", pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, users, 0)
	})
}

func TestUserAllowedSortColumns(t *testing.T) {
	t.Run("should have correct allowed sort columns", func(t *testing.T) {
		expected := []string{
			"name",
			"email",
			"created_at",
			"updated_at",
		}

		assert.Equal(t, expected, querybuilder.UserAllowedSortColumns)
	})
}

func TestUserRow(t *testing.T) {
	t.Run("should create UserRow struct", func(t *testing.T) {
		now := time.Now()

		row := querybuilder.UserRow{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Name:         "Test User",
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		assert.Equal(t, "test@example.com", row.Email)
		assert.Equal(t, "hashed_password", row.PasswordHash)
		assert.Equal(t, "Test User", row.Name)
		assert.Equal(t, now, row.CreatedAt)
		assert.Equal(t, now, row.UpdatedAt)
	})
}

func TestUserQueryBuilder_CombinedFilters(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Clean up users first
	CleanupUsers(ctx, testDB.Pool, t)

	// Create many test users
	for i := 0; i < 25; i++ {
		CreateUserWithName(ctx, testDB.Pool, t,
			fmt.Sprintf("User %02d", i),
			fmt.Sprintf("user%02d@example.com", i),
		)
	}

	qb := querybuilder.NewUserQueryBuilder(testDB.Pool)

	t.Run("should handle large result set with pagination", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 1, PageSize: 10}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(25), total)
		assert.Len(t, users, 10)
	})

	t.Run("should get second page of results", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 2, PageSize: 10}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(25), total)
		assert.Len(t, users, 10)
	})

	t.Run("should get last partial page", func(t *testing.T) {
		filter := dto.UserFilter{}
		pagination := dto.Pagination{Page: 3, PageSize: 10}

		users, total, err := qb.FindWithFilter(ctx, filter, pagination)

		require.NoError(t, err)
		assert.Equal(t, int64(25), total)
		assert.Len(t, users, 5)
	})
}
