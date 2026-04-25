package label

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
)

var _ input.LabelService = (*LabelUseCase)(nil)

var errLabelNameAlreadyExists = domainerror.NewDomainError(domainerror.CodeConflict, "label name already exists")

type LabelUseCase struct {
	labelRepo output.LabelRepository
	validator validation.Validator
	logger    *slog.Logger
}

func NewLabelUseCase(labelRepo output.LabelRepository, validator validation.Validator, logger *slog.Logger) *LabelUseCase {
	return &LabelUseCase{
		labelRepo: labelRepo,
		validator: validator,
		logger:    logger,
	}
}

func (uc *LabelUseCase) CreateLabel(ctx context.Context, input dto.CreateLabelInput) (*dto.LabelOutput, error) {
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	existing, err := uc.labelRepo.FindByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errLabelNameAlreadyExists
	}

	label, err := entity.NewLabel(input.Name, input.Color)
	if err != nil {
		return nil, err
	}

	if err := uc.labelRepo.Save(ctx, label); err != nil {
		return nil, err
	}

	uc.logger.Info("label created", "labelId", label.ID, "name", label.Name)

	return dto.LabelFromEntity(label), nil
}

func (uc *LabelUseCase) UpdateLabel(ctx context.Context, id uuid.UUID, input dto.UpdateLabelInput) (*dto.LabelOutput, error) {
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	label, err := uc.labelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, domainerror.ErrLabelNotFound
	}

	if input.Name != nil {
		existing, err := uc.labelRepo.FindByName(ctx, *input.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != label.ID {
			return nil, errLabelNameAlreadyExists
		}
		if err := label.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}

	if input.Color != nil {
		if err := label.UpdateColor(*input.Color); err != nil {
			return nil, err
		}
	}

	if err := uc.labelRepo.Update(ctx, label); err != nil {
		return nil, err
	}

	uc.logger.Info("label updated", "labelId", label.ID, "name", label.Name)

	return dto.LabelFromEntity(label), nil
}

func (uc *LabelUseCase) DeleteLabel(ctx context.Context, id uuid.UUID) error {
	exists, err := uc.labelRepo.ExistsByID(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domainerror.ErrLabelNotFound
	}

	if err := uc.labelRepo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("label deleted", "labelId", id)

	return nil
}

func (uc *LabelUseCase) GetLabel(ctx context.Context, id uuid.UUID) (*dto.LabelOutput, error) {
	label, err := uc.labelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, domainerror.ErrLabelNotFound
	}

	return dto.LabelFromEntity(label), nil
}

func (uc *LabelUseCase) ListLabels(ctx context.Context) ([]*dto.LabelOutput, error) {
	labels, err := uc.labelRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	outputs := make([]*dto.LabelOutput, 0, len(labels))
	for _, label := range labels {
		outputs = append(outputs, dto.LabelFromEntity(label))
	}

	return outputs, nil
}
