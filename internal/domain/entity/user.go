package entity

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the task management system.
// Users can create, assign, and manage tasks based on their roles and permissions.
type User struct {
	// ID is the unique identifier for the user.
	ID uuid.UUID

	// Email is the user's email address, used for authentication.
	Email string

	// PasswordHash is the bcrypt hash of the user's password.
	PasswordHash string

	// Name is the display name of the user.
	Name string

	// Roles contains the roles assigned to this user.
	Roles []Role

	// CreatedAt is the timestamp when the user was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt time.Time
}

// NewUser creates a new User with the given parameters.
// The password is hashed before storing.
func NewUser(email, password, name string) (*User, error) {
	if email == "" {
		return nil, ErrEmptyEmail
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}
	if name == "" {
		return nil, ErrEmptyName
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		Roles:        make([]Role, 0),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// VerifyPassword checks if the provided password matches the stored hash.
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// UpdatePassword updates the user's password.
func (u *User) UpdatePassword(newPassword string) error {
	if newPassword == "" {
		return ErrEmptyPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateName updates the user's display name.
func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyName
	}
	u.Name = name
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateEmail updates the user's email address.
func (u *User) UpdateEmail(email string) error {
	if email == "" {
		return ErrEmptyEmail
	}
	u.Email = email
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// AssignRole assigns a role to the user.
func (u *User) AssignRole(role Role) {
	for _, r := range u.Roles {
		if r.ID == role.ID {
			return // Role already assigned
		}
	}
	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now().UTC()
}

// RemoveRole removes a role from the user.
func (u *User) RemoveRole(roleID uuid.UUID) {
	for i, r := range u.Roles {
		if r.ID == roleID {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

// HasRole checks if the user has the specified role.
func (u *User) HasRole(roleName string) bool {
	for _, r := range u.Roles {
		if r.Name == roleName {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has the specified permission.
// The permission is checked against all roles assigned to the user.
func (u *User) HasPermission(resource, action string) bool {
	for _, role := range u.Roles {
		if role.HasPermission(resource, action) {
			return true
		}
	}
	return false
}

// GetAllPermissions returns all unique permissions from all roles.
func (u *User) GetAllPermissions() []Permission {
	permMap := make(map[uuid.UUID]Permission)
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			permMap[perm.ID] = perm
		}
	}

	permissions := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		permissions = append(permissions, perm)
	}
	return permissions
}

// IsAdmin checks if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}
