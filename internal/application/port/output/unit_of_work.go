package output

import "context"

// UnitOfWork defines a transaction boundary for multiple repository operations.
type UnitOfWork interface {
	// Begin starts a new transaction.
	Begin(ctx context.Context) (UnitOfWork, error)

	// Commit commits the transaction.
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction.
	Rollback(ctx context.Context) error

	// TaskRepository returns the task repository within this unit of work.
	TaskRepository() TaskRepository

	// UserRepository returns the user repository within this unit of work.
	UserRepository() UserRepository

	// RoleRepository returns the role repository within this unit of work.
	RoleRepository() RoleRepository

	// PermissionRepository returns the permission repository within this unit of work.
	PermissionRepository() PermissionRepository

	// LabelRepository returns the label repository within this unit of work.
	LabelRepository() LabelRepository

	// ACLRepository returns the ACL repository within this unit of work.
	ACLRepository() ACLRepository
}

// TransactionManager provides a higher-level API for running transactional operations.
type TransactionManager interface {
	// RunInTransaction executes the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	RunInTransaction(ctx context.Context, fn func(uow UnitOfWork) error) error
}
