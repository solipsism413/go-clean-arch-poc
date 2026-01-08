package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	user := &entity.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: "hashedpassword123",
		Name:         "Test User",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(ctx, user)
	require.NoError(t, err)

	saved, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, user.Email, saved.Email)
	assert.Equal(t, user.Name, saved.Name)
}

func TestUserRepository_Update(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	user := &entity.User{
		ID:           uuid.New(),
		Email:        "original@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Original Name",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, user))

	user.Email = "updated@example.com"
	user.Name = "Updated Name"
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	updated, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", updated.Email)
	assert.Equal(t, "Updated Name", updated.Name)
}

func TestUserRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	user := &entity.User{
		ID:           uuid.New(),
		Email:        "delete@example.com",
		PasswordHash: "hashedpassword",
		Name:         "To Delete",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, user))

	err := repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	deleted, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestUserRepository_FindByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	t.Run("find existing", func(t *testing.T) {
		user := &entity.User{
			ID:           uuid.New(),
			Email:        "findbyid@example.com",
			PasswordHash: "hashedpassword",
			Name:         "Find Me",
			Roles:        []entity.Role{},
			CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, user))

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("find non-existent returns nil", func(t *testing.T) {
		found, err := repo.FindByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	email := "findbyemail@example.com"
	user := &entity.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hashedpassword",
		Name:         "Email User",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, user))

	t.Run("find existing by email", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, email, found.Email)
	})

	t.Run("find non-existent email returns nil", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, "nonexistent@example.com")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserRepository_FindAll(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	// Clean users (but keep any test users we need)
	users := []*entity.User{
		{ID: uuid.New(), Email: "user1@example.com", PasswordHash: "hash", Name: "User One", Roles: []entity.Role{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), Email: "user2@example.com", PasswordHash: "hash", Name: "User Two", Roles: []entity.Role{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), Email: "searchable@example.com", PasswordHash: "hash", Name: "Searchable User", Roles: []entity.Role{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	for _, user := range users {
		require.NoError(t, repo.Save(ctx, user))
	}

	t.Run("find all without filter", func(t *testing.T) {
		all, pagination, err := repo.FindAll(ctx, output.UserFilter{}, output.Pagination{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(all), 3)
		assert.NotNil(t, pagination)
	})

	t.Run("find with search filter", func(t *testing.T) {
		filtered, _, err := repo.FindAll(ctx, output.UserFilter{Search: "Searchable"}, output.Pagination{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "Searchable User", filtered[0].Name)
	})
}

func TestUserRepository_ExistsByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	user := &entity.User{
		ID:           uuid.New(),
		Email:        "exists@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Exists User",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, user))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, user.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists returns false", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewUserRepository(testDB.Pool)

	email := "emailexists@example.com"
	user := &entity.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hashedpassword",
		Name:         "Email Exists User",
		Roles:        []entity.Role{},
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, user))

	t.Run("exists by email returns true", func(t *testing.T) {
		exists, err := repo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists by email returns false", func(t *testing.T) {
		exists, err := repo.ExistsByEmail(ctx, "notexists@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
