package graphql

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMutationResolver_TaskLifecycleParity(t *testing.T) {
	taskService := mocks.NewMockTaskService(t)
	userService := mocks.NewMockUserService(t)

	resolver := &mutationResolver{&Resolver{taskService: taskService, userService: userService}}
	ctx := authenticatedGraphQLContext()
	creator := testGraphQLUserOutput()

	tests := []struct {
		name   string
		status string
		call   func(context.Context, uuid.UUID) (*Task, error)
		setup  func(uuid.UUID, *dto.TaskOutput)
	}{
		{
			name:   "complete task",
			status: "DONE",
			call:   resolver.CompleteTask,
			setup: func(taskID uuid.UUID, task *dto.TaskOutput) {
				taskService.EXPECT().CompleteTask(mock.Anything, taskID).Return(task, nil).Once()
			},
		},
		{
			name:   "archive task",
			status: "ARCHIVED",
			call:   resolver.ArchiveTask,
			setup: func(taskID uuid.UUID, task *dto.TaskOutput) {
				taskService.EXPECT().ArchiveTask(mock.Anything, taskID).Return(task, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskID := uuid.New()
			task := testGraphQLTaskOutput(taskID, creator.ID, tt.status)
			tt.setup(taskID, task)
			userService.EXPECT().GetUser(mock.Anything, creator.ID).Return(creator, nil).Once()

			result, err := tt.call(ctx, taskID)
			require.NoError(t, err)
			require.Equal(t, taskID, result.ID)
			require.Equal(t, TaskStatus(tt.status), result.Status)
			require.Equal(t, creator.ID, result.Creator.ID)
		})
	}
}

func TestQueryResolver_OverdueTasks(t *testing.T) {
	taskService := mocks.NewMockTaskService(t)
	userService := mocks.NewMockUserService(t)

	resolver := &queryResolver{&Resolver{taskService: taskService, userService: userService}}
	ctx := authenticatedGraphQLContext()
	creator := testGraphQLUserOutput()
	task := testGraphQLTaskOutput(uuid.New(), creator.ID, "TODO")
	expectedPagination := dto.Pagination{Page: 2, PageSize: 5, SortBy: "createdAt", SortDesc: true}
	page := 2
	pageSize := 5
	pagination := &PaginationInput{Page: &page, PageSize: &pageSize}

	taskService.EXPECT().GetOverdueTasks(mock.Anything, expectedPagination).Return(&dto.TaskListOutput{
		Tasks:      []*dto.TaskOutput{task},
		Total:      1,
		Page:       2,
		PageSize:   5,
		TotalPages: 3,
	}, nil).Once()
	userService.EXPECT().GetUser(mock.Anything, creator.ID).Return(creator, nil).Once()

	result, err := resolver.OverdueTasks(ctx, pagination)
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Edges, 1)
	require.Equal(t, task.ID, result.Edges[0].Node.ID)
	require.True(t, result.PageInfo.HasNextPage)
	require.True(t, result.PageInfo.HasPreviousPage)
}

func TestQueryResolver_Label(t *testing.T) {
	labelService := mocks.NewMockLabelService(t)
	resolver := &queryResolver{&Resolver{labelService: labelService}}
	ctx := authenticatedGraphQLContext()
	labelID := uuid.New()
	createdAt := time.Now().UTC()

	labelService.EXPECT().GetLabel(mock.Anything, labelID).Return(&dto.LabelOutput{
		ID:        labelID,
		Name:      "backend",
		Color:     "#2563eb",
		CreatedAt: createdAt,
	}, nil).Once()

	result, err := resolver.Label(ctx, labelID)
	require.NoError(t, err)
	require.Equal(t, labelID, result.ID)
	require.Equal(t, "backend", result.Name)
	require.Equal(t, "#2563eb", result.Color)
}

func authenticatedGraphQLContext() context.Context {
	return context.WithValue(context.Background(), auth.ClaimsContextKey, &dto.TokenClaims{
		UserID: uuid.New(),
		Roles:  []string{"admin"},
	})
}

func testGraphQLUserOutput() *dto.UserOutput {
	now := time.Now().UTC()
	return &dto.UserOutput{
		ID:        uuid.New(),
		Email:     "creator@example.com",
		Name:      "Creator",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testGraphQLTaskOutput(taskID, creatorID uuid.UUID, status string) *dto.TaskOutput {
	now := time.Now().UTC()
	return &dto.TaskOutput{
		ID:        taskID,
		Title:     "Task",
		Status:    status,
		Priority:  "MEDIUM",
		CreatorID: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
