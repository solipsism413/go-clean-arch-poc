package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
)

// Ensure ACLRepository implements the output.ACLRepository interface.
var _ output.ACLRepository = (*ACLRepository)(nil)

// ACLRepository implements the ACL repository using PostgreSQL.
type ACLRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewACLRepository creates a new ACLRepository.
func NewACLRepository(db sqlc.DBTX) *ACLRepository {
	return &ACLRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// Save creates a new ACL entry.
func (r *ACLRepository) Save(ctx context.Context, entry *entity.ACLEntry) error {
	_, err := r.queries.CreateACLEntry(ctx, sqlc.CreateACLEntryParams{
		ID:           entry.ID,
		ResourceType: string(entry.ResourceType),
		ResourceID:   entry.ResourceID,
		SubjectType:  string(entry.SubjectType),
		SubjectID:    entry.SubjectID,
		Permission:   string(entry.Permission),
		CreatedAt:    entry.CreatedAt,
	})
	return err
}

// Delete removes an ACL entry by ID.
func (r *ACLRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteACLEntry(ctx, id)
}

// FindByResource retrieves ACL entries for a specific resource.
func (r *ACLRepository) FindByResource(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID) ([]*entity.ACLEntry, error) {
	rows, err := r.queries.GetACLEntriesByResource(ctx, sqlc.GetACLEntriesByResourceParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]*entity.ACLEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, &entity.ACLEntry{
			ID:           row.ID,
			ResourceType: entity.ResourceType(row.ResourceType),
			ResourceID:   row.ResourceID,
			SubjectType:  entity.ACLSubjectType(row.SubjectType),
			SubjectID:    row.SubjectID,
			Permission:   entity.ACLPermission(row.Permission),
			CreatedAt:    row.CreatedAt,
		})
	}

	return entries, nil
}

// FindBySubject retrieves ACL entries for a specific subject.
func (r *ACLRepository) FindBySubject(ctx context.Context, subjectType entity.ACLSubjectType, subjectID uuid.UUID) ([]*entity.ACLEntry, error) {
	rows, err := r.queries.GetACLEntriesBySubject(ctx, sqlc.GetACLEntriesBySubjectParams{
		SubjectType: string(subjectType),
		SubjectID:   subjectID,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]*entity.ACLEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, &entity.ACLEntry{
			ID:           row.ID,
			ResourceType: entity.ResourceType(row.ResourceType),
			ResourceID:   row.ResourceID,
			SubjectType:  entity.ACLSubjectType(row.SubjectType),
			SubjectID:    row.SubjectID,
			Permission:   entity.ACLPermission(row.Permission),
			CreatedAt:    row.CreatedAt,
		})
	}

	return entries, nil
}

// HasPermission checks if a subject has a specific permission on a resource.
func (r *ACLRepository) HasPermission(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID, subjectType entity.ACLSubjectType, subjectID uuid.UUID, permission entity.ACLPermission) (bool, error) {
	return r.queries.HasPermission(ctx, sqlc.HasPermissionParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
		SubjectType:  string(subjectType),
		SubjectID:    subjectID,
		Permission:   string(permission),
	})
}

// DeleteByResource removes all ACL entries for a resource.
func (r *ACLRepository) DeleteByResource(ctx context.Context, resourceType entity.ResourceType, resourceID uuid.UUID) error {
	return r.queries.DeleteACLEntriesByResource(ctx, sqlc.DeleteACLEntriesByResourceParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
	})
}
