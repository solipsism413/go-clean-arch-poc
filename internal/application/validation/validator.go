// Package validation provides input validation using go-playground/validator.
package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator defines the interface for data validation.
type Validator interface {
	Validate(data any) error
	ValidateVar(field any, tag string) error
}

// StructValidator wraps the go-playground validator with custom validations.
type StructValidator struct {
	validate *validator.Validate
}

// NewValidator creates a new instance of StructValidator.
func NewValidator() *StructValidator {
	v := validator.New()

	// Use JSON tag names in error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validations
	registerCustomValidations(v)

	return &StructValidator{validate: v}
}

// Validate validates a struct and returns validation errors.
func (v *StructValidator) Validate(data any) error {
	err := v.validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	return NewValidationError(validationErrs)
}

// ValidateVar validates a single variable.
func (v *StructValidator) ValidateVar(field any, tag string) error {
	return v.validate.Var(field, tag)
}

func registerCustomValidations(v *validator.Validate) {
	// Register hexcolor validation if not already present
	_ = v.RegisterValidation("hexcolor", validateHexColor)
}

func validateHexColor(fl validator.FieldLevel) bool {
	color := fl.Field().String()
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ValidationError represents a validation error with field details.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

// FieldError represents an error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	var sb strings.Builder
	sb.WriteString("validation errors: ")
	for i, err := range e.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Field)
		sb.WriteString(": ")
		sb.WriteString(err.Message)
	}
	return sb.String()
}

// NewValidationError creates a new ValidationError from validator.ValidationErrors.
func NewValidationError(errs validator.ValidationErrors) *ValidationError {
	fieldErrors := make([]FieldError, 0, len(errs))
	for _, err := range errs {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   err.Field(),
			Message: getErrorMessage(err),
			Tag:     err.Tag(),
			Value:   formatValue(err.Value()),
		})
	}
	return &ValidationError{Errors: fieldErrors}
}

func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short, minimum is " + err.Param()
	case "max":
		return "Value is too long, maximum is " + err.Param()
	case "oneof":
		return "Value must be one of: " + err.Param()
	case "hexcolor":
		return "Invalid hex color format (e.g., #FF5733)"
	case "uuid":
		return "Invalid UUID format"
	default:
		return "Invalid value"
	}
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		if len(s) > 50 {
			return s[:50] + "..."
		}
		return s
	}
	return ""
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
