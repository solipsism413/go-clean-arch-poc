// Package domainerror contains domain-specific errors that provide
// rich error information for the application layer.
package domainerror

import (
	"errors"
	"fmt"
)

// ErrorCode represents a domain error code.
type ErrorCode string

// Error codes for domain errors.
const (
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeValidation       ErrorCode = "VALIDATION_ERROR"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeInternalError    ErrorCode = "INTERNAL_ERROR"
	CodeInvalidOperation ErrorCode = "INVALID_OPERATION"
	CodeInvalidState     ErrorCode = "INVALID_STATE"
)

// DomainError represents a domain-specific error.
type DomainError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *DomainError) Unwrap() error {
	return e.Err
}

// Is implements error comparison.
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewDomainError creates a new domain error.
func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with a domain error.
func Wrap(code ErrorCode, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common domain errors.
var (
	// Not found errors
	ErrTaskNotFound       = NewDomainError(CodeNotFound, "task not found")
	ErrUserNotFound       = NewDomainError(CodeNotFound, "user not found")
	ErrRoleNotFound       = NewDomainError(CodeNotFound, "role not found")
	ErrPermissionNotFound = NewDomainError(CodeNotFound, "permission not found")
	ErrLabelNotFound      = NewDomainError(CodeNotFound, "label not found")

	// Conflict errors
	ErrUserEmailExists = NewDomainError(CodeConflict, "user with this email already exists")
	ErrRoleNameExists  = NewDomainError(CodeConflict, "role with this name already exists")

	// Authorization errors
	ErrUnauthorized       = NewDomainError(CodeUnauthorized, "authentication required")
	ErrForbidden          = NewDomainError(CodeForbidden, "access denied")
	ErrInvalidCredentials = NewDomainError(CodeUnauthorized, "invalid email or password")
	ErrInvalidToken       = NewDomainError(CodeUnauthorized, "invalid or expired token")
	ErrEmailAlreadyExists = NewDomainError(CodeConflict, "email already exists")

	// Invalid operation errors
	ErrInvalidStatusTransition = NewDomainError(CodeInvalidOperation, "invalid status transition")
	ErrCannotModifyArchived    = NewDomainError(CodeInvalidOperation, "cannot modify archived resource")
)

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == CodeNotFound
	}
	return false
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == CodeValidation
	}
	return false
}

// IsConflictError checks if the error is a conflict error.
func IsConflictError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == CodeConflict
	}
	return false
}

// IsUnauthorizedError checks if the error is an unauthorized error.
func IsUnauthorizedError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == CodeUnauthorized
	}
	return false
}

// IsForbiddenError checks if the error is a forbidden error.
func IsForbiddenError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == CodeForbidden
	}
	return false
}
