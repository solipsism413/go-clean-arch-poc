package graphql

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
)

// ===== User Mappers =====

func userToGraphQL(u *dto.UserOutput) *User {
	if u == nil {
		return nil
	}
	roles := make([]*Role, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, roleToGraphQL(&r))
	}
	return &User{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func userBasicToGraphQL(u *dto.UserBasicOutput) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
		Roles: []*Role{},
	}
}

// ===== Role Mappers =====

func roleToGraphQL(r *dto.RoleOutput) *Role {
	if r == nil {
		return nil
	}
	perms := make([]*Permission, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, permissionToGraphQL(&p))
	}
	var desc *string
	if r.Description != "" {
		desc = &r.Description
	}
	return &Role{
		ID:          r.ID,
		Name:        r.Name,
		Description: desc,
		Permissions: perms,
		CreatedAt:   r.CreatedAt,
	}
}

// ===== Permission Mappers =====

func permissionToGraphQL(p *dto.PermissionOutput) *Permission {
	if p == nil {
		return nil
	}
	return &Permission{
		ID:       p.ID,
		Name:     p.Name,
		Resource: p.Resource,
		Action:   p.Action,
	}
}

// ===== Label Mappers =====

func labelToGraphQL(l *dto.LabelOutput) *Label {
	if l == nil {
		return nil
	}
	return &Label{
		ID:        l.ID,
		Name:      l.Name,
		Color:     l.Color,
		CreatedAt: l.CreatedAt,
	}
}

// ===== Task Mappers =====

func taskToGraphQL(t *dto.TaskOutput, assignee, creator *dto.UserOutput) *Task {
	if t == nil {
		return nil
	}
	labels := make([]*Label, 0, len(t.Labels))
	for _, l := range t.Labels {
		labels = append(labels, labelToGraphQL(&l))
	}
	var desc *string
	if t.Description != "" {
		desc = &t.Description
	}
	return &Task{
		ID:          t.ID,
		Title:       t.Title,
		Description: desc,
		Status:      TaskStatus(t.Status),
		Priority:    Priority(t.Priority),
		DueDate:     t.DueDate,
		Assignee:    userToGraphQL(assignee),
		Creator:     userToGraphQL(creator),
		Labels:      labels,
		Attachments: []*TaskAttachment{},
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func taskAttachmentToGraphQL(attachment *dto.TaskAttachmentOutput) *TaskAttachment {
	if attachment == nil {
		return nil
	}
	return &TaskAttachment{
		ID:          attachment.ID,
		TaskID:      attachment.TaskID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		SizeBytes:   int(attachment.SizeBytes),
		UploadedBy:  attachment.UploadedBy,
		CreatedAt:   attachment.CreatedAt,
	}
}

func taskAttachmentListToGraphQL(list *dto.TaskAttachmentListOutput) []*TaskAttachment {
	if list == nil {
		return []*TaskAttachment{}
	}
	attachments := make([]*TaskAttachment, 0, len(list.Attachments))
	for i := range list.Attachments {
		attachments = append(attachments, taskAttachmentToGraphQL(&list.Attachments[i]))
	}
	return attachments
}

func taskAttachmentDownloadToGraphQL(attachment *dto.TaskAttachmentOutput, content []byte) *TaskAttachmentDownload {
	if attachment == nil {
		return nil
	}
	return &TaskAttachmentDownload{
		Attachment:    taskAttachmentToGraphQL(attachment),
		ContentBase64: base64.StdEncoding.EncodeToString(content),
	}
}

// ===== Auth Mappers =====

func authToGraphQL(a *dto.AuthOutput) *AuthPayload {
	if a == nil {
		return nil
	}
	return &AuthPayload{
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		ExpiresAt:    a.ExpiresAt,
		User:         userToGraphQL(a.User),
	}
}

// ===== Pagination Mappers =====

func paginationFromInput(p *PaginationInput) dto.Pagination {
	pagination := dto.DefaultPagination()
	if p != nil {
		if p.Page != nil && *p.Page > 0 {
			pagination.Page = *p.Page
		}
		if p.PageSize != nil && *p.PageSize > 0 {
			pagination.PageSize = *p.PageSize
		}
		if p.SortBy != nil {
			pagination.SortBy = *p.SortBy
		}
		if p.SortDesc != nil {
			pagination.SortDesc = *p.SortDesc
		}
	}
	return pagination
}

func taskConnectionToGraphQL(list *dto.TaskListOutput) *TaskConnection {
	if list == nil {
		return &TaskConnection{
			Edges:      []*TaskEdge{},
			PageInfo:   &PageInfo{HasNextPage: false, HasPreviousPage: false},
			TotalCount: 0,
		}
	}
	edges := make([]*TaskEdge, 0, len(list.Tasks))
	for i, task := range list.Tasks {
		cursor := encodeCursor(i + 1 + (list.Page-1)*list.PageSize)
		edges = append(edges, &TaskEdge{
			Node:   taskToGraphQL(task, nil, nil),
			Cursor: cursor,
		})
	}
	return &TaskConnection{
		Edges:      edges,
		PageInfo:   pageInfoFromPagination(list.Page, list.PageSize, list.TotalPages, int(list.Total)),
		TotalCount: int(list.Total),
	}
}

func userConnectionToGraphQL(list *dto.UserListOutput) *UserConnection {
	if list == nil {
		return &UserConnection{
			Edges:      []*UserEdge{},
			PageInfo:   &PageInfo{HasNextPage: false, HasPreviousPage: false},
			TotalCount: 0,
		}
	}
	edges := make([]*UserEdge, 0, len(list.Users))
	for i, user := range list.Users {
		cursor := encodeCursor(i + 1 + (list.Page-1)*list.PageSize)
		edges = append(edges, &UserEdge{
			Node:   userToGraphQL(user),
			Cursor: cursor,
		})
	}
	return &UserConnection{
		Edges:      edges,
		PageInfo:   pageInfoFromPagination(list.Page, list.PageSize, list.TotalPages, int(list.Total)),
		TotalCount: int(list.Total),
	}
}

func pageInfoFromPagination(page, pageSize, totalPages, totalCount int) *PageInfo {
	hasNext := page < totalPages
	hasPrev := page > 1
	var startCursor, endCursor *string
	if totalCount > 0 {
		sc := encodeCursor((page-1)*pageSize + 1)
		startCursor = &sc
		ec := encodeCursor((page-1)*pageSize + totalCount)
		endCursor = &ec
	}
	return &PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
}

func encodeCursor(i int) string {
	return fmt.Sprintf("cursor_%s", strconv.Itoa(i))
}
