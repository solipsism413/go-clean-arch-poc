package task_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/usecase/task"
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

func TestTaskUseCase_CreateTask_Validation(t *testing.T) {
	// Simple test to verify that the mock validator is called
	mockValidator := new(MockValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// We don't need real repos for this validation check
	uc := task.NewTaskUseCase(nil, nil, nil, nil, nil, mockValidator, logger)

	input := dto.CreateTaskInput{
		Title: "Test Task",
	}

	mockValidator.On("Validate", input).Return(assert.AnError)

	ctx := context.Background()
	_, err := uc.CreateTask(ctx, input)

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockValidator.AssertExpectations(t)
}
