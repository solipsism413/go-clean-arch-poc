package entity

// ResourceType represents the type of resource in the system.
type ResourceType string

const (
	// Plural forms (often used for RBAC/general scopes)
	ResourceTypeTasks       ResourceType = "tasks"
	ResourceTypeUsers       ResourceType = "users"
	ResourceTypeRoles       ResourceType = "roles"
	ResourceTypePermissions ResourceType = "permissions"
	ResourceTypeLabels      ResourceType = "labels"

	// Singular forms (often used for ACL/specific instances)
	ResourceTypeTask    ResourceType = "task"
	ResourceTypeUser    ResourceType = "user"
	ResourceTypeRole    ResourceType = "role"
	ResourceTypeLabel   ResourceType = "label"
	ResourceTypeProject ResourceType = "project"

	ResourceTypeAll ResourceType = "*"
)
