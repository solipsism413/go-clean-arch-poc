package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// UserFilter defines the filtering options for user queries.
type UserFilter struct {
	Search string
	RoleID *uuid.UUID
}

// UserRepository defines the output port for user persistence.
type UserRepository interface {
	// Save creates a new user.
	Save(ctx context.Context, user *entity.User) error

	// Update updates an existing user.
	Update(ctx context.Context, user *entity.User) error

	// Delete removes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a user by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)

	// FindByEmail retrieves a user by email.
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindAll retrieves users with filtering and pagination.
	FindAll(ctx context.Context, filter UserFilter, pagination Pagination) ([]*entity.User, *PaginatedResult, error)

	// ExistsByID checks if a user exists.
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// ExistsByEmail checks if a user with the given email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// RoleRepository defines the output port for role persistence.
type RoleRepository interface {
	// Save creates a new role.
	Save(ctx context.Context, role *entity.Role) error

	// Update updates an existing role.
	Update(ctx context.Context, role *entity.Role) error

	// Delete removes a role by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a role by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Role, error)

	// FindByName retrieves a role by name.
	FindByName(ctx context.Context, name string) (*entity.Role, error)

	// FindAll retrieves all roles.
	FindAll(ctx context.Context) ([]*entity.Role, error)

	// FindByUserID retrieves roles for a specific user.
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error)

	// ExistsByID checks if a role exists.
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// ExistsByName checks if a role with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// PermissionRepository defines the output port for permission persistence.
type PermissionRepository interface {
	// Save creates a new permission.
	Save(ctx context.Context, permission *entity.Permission) error

	// Delete removes a permission by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a permission by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error)

	// FindAll retrieves all permissions.
	FindAll(ctx context.Context) ([]*entity.Permission, error)

	// FindByRoleID retrieves permissions for a specific role.
	FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error)

	// ExistsByID checks if a permission exists.
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
}

// LabelRepository defines the output port for label persistence.
type LabelRepository interface {
	// Save creates a new label.
	Save(ctx context.Context, label *entity.Label) error

	// Update updates an existing label.
	Update(ctx context.Context, label *entity.Label) error

	// Delete removes a label by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a label by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Label, error)

	// FindAll retrieves all labels.
	FindAll(ctx context.Context) ([]*entity.Label, error)

	// FindByTaskID retrieves labels for a specific task.
	FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]*entity.Label, error)

	// ExistsByID checks if a label exists.
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
}

// ACLRepository defines the output port for ACL entry persistence.
type ACLRepository interface {
	// Save creates a new ACL entry.
	Save(ctx context.Context, entry *entity.ACLEntry) error

	// Delete removes an ACL entry by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByResource retrieves ACL entries for a specific resource.
	FindByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*entity.ACLEntry, error)

	// FindBySubject retrieves ACL entries for a specific subject.
	FindBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) ([]*entity.ACLEntry, error)

	// HasPermission checks if a subject has a specific permission on a resource.
	HasPermission(ctx context.Context, resourceType string, resourceID uuid.UUID, subjectType string, subjectID uuid.UUID, permission string) (bool, error)

	// DeleteByResource removes all ACL entries for a resource.
	DeleteByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) error
}
