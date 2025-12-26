package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// CreateUserInput represents the input for creating a user.
type CreateUserInput struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	Name     string `json:"name" validate:"required,min=1,max=255"`
}

// UpdateUserInput represents the input for updating a user.
type UpdateUserInput struct {
	Email *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
}

// UserOutput represents the output for user operations.
type UserOutput struct {
	ID        uuid.UUID    `json:"id"`
	Email     string       `json:"email"`
	Name      string       `json:"name"`
	Roles     []RoleOutput `json:"roles,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

// UserBasicOutput represents minimal user info for embedding in other DTOs.
type UserBasicOutput struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

// UserFromEntity converts a User entity to UserOutput DTO.
func UserFromEntity(user *entity.User) *UserOutput {
	if user == nil {
		return nil
	}
	output := &UserOutput{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     make([]RoleOutput, 0, len(user.Roles)),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	for _, role := range user.Roles {
		output.Roles = append(output.Roles, *RoleFromEntity(&role))
	}
	return output
}

// UserBasicFromEntity converts a User entity to UserBasicOutput DTO.
func UserBasicFromEntity(user *entity.User) *UserBasicOutput {
	if user == nil {
		return nil
	}
	return &UserBasicOutput{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}
}

// UserFilter represents filtering options for user queries.
type UserFilter struct {
	Search string     `json:"search,omitempty" validate:"max=100"`
	RoleID *uuid.UUID `json:"roleId,omitempty"`
}

// UserListOutput represents a paginated list of users.
type UserListOutput struct {
	Users      []*UserOutput `json:"users"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// LoginInput represents the input for user login.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthOutput represents the output for authentication operations.
type AuthOutput struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresAt    time.Time   `json:"expiresAt"`
	User         *UserOutput `json:"user"`
}

// ChangePasswordInput represents the input for changing password.
type ChangePasswordInput struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=100"`
}

// TokenClaims represents the claims extracted from a JWT token.
type TokenClaims struct {
	UserID      uuid.UUID   `json:"userId"`
	Email       string      `json:"email"`
	Roles       []string    `json:"roles"`
	RoleIDs     []uuid.UUID `json:"roleIds"`
	Permissions []string    `json:"permissions"`
	ExpiresAt   time.Time   `json:"expiresAt"`
}
