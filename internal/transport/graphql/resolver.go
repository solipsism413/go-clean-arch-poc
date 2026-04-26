package graphql

import (
	"context"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
	"github.com/handiism/go-clean-arch-poc/internal/auth/rbac"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
)

// Resolver serves as dependency injection for the GraphQL resolvers.
type Resolver struct {
	taskService   input.TaskService
	userService   input.UserService
	authService   input.AuthService
	labelService  input.LabelService
	roleService   input.RoleService
	authorizer    *rbac.Authorizer
	subscriptions *SubscriptionBroker
}

// NewResolver creates a new Resolver with the given services.
func NewResolver(
	taskService input.TaskService,
	userService input.UserService,
	authService input.AuthService,
	labelService input.LabelService,
	roleService input.RoleService,
	authorizer *rbac.Authorizer,
	subscriptions *SubscriptionBroker,
) *Resolver {
	return &Resolver{
		taskService:   taskService,
		userService:   userService,
		authService:   authService,
		labelService:  labelService,
		roleService:   roleService,
		authorizer:    authorizer,
		subscriptions: subscriptions,
	}
}

// requireAuth extracts claims from context and returns an error if missing.
func requireAuth(ctx context.Context) (*dto.TokenClaims, error) {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, domainerror.ErrUnauthorized
	}
	return claims, nil
}

// enrichAndMapTask fetches assignee/creator users and maps a TaskOutput to GraphQL Task.
func (r *Resolver) enrichAndMapTask(ctx context.Context, task *dto.TaskOutput) (*Task, error) {
	if task == nil {
		return nil, nil
	}
	var assignee *dto.UserOutput
	var creator *dto.UserOutput
	var err error

	if task.AssigneeID != nil {
		assignee, err = r.userService.GetUser(ctx, *task.AssigneeID)
		if err != nil && !domainerror.IsNotFoundError(err) {
			return nil, err
		}
	}

	creator, err = r.userService.GetUser(ctx, task.CreatorID)
	if err != nil && !domainerror.IsNotFoundError(err) {
		return nil, err
	}

	return taskToGraphQL(task, assignee, creator), nil
}

// requireAnyRole checks if the authenticated user has any of the given roles.
func (r *Resolver) requireAnyRole(ctx context.Context, roles ...string) error {
	claims, err := requireAuth(ctx)
	if err != nil {
		return err
	}
	if !r.authorizer.HasAnyRoleFromClaims(claims.Roles, roles...) {
		return domainerror.ErrForbidden
	}
	return nil
}
