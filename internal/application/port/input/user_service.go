package input

import (
	"context"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
)

// UserService defines the input port for user-related use cases.
type UserService interface {
	// CreateUser creates a new user.
	CreateUser(ctx context.Context, input dto.CreateUserInput) (*dto.UserOutput, error)

	// UpdateUser updates an existing user.
	UpdateUser(ctx context.Context, id uuid.UUID, input dto.UpdateUserInput) (*dto.UserOutput, error)

	// DeleteUser deletes a user by ID.
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// GetUser retrieves a user by ID.
	GetUser(ctx context.Context, id uuid.UUID) (*dto.UserOutput, error)

	// GetUserByEmail retrieves a user by email.
	GetUserByEmail(ctx context.Context, email string) (*dto.UserOutput, error)

	// ListUsers retrieves users with filtering and pagination.
	ListUsers(ctx context.Context, filter dto.UserFilter, pagination dto.Pagination) (*dto.UserListOutput, error)

	// AssignRole assigns a role to a user.
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) (*dto.UserOutput, error)

	// RemoveRole removes a role from a user.
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) (*dto.UserOutput, error)

	// SeedSystemRoles seeds system-defined roles.
	SeedSystemRoles(ctx context.Context) error
}

// AuthService defines the input port for authentication use cases.
type AuthService interface {
	// Login authenticates a user and returns tokens.
	Login(ctx context.Context, input dto.LoginInput) (*dto.AuthOutput, error)

	// Logout invalidates the user's session/tokens.
	Logout(ctx context.Context, userID uuid.UUID) error

	// RefreshToken refreshes the access token.
	RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthOutput, error)

	// ChangePassword changes the user's password.
	ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordInput) error

	// ValidateToken validates an access token.
	ValidateToken(ctx context.Context, token string) (*dto.TokenClaims, error)
}
