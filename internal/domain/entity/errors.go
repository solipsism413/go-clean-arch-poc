package entity

import "errors"

// Domain errors for entity validation and business rule violations.
// These errors are used throughout the domain layer to indicate
// specific validation failures or invariant violations.
var (
	// Task errors
	ErrEmptyTitle              = errors.New("task title cannot be empty")
	ErrInvalidPriority         = errors.New("invalid task priority")
	ErrInvalidStatus           = errors.New("invalid task status")
	ErrTaskArchived            = errors.New("cannot modify archived task")
	ErrTaskNotDone             = errors.New("task must be done before archiving")
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// User errors
	ErrEmptyEmail    = errors.New("email cannot be empty")
	ErrEmptyPassword = errors.New("password cannot be empty")
	ErrEmptyName     = errors.New("name cannot be empty")
	ErrInvalidEmail  = errors.New("invalid email format")

	// Role errors
	ErrEmptyRoleName = errors.New("role name cannot be empty")

	// Permission errors
	ErrEmptyPermissionName = errors.New("permission name cannot be empty")
	ErrEmptyResource       = errors.New("resource cannot be empty")
	ErrEmptyAction         = errors.New("action cannot be empty")

	// Label errors
	ErrEmptyLabelName  = errors.New("label name cannot be empty")
	ErrEmptyLabelColor = errors.New("label color cannot be empty")

	// ACL errors
	ErrEmptyResourceType = errors.New("resource type cannot be empty")
	ErrEmptySubjectType  = errors.New("subject type cannot be empty")
)
