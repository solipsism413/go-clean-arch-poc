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

func TestACLRepository_Save(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	entry := &entity.ACLEntry{
		ID:           uuid.New(),
		ResourceType: entity.ResourceType("task"),
		ResourceID:   uuid.New(),
		SubjectType:  entity.ACLSubjectType("user"),
		SubjectID:    uuid.New(),
		Permission:   entity.ACLPermission("read"),
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(ctx, entry)
	require.NoError(t, err)

	// Verify by finding by resource
	entries, err := repo.FindByResource(ctx, entry.ResourceType, entry.ResourceID)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entry.Permission, entries[0].Permission)
}

func TestACLRepository_Delete(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	resourceID := uuid.New()
	entry := &entity.ACLEntry{
		ID:           uuid.New(),
		ResourceType: entity.ResourceType("task"),
		ResourceID:   resourceID,
		SubjectType:  entity.ACLSubjectType("user"),
		SubjectID:    uuid.New(),
		Permission:   entity.ACLPermission("write"),
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, entry))

	err := repo.Delete(ctx, entry.ID)
	require.NoError(t, err)

	// Verify deletion
	entries, err := repo.FindByResource(ctx, entry.ResourceType, resourceID)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestACLRepository_FindByResource(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	resourceID := uuid.New()
	resourceType := entity.ResourceType("task")

	// Create multiple ACL entries for same resource
	entries := []*entity.ACLEntry{
		{ID: uuid.New(), ResourceType: resourceType, ResourceID: resourceID, SubjectType: entity.ACLSubjectType("user"), SubjectID: uuid.New(), Permission: entity.ACLPermission("read"), CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), ResourceType: resourceType, ResourceID: resourceID, SubjectType: entity.ACLSubjectType("role"), SubjectID: uuid.New(), Permission: entity.ACLPermission("write"), CreatedAt: time.Now().UTC()},
	}
	for _, entry := range entries {
		require.NoError(t, repo.Save(ctx, entry))
	}

	found, err := repo.FindByResource(ctx, resourceType, resourceID)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestACLRepository_FindBySubject(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	subjectID := uuid.New()
	subjectType := entity.ACLSubjectType("user")

	// Create multiple ACL entries for same subject
	entries := []*entity.ACLEntry{
		{ID: uuid.New(), ResourceType: entity.ResourceType("task"), ResourceID: uuid.New(), SubjectType: subjectType, SubjectID: subjectID, Permission: entity.ACLPermission("read"), CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), ResourceType: entity.ResourceType("label"), ResourceID: uuid.New(), SubjectType: subjectType, SubjectID: subjectID, Permission: entity.ACLPermission("write"), CreatedAt: time.Now().UTC()},
	}
	for _, entry := range entries {
		require.NoError(t, repo.Save(ctx, entry))
	}

	found, err := repo.FindBySubject(ctx, subjectType, subjectID)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestACLRepository_HasPermission(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	resourceID := uuid.New()
	resourceType := entity.ResourceType("task")
	subjectID := uuid.New()
	subjectType := entity.ACLSubjectType("user")
	permission := entity.ACLPermission("read")

	entry := &entity.ACLEntry{
		ID:           uuid.New(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		Permission:   permission,
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, repo.Save(ctx, entry))

	t.Run("has permission returns true", func(t *testing.T) {
		has, err := repo.HasPermission(ctx, resourceType, resourceID, subjectType, subjectID, permission)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("no permission returns false", func(t *testing.T) {
		has, err := repo.HasPermission(ctx, resourceType, resourceID, subjectType, subjectID, entity.ACLPermission("write"))
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("different resource returns false", func(t *testing.T) {
		has, err := repo.HasPermission(ctx, resourceType, uuid.New(), subjectType, subjectID, permission)
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestACLRepository_DeleteByResource(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	repo := repository.NewACLRepository(testDB.Pool)

	resourceID := uuid.New()
	resourceType := entity.ResourceType("task")

	// Create multiple ACL entries for same resource
	entries := []*entity.ACLEntry{
		{ID: uuid.New(), ResourceType: resourceType, ResourceID: resourceID, SubjectType: entity.ACLSubjectType("user"), SubjectID: uuid.New(), Permission: entity.ACLPermission("read"), CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), ResourceType: resourceType, ResourceID: resourceID, SubjectType: entity.ACLSubjectType("role"), SubjectID: uuid.New(), Permission: entity.ACLPermission("write"), CreatedAt: time.Now().UTC()},
	}
	for _, entry := range entries {
		require.NoError(t, repo.Save(ctx, entry))
	}

	err := repo.DeleteByResource(ctx, resourceType, resourceID)
	require.NoError(t, err)

	// Verify all deleted
	found, err := repo.FindByResource(ctx, resourceType, resourceID)
	require.NoError(t, err)
	assert.Len(t, found, 0)
}
