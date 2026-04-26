// Package auth contains authentication use cases.
package auth

import (
	"context"
	"log/slog"
	"time"

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

const cachePrefix = "app"

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
func (uc *AuthUseCase) Logout(ctx context.Context, userID uuid.UUID, accessToken string) error {
	// Extract token ID from the raw access token
	tokenID, err := uc.tokenService.ExtractTokenID(accessToken)
	if err != nil {
		return err
	}

	if uc.cache != nil {
		// Validate token to get claims and extract refresh token JTI from session
		claims, err := uc.tokenService.ValidateToken(ctx, accessToken)
		if err == nil && claims != nil {
			remainingTTL := time.Until(claims.ExpiresAt)
			if remainingTTL > 0 {
				// Blacklist access token
				blacklistKey := output.NewCacheKeyBuilder(cachePrefix).TokenBlacklist(tokenID)
				_ = uc.cache.Set(ctx, blacklistKey, []byte("revoked"), remainingTTL)
			}
		}

		// Find session and blacklist refresh token too
		sessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(userID.String(), tokenID)
		var sessionData map[string]any
		if err := uc.cache.GetJSON(ctx, sessionKey, &sessionData); err == nil {
			if refreshTokenID, ok := sessionData["refreshTokenId"].(string); ok && refreshTokenID != "" {
				refreshBlacklistKey := output.NewCacheKeyBuilder(cachePrefix).TokenBlacklist(refreshTokenID)
				_ = uc.cache.Set(ctx, refreshBlacklistKey, []byte("revoked"), uc.tokenService.RefreshTokenTTL())
				// Delete refresh token session too
				refreshSessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(userID.String(), refreshTokenID)
				_ = uc.cache.Delete(ctx, refreshSessionKey)
			}
		}

		// Delete the access token session
		_ = uc.cache.Delete(ctx, sessionKey)
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

	// Check if refresh token has been revoked or its session invalidated
	if uc.cache != nil {
		tokenID, err := uc.tokenService.ExtractTokenID(refreshToken)
		if err == nil && tokenID != "" {
			blacklistKey := output.NewCacheKeyBuilder(cachePrefix).TokenBlacklist(tokenID)
			revoked, _ := uc.cache.Exists(ctx, blacklistKey)
			if revoked {
				return nil, domainerror.ErrInvalidToken
			}

			// Verify the session still exists (handles bulk invalidation)
			sessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(userID.String(), tokenID)
			hasSession, _ := uc.cache.Exists(ctx, sessionKey)
			if !hasSession {
				return nil, domainerror.ErrInvalidToken
			}
		}
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

	// Store sessions in Redis for invalidation support
	if uc.cache != nil {
		accessTokenID, _ := uc.tokenService.ExtractTokenID(authOutput.AccessToken)
		refreshTokenID, _ := uc.tokenService.ExtractTokenID(authOutput.RefreshToken)
		if accessTokenID != "" {
			sessionData := map[string]any{
				"userId":         user.ID.String(),
				"accessTokenId":  accessTokenID,
				"refreshTokenId": refreshTokenID,
				"createdAt":      time.Now().UTC(),
				"expiresAt":      authOutput.ExpiresAt,
			}
			// Store session keyed by access token JTI
			accessSessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(user.ID.String(), accessTokenID)
			_ = uc.cache.SetJSON(ctx, accessSessionKey, sessionData, uc.tokenService.RefreshTokenTTL())
			// Also store session keyed by refresh token JTI so RefreshToken can validate it
			if refreshTokenID != "" {
				refreshSessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(user.ID.String(), refreshTokenID)
				_ = uc.cache.SetJSON(ctx, refreshSessionKey, sessionData, uc.tokenService.RefreshTokenTTL())
			}
		}
	}

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

	// Invalidate all sessions so user must re-login everywhere
	uc.invalidateAllUserSessions(ctx, userID)

	// Publish event
	evt := event.NewUserPasswordChanged(userID)
	if err := uc.eventPublisher.Publish(ctx, output.TopicUserEvents, evt); err != nil {
		uc.logger.Error("failed to publish password changed event", "userId", userID, "error", err)
	}

	uc.logger.Info("password changed", "userId", userID)

	return nil
}

// ValidateToken validates an access token and checks the revocation list.
func (uc *AuthUseCase) ValidateToken(ctx context.Context, token string) (*dto.TokenClaims, error) {
	claims, err := uc.tokenService.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check token blacklist and session in Redis
	if uc.cache != nil && claims.TokenID != "" {
		blacklistKey := output.NewCacheKeyBuilder(cachePrefix).TokenBlacklist(claims.TokenID)
		revoked, _ := uc.cache.Exists(ctx, blacklistKey)
		if revoked {
			return nil, domainerror.ErrInvalidToken
		}

		// Also verify the session still exists (handles bulk invalidation)
		sessionKey := output.NewCacheKeyBuilder(cachePrefix).UserSession(claims.UserID.String(), claims.TokenID)
		hasSession, _ := uc.cache.Exists(ctx, sessionKey)
		if !hasSession {
			return nil, domainerror.ErrInvalidToken
		}
	}

	return claims, nil
}

// invalidateAllUserSessions revokes all active tokens for a user.
func (uc *AuthUseCase) invalidateAllUserSessions(ctx context.Context, userID uuid.UUID) {
	if uc.cache == nil {
		return
	}

	// Delete all session keys for this user
	sessionPattern := output.NewCacheKeyBuilder(cachePrefix).UserSessions(userID.String())
	_ = uc.cache.DeletePattern(ctx, sessionPattern)

	uc.logger.Info("invalidated all user sessions", "userId", userID)
}

// InvalidateAllUserSessions is a public hook for external use cases (e.g., role changes).
func (uc *AuthUseCase) InvalidateAllUserSessions(ctx context.Context, userID uuid.UUID) {
	uc.invalidateAllUserSessions(ctx, userID)
}
