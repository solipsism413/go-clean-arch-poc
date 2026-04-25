// Package auth contains authentication use cases.
package auth

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
)

const defaultRegistrationRole = entity.RoleMember

// Ensure AuthUseCase implements input.AuthService.
var _ input.AuthService = (*AuthUseCase)(nil)

// AuthUseCase implements authentication use cases.
type AuthUseCase struct {
	userRepo       output.UserRepository
	roleRepo       output.RoleRepository
	cache          output.CacheRepository
	eventPublisher output.EventPublisher
	tm             output.TransactionManager
	tokenService   *jwt.TokenService
	validator      validation.Validator
	logger         *slog.Logger
}

// NewAuthUseCase creates a new AuthUseCase.
func NewAuthUseCase(
	userRepo output.UserRepository,
	roleRepo output.RoleRepository,
	cache output.CacheRepository,
	eventPublisher output.EventPublisher,
	tm output.TransactionManager,
	tokenService *jwt.TokenService,
	validator validation.Validator,
	logger *slog.Logger,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		cache:          cache,
		eventPublisher: eventPublisher,
		tm:             tm,
		tokenService:   tokenService,
		validator:      validator,
		logger:         logger,
	}
}

// Register creates a new user account and returns tokens.
func (uc *AuthUseCase) Register(ctx context.Context, input dto.CreateUserInput) (*dto.AuthOutput, error) {
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	exists, err := uc.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domainerror.ErrEmailAlreadyExists
	}

	role, err := uc.roleRepo.FindByName(ctx, defaultRegistrationRole)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domainerror.ErrRoleNotFound
	}

	user, err := entity.NewUser(input.Email, input.Password, input.Name)
	if err != nil {
		return nil, err
	}
	user.AssignRole(*role)

	if err := uc.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	authOutput, err := uc.generateAuthOutput(ctx, user)
	if err != nil {
		return nil, err
	}

	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, event.NewUserCreated(user.ID, user.Email, user.Name)); err != nil {
		uc.logger.Error("failed to publish user created event", "userId", user.ID, "error", err)
	}

	uc.logger.Info("user registered", "userId", user.ID, "email", user.Email)

	return authOutput, nil
}

// Login authenticates a user and returns tokens.
func (uc *AuthUseCase) Login(ctx context.Context, input dto.LoginInput) (*dto.AuthOutput, error) {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return nil, err
	}

	// Find user by email
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrInvalidCredentials
	}

	// Verify password
	if !user.VerifyPassword(input.Password) {
		return nil, domainerror.ErrInvalidCredentials
	}

	authOutput, err := uc.generateAuthOutput(ctx, user)
	if err != nil {
		return nil, err
	}

	// Publish login event
	evt := event.NewUserLoggedIn(user.ID, "", "") // IP and User-Agent can be extracted from context
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user logged in event", "userId", user.ID, "error", err)
	}

	uc.logger.Info("user logged in", "userId", user.ID, "email", user.Email)

	return authOutput, nil
}

// Logout invalidates the user's session/tokens.
func (uc *AuthUseCase) Logout(ctx context.Context, userID uuid.UUID) error {
	// In a stateless JWT setup, we can use a blacklist in Redis
	// For now, just publish the event
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerror.ErrUserNotFound
	}

	// Publish logout event
	evt := event.NewUserLoggedOut(userID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish user logged out event", "userId", userID, "error", err)
	}

	uc.logger.Info("user logged out", "userId", userID)

	return nil
}

// RefreshToken refreshes the access token.
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthOutput, error) {
	// Validate refresh token
	userID, err := uc.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, domainerror.ErrInvalidToken
	}

	// Get user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domainerror.ErrUserNotFound
	}

	authOutput, err := uc.generateAuthOutput(ctx, user)
	if err != nil {
		return nil, err
	}

	uc.logger.Debug("token refreshed", "userId", userID)

	return authOutput, nil
}

func (uc *AuthUseCase) generateAuthOutput(ctx context.Context, user *entity.User) (*dto.AuthOutput, error) {
	roles := make([]string, 0, len(user.Roles))
	roleIDs := make([]uuid.UUID, 0, len(user.Roles))
	permissions := make([]string, 0)
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
		roleIDs = append(roleIDs, role.ID)
		for _, perm := range role.Permissions {
			permissions = append(permissions, string(perm.Resource)+":"+string(perm.Action))
		}
	}

	authOutput, err := uc.tokenService.GenerateTokenPair(ctx, user.ID, user.Email, roles, roleIDs, permissions)
	if err != nil {
		return nil, err
	}

	authOutput.User = dto.UserFromEntity(user)

	return authOutput, nil
}

// ChangePassword changes the user's password.
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordInput) error {
	// Validate input
	if err := uc.validator.Validate(input); err != nil {
		return err
	}

	// Get user
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerror.ErrUserNotFound
	}

	// Verify old password
	if !user.VerifyPassword(input.OldPassword) {
		return domainerror.ErrInvalidCredentials
	}

	// Update password
	if err := user.UpdatePassword(input.NewPassword); err != nil {
		return err
	}

	// Save user
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Publish event
	evt := event.NewUserPasswordChanged(userID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish password changed event", "userId", userID, "error", err)
	}

	uc.logger.Info("password changed", "userId", userID)

	return nil
}

// ValidateToken validates an access token.
func (uc *AuthUseCase) ValidateToken(ctx context.Context, token string) (*dto.TokenClaims, error) {
	return uc.tokenService.ValidateToken(ctx, token)
}
