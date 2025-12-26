package validation_test

import (
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
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

func TestValidatorInterface(t *testing.T) {
	// Test that StructValidator implements Validator interface
	var _ validation.Validator = validation.GetValidator()

	t.Run("MockValidator test", func(t *testing.T) {
		m := new(MockValidator)
		m.On("Validate", mock.Anything).Return(nil)

		err := m.Validate("some data")
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}
