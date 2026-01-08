package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	role := &entity.Role{
		ID:          uuid.New(),
		Name:        "admin_test",
		Description: "Administrator role",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(ctx, role)
	require.NoError(t, err)

	saved, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, role.Name, saved.Name)
	assert.Equal(t, role.Description, saved.Description)
}

func TestRoleRepository_Update(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	role := &entity.Role{
		ID:          uuid.New(),
		Name:        "original_role",
		Description: "Original description",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	role.Name = "updated_role"
	role.Description = "Updated description"
	err := repo.Update(ctx, role)
	require.NoError(t, err)

	updated, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_role", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
}

func TestRoleRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	role := &entity.Role{
		ID:          uuid.New(),
		Name:        "to_delete_role",
		Description: "Will be deleted",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	err := repo.Delete(ctx, role.ID)
	require.NoError(t, err)

	deleted, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestRoleRepository_FindByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	t.Run("find existing", func(t *testing.T) {
		role := &entity.Role{
			ID:          uuid.New(),
			Name:        "findable_role",
			Description: "Find me",
			Permissions: []entity.Permission{},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, role))

		found, err := repo.FindByID(ctx, role.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, role.ID, found.ID)
	})

	t.Run("find non-existent returns nil", func(t *testing.T) {
		found, err := repo.FindByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestRoleRepository_FindByName(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	roleName := "findbyname_role"
	role := &entity.Role{
		ID:          uuid.New(),
		Name:        roleName,
		Description: "Find by name",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	t.Run("find existing by name", func(t *testing.T) {
		found, err := repo.FindByName(ctx, roleName)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, roleName, found.Name)
	})

	t.Run("find non-existent name returns nil", func(t *testing.T) {
		found, err := repo.FindByName(ctx, "nonexistent_role")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestRoleRepository_FindAll(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	// Clean existing roles
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM role_permissions")
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM user_roles")
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM roles")

	roles := []*entity.Role{
		{ID: uuid.New(), Name: "role1", Description: "Role 1", Permissions: []entity.Permission{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "role2", Description: "Role 2", Permissions: []entity.Permission{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	for _, role := range roles {
		require.NoError(t, repo.Save(ctx, role))
	}

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestRoleRepository_ExistsByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	role := &entity.Role{
		ID:          uuid.New(),
		Name:        "exists_role",
		Description: "Check existence",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, role.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists returns false", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRoleRepository_ExistsByName(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	roleName := "nameexists_role"
	role := &entity.Role{
		ID:          uuid.New(),
		Name:        roleName,
		Description: "Check name existence",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	t.Run("exists by name returns true", func(t *testing.T) {
		exists, err := repo.ExistsByName(ctx, roleName)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists by name returns false", func(t *testing.T) {
		exists, err := repo.ExistsByName(ctx, "notexists_role")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRoleRepository_DeleteByName(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewRoleRepository(testDB.Pool)

	roleName := "deletebyname_role"
	role := &entity.Role{
		ID:          uuid.New(),
		Name:        roleName,
		Description: "Delete by name",
		Permissions: []entity.Permission{},
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, role))

	err := repo.DeleteByName(ctx, roleName)
	require.NoError(t, err)

	exists, err := repo.ExistsByName(ctx, roleName)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPermissionRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewPermissionRepository(testDB.Pool)

	permission := &entity.Permission{
		ID:        uuid.New(),
		Name:      "test_permission",
		Resource:  entity.ResourceType("task"),
		Action:    entity.PermissionAction("read"),
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(ctx, permission)
	require.NoError(t, err)

	saved, err := repo.FindByID(ctx, permission.ID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, permission.Name, saved.Name)
}

func TestPermissionRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewPermissionRepository(testDB.Pool)

	permission := &entity.Permission{
		ID:        uuid.New(),
		Name:      "delete_permission",
		Resource:  entity.ResourceType("task"),
		Action:    entity.PermissionAction("write"),
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, permission))

	err := repo.Delete(ctx, permission.ID)
	require.NoError(t, err)

	deleted, err := repo.FindByID(ctx, permission.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestPermissionRepository_FindByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewPermissionRepository(testDB.Pool)

	t.Run("find existing", func(t *testing.T) {
		permission := &entity.Permission{
			ID:        uuid.New(),
			Name:      "findable_permission",
			Resource:  entity.ResourceType("user"),
			Action:    entity.PermissionAction("read"),
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, repo.Save(ctx, permission))

		found, err := repo.FindByID(ctx, permission.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, permission.ID, found.ID)
	})

	t.Run("find non-existent returns nil", func(t *testing.T) {
		found, err := repo.FindByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestPermissionRepository_FindAll(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewPermissionRepository(testDB.Pool)

	// Clean existing permissions
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM role_permissions")
	_, _ = testDB.Pool.Exec(ctx, "DELETE FROM permissions")

	permissions := []*entity.Permission{
		{ID: uuid.New(), Name: "perm1", Resource: entity.ResourceType("task"), Action: entity.PermissionAction("read"), CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "perm2", Resource: entity.ResourceType("task"), Action: entity.PermissionAction("write"), CreatedAt: time.Now().UTC()},
	}
	for _, perm := range permissions {
		require.NoError(t, repo.Save(ctx, perm))
	}

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestPermissionRepository_ExistsByID(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewPermissionRepository(testDB.Pool)

	permission := &entity.Permission{
		ID:        uuid.New(),
		Name:      "exists_permission",
		Resource:  entity.ResourceType("label"),
		Action:    entity.PermissionAction("delete"),
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, permission))

	t.Run("exists returns true", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, permission.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists returns false", func(t *testing.T) {
		exists, err := repo.ExistsByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
