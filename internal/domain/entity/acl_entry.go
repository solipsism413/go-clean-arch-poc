package entity

import (
	"time"

	"github.com/google/uuid"
)

// ACLEntry represents an Access Control List entry for fine-grained permissions.
// ACL entries define permissions on specific resources for specific subjects.
type ACLEntry struct {
	// ID is the unique identifier for the ACL entry.
	ID uuid.UUID

	// ResourceType is the type of resource (e.g., "task", "project").
	ResourceType string

	// ResourceID is the ID of the specific resource.
	ResourceID uuid.UUID

	// SubjectType is the type of subject (e.g., "user", "role").
	SubjectType string

	// SubjectID is the ID of the subject.
	SubjectID uuid.UUID

	// Permission is the permission granted (e.g., "read", "write", "delete", "admin").
	Permission string

	// CreatedAt is the timestamp when the ACL entry was created.
	CreatedAt time.Time
}

// NewACLEntry creates a new ACLEntry with the given parameters.
func NewACLEntry(resourceType string, resourceID uuid.UUID, subjectType string, subjectID uuid.UUID, permission string) (*ACLEntry, error) {
	if resourceType == "" {
		return nil, ErrEmptyResourceType
	}
	if subjectType == "" {
		return nil, ErrEmptySubjectType
	}
	if permission == "" {
		return nil, ErrEmptyAction
	}

	return &ACLEntry{
		ID:           uuid.New(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		Permission:   permission,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// Matches checks if this ACL entry matches the given parameters.
func (a *ACLEntry) Matches(resourceType string, resourceID uuid.UUID, subjectType string, subjectID uuid.UUID, permission string) bool {
	return a.ResourceType == resourceType &&
		a.ResourceID == resourceID &&
		a.SubjectType == subjectType &&
		a.SubjectID == subjectID &&
		(a.Permission == permission || a.Permission == "admin")
}

// GrantsPermission checks if this ACL entry grants the specified permission.
// Admin permission grants all other permissions.
func (a *ACLEntry) GrantsPermission(permission string) bool {
	if a.Permission == "admin" {
		return true
	}
	return a.Permission == permission
}
