package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

// Ensure LabelRepository implements the output.LabelRepository interface.
var _ output.LabelRepository = (*LabelRepository)(nil)

// LabelRepository implements the label repository using PostgreSQL.
type LabelRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewLabelRepository creates a new LabelRepository.
func NewLabelRepository(db sqlc.DBTX) *LabelRepository {
	return &LabelRepository{
		db:      db,
		queries: sqlc.New(db),
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

// FindByName retrieves a label by name using case-insensitive matching.
func (r *LabelRepository) FindByName(ctx context.Context, name string) (*entity.Label, error) {
	row, err := r.queries.GetLabelByName(ctx, name)
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
