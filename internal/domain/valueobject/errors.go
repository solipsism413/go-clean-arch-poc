package valueobject

import "errors"

// Value object errors.
var (
	ErrInvalidTaskStatus = errors.New("invalid task status")
	ErrInvalidPriority   = errors.New("invalid priority")
	ErrEmptyEmail        = errors.New("email cannot be empty")
	ErrInvalidEmail      = errors.New("invalid email format")
)
