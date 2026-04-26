package validation_test

import (
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidatorInterface(t *testing.T) {
	// Test that StructValidator implements Validator interface
	var _ validation.Validator = validation.NewValidator()

	t.Run("MockValidator test", func(t *testing.T) {
		m := new(mocks.MockValidator)
		m.On("Validate", mock.Anything).Return(nil)

		err := m.Validate("some data")
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}
