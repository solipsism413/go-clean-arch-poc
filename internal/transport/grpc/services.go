package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/transport/grpc/pb"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterApplicationServices(
	server gogrpc.ServiceRegistrar,
	taskService input.TaskService,
	userService input.UserService,
	authService input.AuthService,
	labelService input.LabelService,
) {
	pb.RegisterTaskServiceServer(server, &TaskServiceHandler{taskService: taskService, authService: authService})
	pb.RegisterUserServiceServer(server, &UserServiceHandler{userService: userService, authService: authService})
	pb.RegisterAuthServiceServer(server, &AuthServiceHandler{authService: authService})
	pb.RegisterLabelServiceServer(server, &LabelServiceHandler{labelService: labelService, authService: authService})
}

type TaskServiceHandler struct {
	pb.UnimplementedTaskServiceServer
	taskService input.TaskService
	authService input.AuthService
}

func (h *TaskServiceHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	input := dto.CreateTaskInput{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Priority:    priorityFromPB(req.GetPriority()),
	}
	if req.GetDueDate() != nil {
		dueDate := req.GetDueDate().AsTime()
		input.DueDate = &dueDate
	}
	if req.GetAssigneeId() != "" {
		assigneeID, err := parseUUID(req.GetAssigneeId(), "assignee_id")
		if err != nil {
			return nil, err
		}
		input.AssigneeID = &assigneeID
	}
	for _, rawID := range req.GetLabelIds() {
		labelID, err := parseUUID(rawID, "label_ids")
		if err != nil {
			return nil, err
		}
		input.LabelIDs = append(input.LabelIDs, labelID)
	}

	task, err := h.taskService.CreateTask(ctx, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.GetTask(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	filter, err := taskFilterFromPB(req.GetFilter())
	if err != nil {
		return nil, err
	}

	tasks, err := h.taskService.ListTasks(ctx, filter, paginationFromPB(req.GetPagination()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*pb.Task, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		items = append(items, taskToPB(task))
	}

	return &pb.ListTasksResponse{
		Tasks:    items,
		Total:    tasks.Total,
		Page:     int32(tasks.Page),
		PageSize: int32(tasks.PageSize),
	}, nil
}

func (h *TaskServiceHandler) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	input := dto.UpdateTaskInput{}
	if req.Title != nil {
		title := req.GetTitle()
		input.Title = &title
	}
	if req.Description != nil {
		description := req.GetDescription()
		input.Description = &description
	}
	if req.Priority != nil {
		priority := priorityFromPB(req.GetPriority())
		input.Priority = &priority
	}
	if req.GetDueDate() != nil {
		dueDate := req.GetDueDate().AsTime()
		input.DueDate = &dueDate
	}

	task, err := h.taskService.UpdateTask(ctx, id, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*emptypb.Empty, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	if err := h.taskService.DeleteTask(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *TaskServiceHandler) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	taskID, err := parseUUID(req.GetTaskId(), "task_id")
	if err != nil {
		return nil, err
	}
	assigneeID, err := parseUUID(req.GetAssigneeId(), "assignee_id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.AssignTask(ctx, taskID, assigneeID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) ChangeTaskStatus(ctx context.Context, req *pb.ChangeTaskStatusRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	taskID, err := parseUUID(req.GetTaskId(), "task_id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.ChangeTaskStatus(ctx, taskID, taskStatusFromPB(req.GetStatus()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) UnassignTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.UnassignTask(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) CompleteTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.CompleteTask(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) ArchiveTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.ArchiveTask(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) AddLabel(ctx context.Context, req *pb.AddLabelRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	taskID, err := parseUUID(req.GetTaskId(), "task_id")
	if err != nil {
		return nil, err
	}
	labelID, err := parseUUID(req.GetLabelId(), "label_id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.AddLabel(ctx, taskID, labelID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) RemoveLabel(ctx context.Context, req *pb.RemoveLabelRequest) (*pb.Task, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	taskID, err := parseUUID(req.GetTaskId(), "task_id")
	if err != nil {
		return nil, err
	}
	labelID, err := parseUUID(req.GetLabelId(), "label_id")
	if err != nil {
		return nil, err
	}

	task, err := h.taskService.RemoveLabel(ctx, taskID, labelID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return taskToPB(task), nil
}

func (h *TaskServiceHandler) SearchTasks(ctx context.Context, req *pb.SearchTasksRequest) (*pb.ListTasksResponse, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	tasks, err := h.taskService.SearchTasks(ctx, req.GetQuery(), paginationFromPB(req.GetPagination()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*pb.Task, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		items = append(items, taskToPB(task))
	}

	return &pb.ListTasksResponse{
		Tasks:    items,
		Total:    tasks.Total,
		Page:     int32(tasks.Page),
		PageSize: int32(tasks.PageSize),
	}, nil
}

func (h *TaskServiceHandler) GetOverdueTasks(ctx context.Context, req *pb.ListOverdueTasksRequest) (*pb.ListTasksResponse, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	tasks, err := h.taskService.GetOverdueTasks(ctx, paginationFromPB(req.GetPagination()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*pb.Task, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		items = append(items, taskToPB(task))
	}

	return &pb.ListTasksResponse{
		Tasks:    items,
		Total:    tasks.Total,
		Page:     int32(tasks.Page),
		PageSize: int32(tasks.PageSize),
	}, nil
}

func (h *TaskServiceHandler) StreamTaskUpdates(*pb.StreamTaskUpdatesRequest, pb.TaskService_StreamTaskUpdatesServer) error {
	return status.Error(codes.Unimplemented, "stream_task_updates is not implemented")
}

type UserServiceHandler struct {
	pb.UnimplementedUserServiceServer
	userService input.UserService
	authService input.AuthService
}

func (h *UserServiceHandler) GetMe(ctx context.Context, _ *emptypb.Empty) (*pb.User, error) {
	ctx, claims, err := contextWithRequiredClaims(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	user, err := h.userService.GetUser(ctx, claims.UserID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return userToPB(user), nil
}

func (h *UserServiceHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	user, err := h.userService.GetUser(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return userToPB(user), nil
}

func (h *UserServiceHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	filter, err := userFilterFromPB(req.GetFilter())
	if err != nil {
		return nil, err
	}

	users, err := h.userService.ListUsers(ctx, filter, paginationFromPB(req.GetPagination()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*pb.User, 0, len(users.Users))
	for _, user := range users.Users {
		items = append(items, userToPB(user))
	}

	return &pb.ListUsersResponse{
		Users:    items,
		Total:    users.Total,
		Page:     int32(users.Page),
		PageSize: int32(users.PageSize),
	}, nil
}

func (h *UserServiceHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	input := dto.UpdateUserInput{}
	if req.Email != nil {
		email := req.GetEmail()
		input.Email = &email
	}
	if req.Name != nil {
		name := req.GetName()
		input.Name = &name
	}

	user, err := h.userService.UpdateUser(ctx, id, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return userToPB(user), nil
}

func (h *UserServiceHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	if err := h.userService.DeleteUser(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *UserServiceHandler) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.User, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	userID, err := parseUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}
	roleID, err := parseUUID(req.GetRoleId(), "role_id")
	if err != nil {
		return nil, err
	}

	user, err := h.userService.AssignRole(ctx, userID, roleID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return userToPB(user), nil
}

func (h *UserServiceHandler) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.User, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	userID, err := parseUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}
	roleID, err := parseUUID(req.GetRoleId(), "role_id")
	if err != nil {
		return nil, err
	}

	user, err := h.userService.RemoveRole(ctx, userID, roleID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return userToPB(user), nil
}

type AuthServiceHandler struct {
	pb.UnimplementedAuthServiceServer
	authService input.AuthService
}

func (h *AuthServiceHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	authOutput, err := h.authService.Login(ctx, dto.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return authToPB(authOutput), nil
}

func (h *AuthServiceHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	authOutput, err := h.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return authToPB(authOutput), nil
}

func (h *AuthServiceHandler) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	_, claims, err := contextWithRequiredClaims(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	if err := h.authService.Logout(ctx, claims.UserID); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *AuthServiceHandler) Register(ctx context.Context, req *pb.CreateUserRequest) (*pb.AuthResponse, error) {
	authOutput, err := h.authService.Register(ctx, dto.CreateUserInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		Name:     req.GetName(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return authToPB(authOutput), nil
}

func (h *AuthServiceHandler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*emptypb.Empty, error) {
	_, claims, err := contextWithRequiredClaims(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	if err := h.authService.ChangePassword(ctx, claims.UserID, dto.ChangePasswordInput{
		OldPassword: req.GetOldPassword(),
		NewPassword: req.GetNewPassword(),
	}); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

type LabelServiceHandler struct {
	pb.UnimplementedLabelServiceServer
	labelService input.LabelService
	authService  input.AuthService
}

func (h *LabelServiceHandler) CreateLabel(ctx context.Context, req *pb.CreateLabelRequest) (*pb.Label, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	label, err := h.labelService.CreateLabel(ctx, dto.CreateLabelInput{
		Name:  req.GetName(),
		Color: req.GetColor(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return labelToPB(label), nil
}

func (h *LabelServiceHandler) GetLabel(ctx context.Context, req *pb.GetLabelRequest) (*pb.Label, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	label, err := h.labelService.GetLabel(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return labelToPB(label), nil
}

func (h *LabelServiceHandler) ListLabels(ctx context.Context, _ *emptypb.Empty) (*pb.ListLabelsResponse, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	labels, err := h.labelService.ListLabels(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*pb.Label, 0, len(labels))
	for _, label := range labels {
		items = append(items, labelToPB(label))
	}

	return &pb.ListLabelsResponse{Labels: items}, nil
}

func (h *LabelServiceHandler) UpdateLabel(ctx context.Context, req *pb.UpdateLabelRequest) (*pb.Label, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	input := dto.UpdateLabelInput{}
	if req.Name != nil {
		name := req.GetName()
		input.Name = &name
	}
	if req.Color != nil {
		color := req.GetColor()
		input.Color = &color
	}

	label, err := h.labelService.UpdateLabel(ctx, id, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return labelToPB(label), nil
}

func (h *LabelServiceHandler) DeleteLabel(ctx context.Context, req *pb.DeleteLabelRequest) (*emptypb.Empty, error) {
	ctx, err := requireAuth(ctx, h.authService)
	if err != nil {
		return nil, err
	}

	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}

	if err := h.labelService.DeleteLabel(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func requireAuth(ctx context.Context, authService input.AuthService) (context.Context, error) {
	authorization, ok := authorizationFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	claims, err := validateAuthorization(ctx, authService, authorization)
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, auth.ClaimsContextKey, claims), nil
}

func contextWithRequiredClaims(ctx context.Context, authService input.AuthService) (context.Context, *dto.TokenClaims, error) {
	authorization, ok := authorizationFromMetadata(ctx)
	if !ok {
		return nil, nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	claims, err := validateAuthorization(ctx, authService, authorization)
	if err != nil {
		return nil, nil, err
	}

	ctx = context.WithValue(ctx, auth.ClaimsContextKey, claims)
	return ctx, claims, nil
}

func authorizationFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get("authorization")
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return "", false
	}
	return values[0], true
}

func validateAuthorization(ctx context.Context, authService input.AuthService, authorization string) (*dto.TokenClaims, error) {
	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization metadata format")
	}

	claims, err := authService.ValidateToken(ctx, parts[1])
	if err != nil {
		return nil, toGRPCError(err)
	}

	return claims, nil
}

func parseUUID(raw string, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s", field)
	}
	return id, nil
}

func taskFilterFromPB(filter *pb.TaskFilter) (dto.TaskFilter, error) {
	if filter == nil {
		return dto.TaskFilter{}, nil
	}

	result := dto.TaskFilter{}
	if filter.Status != nil {
		statusValue := taskStatusFromPB(filter.GetStatus())
		result.Status = &statusValue
	}
	if filter.Priority != nil {
		priorityValue := priorityFromPB(filter.GetPriority())
		result.Priority = &priorityValue
	}
	if filter.AssigneeId != nil {
		assigneeID, err := parseUUID(filter.GetAssigneeId(), "filter.assignee_id")
		if err != nil {
			return dto.TaskFilter{}, err
		}
		result.AssigneeID = &assigneeID
	}
	if filter.CreatorId != nil {
		creatorID, err := parseUUID(filter.GetCreatorId(), "filter.creator_id")
		if err != nil {
			return dto.TaskFilter{}, err
		}
		result.CreatorID = &creatorID
	}
	if filter.Search != nil {
		result.Search = filter.GetSearch()
	}

	return result, nil
}

func userFilterFromPB(filter *pb.UserFilter) (dto.UserFilter, error) {
	if filter == nil {
		return dto.UserFilter{}, nil
	}

	result := dto.UserFilter{}
	if filter.Search != nil {
		result.Search = filter.GetSearch()
	}
	if filter.RoleId != nil {
		roleID, err := parseUUID(filter.GetRoleId(), "filter.role_id")
		if err != nil {
			return dto.UserFilter{}, err
		}
		result.RoleID = &roleID
	}

	return result, nil
}

func paginationFromPB(p *pb.Pagination) dto.Pagination {
	pagination := dto.DefaultPagination()
	if p == nil {
		return pagination
	}
	if p.GetPage() > 0 {
		pagination.Page = int(p.GetPage())
	}
	if p.GetPageSize() > 0 {
		pagination.PageSize = int(p.GetPageSize())
	}
	if p.GetSortBy() != "" {
		pagination.SortBy = p.GetSortBy()
	}
	if p.GetSortDesc() {
		pagination.SortDesc = true
	}
	return pagination
}

func taskToPB(task *dto.TaskOutput) *pb.Task {
	if task == nil {
		return nil
	}

	labels := make([]*pb.Label, 0, len(task.Labels))
	for _, label := range task.Labels {
		labels = append(labels, &pb.Label{
			Id:    label.ID.String(),
			Name:  label.Name,
			Color: label.Color,
		})
	}

	result := &pb.Task{
		Id:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		Status:      taskStatusToPB(task.Status),
		Priority:    priorityToPB(task.Priority),
		CreatorId:   task.CreatorID.String(),
		Labels:      labels,
		CreatedAt:   timestamppb.New(task.CreatedAt),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
	}
	if task.DueDate != nil {
		result.DueDate = timestamppb.New(*task.DueDate)
	}
	if task.AssigneeID != nil {
		result.AssigneeId = task.AssigneeID.String()
	}
	return result
}

func userToPB(user *dto.UserOutput) *pb.User {
	if user == nil {
		return nil
	}

	roles := make([]*pb.Role, 0, len(user.Roles))
	for _, role := range user.Roles {
		permissions := make([]*pb.Permission, 0, len(role.Permissions))
		for _, permission := range role.Permissions {
			permissions = append(permissions, &pb.Permission{
				Id:       permission.ID.String(),
				Name:     permission.Name,
				Resource: permission.Resource,
				Action:   permission.Action,
			})
		}

		roles = append(roles, &pb.Role{
			Id:          role.ID.String(),
			Name:        role.Name,
			Description: role.Description,
			Permissions: permissions,
		})
	}

	return &pb.User{
		Id:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Roles:     roles,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func labelToPB(label *dto.LabelOutput) *pb.Label {
	if label == nil {
		return nil
	}

	return &pb.Label{
		Id:    label.ID.String(),
		Name:  label.Name,
		Color: label.Color,
	}
}

func authToPB(authOutput *dto.AuthOutput) *pb.AuthResponse {
	if authOutput == nil {
		return nil
	}

	return &pb.AuthResponse{
		AccessToken:  authOutput.AccessToken,
		RefreshToken: authOutput.RefreshToken,
		ExpiresAt:    timestamppb.New(authOutput.ExpiresAt),
		User:         userToPB(authOutput.User),
	}
}

func taskStatusFromPB(value pb.TaskStatus) string {
	switch value {
	case pb.TaskStatus_TASK_STATUS_TODO:
		return "TODO"
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return "IN_PROGRESS"
	case pb.TaskStatus_TASK_STATUS_IN_REVIEW:
		return "IN_REVIEW"
	case pb.TaskStatus_TASK_STATUS_DONE:
		return "DONE"
	case pb.TaskStatus_TASK_STATUS_ARCHIVED:
		return "ARCHIVED"
	default:
		return ""
	}
}

func taskStatusToPB(value string) pb.TaskStatus {
	switch value {
	case "TODO":
		return pb.TaskStatus_TASK_STATUS_TODO
	case "IN_PROGRESS":
		return pb.TaskStatus_TASK_STATUS_IN_PROGRESS
	case "IN_REVIEW":
		return pb.TaskStatus_TASK_STATUS_IN_REVIEW
	case "DONE":
		return pb.TaskStatus_TASK_STATUS_DONE
	case "ARCHIVED":
		return pb.TaskStatus_TASK_STATUS_ARCHIVED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func priorityFromPB(value pb.Priority) string {
	switch value {
	case pb.Priority_PRIORITY_LOW:
		return "LOW"
	case pb.Priority_PRIORITY_MEDIUM:
		return "MEDIUM"
	case pb.Priority_PRIORITY_HIGH:
		return "HIGH"
	case pb.Priority_PRIORITY_URGENT:
		return "URGENT"
	default:
		return ""
	}
}

func priorityToPB(value string) pb.Priority {
	switch value {
	case "LOW":
		return pb.Priority_PRIORITY_LOW
	case "MEDIUM":
		return pb.Priority_PRIORITY_MEDIUM
	case "HIGH":
		return pb.Priority_PRIORITY_HIGH
	case "URGENT":
		return pb.Priority_PRIORITY_URGENT
	default:
		return pb.Priority_PRIORITY_UNSPECIFIED
	}
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		return status.Error(codes.InvalidArgument, validationErr.Error())
	}
	if domainerror.IsNotFoundError(err) {
		return status.Error(codes.NotFound, err.Error())
	}
	if domainerror.IsValidationError(err) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if domainerror.IsConflictError(err) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if domainerror.IsUnauthorizedError(err) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if domainerror.IsForbiddenError(err) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
