package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ensure LabelRepository implements the output.LabelRepository interface.
var _ output.LabelRepository = (*LabelRepository)(nil)

// LabelRepository implements the label repository using PostgreSQL.
type LabelRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewLabelRepository creates a new LabelRepository.
func NewLabelRepository(pool *pgxpool.Pool) *LabelRepository {
	return &LabelRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Save creates a new label.
func (r *LabelRepository) Save(ctx context.Context, label *entity.Label) error {
	_, err := r.queries.CreateLabel(ctx, sqlc.CreateLabelParams{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		CreatedAt: label.CreatedAt,
		UpdatedAt: label.UpdatedAt,
	})
	return err
}

// Update updates an existing label.
func (r *LabelRepository) Update(ctx context.Context, label *entity.Label) error {
	_, err := r.queries.UpdateLabel(ctx, sqlc.UpdateLabelParams{
		ID:    label.ID,
		Name:  label.Name,
		Color: label.Color,
	})
	return err
}

// Delete removes a label by ID.
func (r *LabelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteLabel(ctx, id)
}

// FindByID retrieves a label by ID.
func (r *LabelRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Label, error) {
	row, err := r.queries.GetLabel(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entity.Label{
		ID:        row.ID,
		Name:      row.Name,
		Color:     row.Color,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// FindAll retrieves all labels.
func (r *LabelRepository) FindAll(ctx context.Context) ([]*entity.Label, error) {
	rows, err := r.queries.ListLabels(ctx)
	if err != nil {
		return nil, err
	}

	labels := make([]*entity.Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, &entity.Label{
			ID:        row.ID,
			Name:      row.Name,
			Color:     row.Color,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return labels, nil
}

// FindByTaskID retrieves labels for a specific task.
func (r *LabelRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]*entity.Label, error) {
	rows, err := r.queries.GetTaskLabels(ctx, taskID)
	if err != nil {
		return nil, err
	}

	labels := make([]*entity.Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, &entity.Label{
			ID:        row.ID,
			Name:      row.Name,
			Color:     row.Color,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return labels, nil
}

// ExistsByID checks if a label exists.
func (r *LabelRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.queries.LabelExists(ctx, id)
}

// Ensure ACLRepository implements the output.ACLRepository interface.
var _ output.ACLRepository = (*ACLRepository)(nil)

// ACLRepository implements the ACL repository using PostgreSQL.
type ACLRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewACLRepository creates a new ACLRepository.
func NewACLRepository(pool *pgxpool.Pool) *ACLRepository {
	return &ACLRepository{
		pool:    pool,
		queries: sqlc.New(pool),
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
