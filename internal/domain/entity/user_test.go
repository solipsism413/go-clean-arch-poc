package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		userName string
		wantErr  error
	}{
		{
			name:     "valid user",
			email:    "test@example.com",
			password: "password123",
			userName: "Test User",
			wantErr:  nil,
		},
		{
			name:     "empty email",
			email:    "",
			password: "password123",
			userName: "Test User",
			wantErr:  entity.ErrEmptyEmail,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			userName: "Test User",
			wantErr:  entity.ErrEmptyPassword,
		},
		{
			name:     "empty name",
			email:    "test@example.com",
			password: "password123",
			userName: "",
			wantErr:  entity.ErrEmptyName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := entity.NewUser(tt.email, tt.password, tt.userName)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewUser() unexpected error = %v", err)
				return
			}

			if user.Email != tt.email {
				t.Errorf("NewUser() email = %v, want %v", user.Email, tt.email)
			}
			if user.Name != tt.userName {
				t.Errorf("NewUser() name = %v, want %v", user.Name, tt.userName)
			}
			if user.PasswordHash == "" {
				t.Error("NewUser() password should be hashed")
			}
			if user.PasswordHash == tt.password {
				t.Error("NewUser() password should not be stored in plain text")
			}
			if user.ID == uuid.Nil {
				t.Error("NewUser() ID should not be nil")
			}
		})
	}
}

func TestUser_VerifyPassword(t *testing.T) {
	password := "password123"
	user, _ := entity.NewUser("test@example.com", password, "Test User")

	t.Run("correct password", func(t *testing.T) {
		if !user.VerifyPassword(password) {
			t.Error("VerifyPassword() should return true for correct password")
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		if user.VerifyPassword("wrongpassword") {
			t.Error("VerifyPassword() should return false for incorrect password")
		}
	})
}

func TestUser_UpdatePassword(t *testing.T) {
	user, _ := entity.NewUser("test@example.com", "oldpassword", "Test User")

	t.Run("update password", func(t *testing.T) {
		newPassword := "newpassword123"
		err := user.UpdatePassword(newPassword)

		if err != nil {
			t.Errorf("UpdatePassword() error = %v", err)
		}
		if !user.VerifyPassword(newPassword) {
			t.Error("UpdatePassword() new password should be verified")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		err := user.UpdatePassword("")

		if err != entity.ErrEmptyPassword {
			t.Errorf("UpdatePassword() error = %v, want %v", err, entity.ErrEmptyPassword)
		}
	})
}

func TestUser_Roles(t *testing.T) {
	user, _ := entity.NewUser("test@example.com", "password123", "Test User")
	role, _ := entity.NewRole(entity.RoleAdmin, "Administrator role")

	t.Run("assign role", func(t *testing.T) {
		user.AssignRole(*role)

		if len(user.Roles) != 1 {
			t.Errorf("AssignRole() should add role, got %d roles", len(user.Roles))
		}
	})

	t.Run("assign duplicate role", func(t *testing.T) {
		user.AssignRole(*role)

		if len(user.Roles) != 1 {
			t.Error("AssignRole() should not add duplicate role")
		}
	})

	t.Run("has role", func(t *testing.T) {
		if !user.HasRole(entity.RoleAdmin) {
			t.Error("HasRole() should return true for assigned role")
		}
		if user.HasRole("nonexistent") {
			t.Error("HasRole() should return false for non-assigned role")
		}
	})

	t.Run("remove role", func(t *testing.T) {
		user.RemoveRole(role.ID)

		if len(user.Roles) != 0 {
			t.Error("RemoveRole() should remove the role")
		}
	})
}

func TestUser_HasPermission(t *testing.T) {
	user, _ := entity.NewUser("test@example.com", "password123", "Test User")
	role, _ := entity.NewRole(entity.RoleAdmin, "Administrator role")
	permission, _ := entity.NewPermission("create_task", "task", "create")
	role.AddPermission(*permission)
	user.AssignRole(*role)

	t.Run("has permission", func(t *testing.T) {
		if !user.HasPermission("task", "create") {
			t.Error("HasPermission() should return true for granted permission")
		}
	})

	t.Run("missing permission", func(t *testing.T) {
		if user.HasPermission("task", "delete") {
			t.Error("HasPermission() should return false for non-granted permission")
		}
	})
}

func TestUser_IsAdmin(t *testing.T) {
	t.Run("admin user", func(t *testing.T) {
		user, _ := entity.NewUser("admin@example.com", "password123", "Admin User")
		adminRole, _ := entity.NewRole(entity.RoleAdmin, "Administrator role")
		user.AssignRole(*adminRole)

		if !user.IsAdmin() {
			t.Error("IsAdmin() should return true for admin user")
		}
	})

	t.Run("non-admin user", func(t *testing.T) {
		user, _ := entity.NewUser("user@example.com", "password123", "Regular User")

		if user.IsAdmin() {
			t.Error("IsAdmin() should return false for non-admin user")
		}
	})
}

func TestUser_GetAllPermissions(t *testing.T) {
	user, _ := entity.NewUser("test@example.com", "password123", "Test User")

	role1, _ := entity.NewRole("role1", "Role 1")
	perm1, _ := entity.NewPermission("perm1", "resource1", "action1")
	perm2, _ := entity.NewPermission("perm2", "resource2", "action2")
	role1.AddPermission(*perm1)
	role1.AddPermission(*perm2)

	role2, _ := entity.NewRole("role2", "Role 2")
	perm3, _ := entity.NewPermission("perm3", "resource3", "action3")
	role2.AddPermission(*perm3)
	role2.AddPermission(*perm1) // Duplicate permission

	user.AssignRole(*role1)
	user.AssignRole(*role2)

	permissions := user.GetAllPermissions()

	// Should have 3 unique permissions (perm1 is duplicated but should appear once)
	if len(permissions) != 3 {
		t.Errorf("GetAllPermissions() got %d permissions, want 3", len(permissions))
	}
}
