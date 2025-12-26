package event

import (
	"time"

	"github.com/google/uuid"
)

// UserCreated is emitted when a new user is created.
type UserCreated struct {
	BaseEvent
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewUserCreated creates a new UserCreated event.
func NewUserCreated(userID uuid.UUID, email, name string) UserCreated {
	return UserCreated{
		BaseEvent: NewBaseEvent("user.created", userID),
		Email:     email,
		Name:      name,
	}
}

// UserUpdated is emitted when a user is updated.
type UserUpdated struct {
	BaseEvent
	Email *string `json:"email,omitempty"`
	Name  *string `json:"name,omitempty"`
}

// NewUserUpdated creates a new UserUpdated event.
func NewUserUpdated(userID uuid.UUID) UserUpdated {
	return UserUpdated{
		BaseEvent: NewBaseEvent("user.updated", userID),
	}
}

// WithEmail sets the email field for the update event.
func (e UserUpdated) WithEmail(email string) UserUpdated {
	e.Email = &email
	return e
}

// WithName sets the name field for the update event.
func (e UserUpdated) WithName(name string) UserUpdated {
	e.Name = &name
	return e
}

// UserDeleted is emitted when a user is deleted.
type UserDeleted struct {
	BaseEvent
}

// NewUserDeleted creates a new UserDeleted event.
func NewUserDeleted(userID uuid.UUID) UserDeleted {
	return UserDeleted{
		BaseEvent: NewBaseEvent("user.deleted", userID),
	}
}

// UserRoleAssigned is emitted when a role is assigned to a user.
type UserRoleAssigned struct {
	BaseEvent
	RoleID     uuid.UUID `json:"role_id"`
	RoleName   string    `json:"role_name"`
	AssignedBy uuid.UUID `json:"assigned_by"`
}

// NewUserRoleAssigned creates a new UserRoleAssigned event.
func NewUserRoleAssigned(userID, roleID uuid.UUID, roleName string, assignedBy uuid.UUID) UserRoleAssigned {
	return UserRoleAssigned{
		BaseEvent:  NewBaseEvent("user.role_assigned", userID),
		RoleID:     roleID,
		RoleName:   roleName,
		AssignedBy: assignedBy,
	}
}

// UserRoleRemoved is emitted when a role is removed from a user.
type UserRoleRemoved struct {
	BaseEvent
	RoleID    uuid.UUID `json:"role_id"`
	RoleName  string    `json:"role_name"`
	RemovedBy uuid.UUID `json:"removed_by"`
}

// NewUserRoleRemoved creates a new UserRoleRemoved event.
func NewUserRoleRemoved(userID, roleID uuid.UUID, roleName string, removedBy uuid.UUID) UserRoleRemoved {
	return UserRoleRemoved{
		BaseEvent: NewBaseEvent("user.role_removed", userID),
		RoleID:    roleID,
		RoleName:  roleName,
		RemovedBy: removedBy,
	}
}

// UserPasswordChanged is emitted when a user's password is changed.
type UserPasswordChanged struct {
	BaseEvent
	ChangedAt time.Time `json:"changed_at"`
}

// NewUserPasswordChanged creates a new UserPasswordChanged event.
func NewUserPasswordChanged(userID uuid.UUID) UserPasswordChanged {
	return UserPasswordChanged{
		BaseEvent: NewBaseEvent("user.password_changed", userID),
		ChangedAt: time.Now().UTC(),
	}
}

// UserLoggedIn is emitted when a user logs in.
type UserLoggedIn struct {
	BaseEvent
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// NewUserLoggedIn creates a new UserLoggedIn event.
func NewUserLoggedIn(userID uuid.UUID, ipAddress, userAgent string) UserLoggedIn {
	return UserLoggedIn{
		BaseEvent: NewBaseEvent("user.logged_in", userID),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

// UserLoggedOut is emitted when a user logs out.
type UserLoggedOut struct {
	BaseEvent
}

// NewUserLoggedOut creates a new UserLoggedOut event.
func NewUserLoggedOut(userID uuid.UUID) UserLoggedOut {
	return UserLoggedOut{
		BaseEvent: NewBaseEvent("user.logged_out", userID),
	}
}
