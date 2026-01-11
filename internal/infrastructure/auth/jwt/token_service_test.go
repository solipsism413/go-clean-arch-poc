// Package jwt_test contains tests for the JWT token service.
package jwt_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/auth/jwt"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestTokenService creates a TokenService for testing with default config.
func createTestTokenService() *jwt.TokenService {
	cfg := config.JWTConfig{
		SecretKey:            "test-secret-key-32-characters-long",
		AccessTokenDuration:  1 * time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	return jwt.NewTokenService(cfg)
}

// createTestTokenServiceWithConfig creates a TokenService with custom config.
func createTestTokenServiceWithConfig(cfg config.JWTConfig) *jwt.TokenService {
	return jwt.NewTokenService(cfg)
}

func TestNewTokenService(t *testing.T) {
	t.Run("should create token service with valid config", func(t *testing.T) {
		cfg := config.JWTConfig{
			SecretKey:            "my-secret-key",
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			Issuer:               "my-app",
		}

		ts := jwt.NewTokenService(cfg)

		assert.NotNil(t, ts)
	})
}

func TestTokenService_GenerateTokenPair(t *testing.T) {
	ctx := context.Background()

	t.Run("should generate valid token pair", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"
		roles := []string{"admin", "user"}
		roleIDs := []uuid.UUID{uuid.New(), uuid.New()}
		permissions := []string{"task:read", "task:write", "user:read"}

		result, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.False(t, result.ExpiresAt.IsZero())
		assert.True(t, result.ExpiresAt.After(time.Now()))
	})

	t.Run("should generate unique tokens for same user", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"
		roles := []string{"admin"}
		roleIDs := []uuid.UUID{uuid.New()}
		permissions := []string{"task:read"}

		result1, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		require.NoError(t, err)

		result2, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		require.NoError(t, err)

		// Tokens should be different due to different JTI
		assert.NotEqual(t, result1.AccessToken, result2.AccessToken)
		assert.NotEqual(t, result1.RefreshToken, result2.RefreshToken)
	})

	t.Run("should generate tokens with correct expiry time", func(t *testing.T) {
		accessDuration := 30 * time.Minute
		cfg := config.JWTConfig{
			SecretKey:            "test-secret-key-32-characters-long",
			AccessTokenDuration:  accessDuration,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts := createTestTokenServiceWithConfig(cfg)

		userID := uuid.New()
		email := "test@example.com"
		roles := []string{"user"}
		roleIDs := []uuid.UUID{uuid.New()}
		permissions := []string{}

		before := time.Now()
		result, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		after := time.Now()

		require.NoError(t, err)

		// ExpiresAt should be within expected range
		expectedMin := before.Add(accessDuration)
		expectedMax := after.Add(accessDuration)

		assert.True(t, result.ExpiresAt.After(expectedMin) || result.ExpiresAt.Equal(expectedMin))
		assert.True(t, result.ExpiresAt.Before(expectedMax) || result.ExpiresAt.Equal(expectedMax))
	})

	t.Run("should generate tokens with empty roles and permissions", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"
		roles := []string{}
		roleIDs := []uuid.UUID{}
		permissions := []string{}

		result, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
	})

	t.Run("should generate tokens with nil roles and permissions", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"

		result, err := ts.GenerateTokenPair(ctx, userID, email, nil, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
	})
}

func TestTokenService_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("should validate a valid access token", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"
		roles := []string{"admin", "user"}
		roleIDs := []uuid.UUID{uuid.New(), uuid.New()}
		permissions := []string{"task:read", "task:write"}

		tokenPair, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		require.NoError(t, err)

		claims, err := ts.ValidateToken(ctx, tokenPair.AccessToken)

		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, roles, claims.Roles)
		assert.Equal(t, roleIDs, claims.RoleIDs)
		assert.Equal(t, permissions, claims.Permissions)
		assert.False(t, claims.ExpiresAt.IsZero())
	})

	t.Run("should return error for invalid token", func(t *testing.T) {
		ts := createTestTokenService()

		claims, err := ts.ValidateToken(ctx, "invalid-token")

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for malformed token", func(t *testing.T) {
		ts := createTestTokenService()

		claims, err := ts.ValidateToken(ctx, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature")

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for token signed with different secret", func(t *testing.T) {
		// Create token with one secret
		cfg1 := config.JWTConfig{
			SecretKey:            "secret-key-one-32-characters-here",
			AccessTokenDuration:  1 * time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts1 := createTestTokenServiceWithConfig(cfg1)

		// Create another service with different secret
		cfg2 := config.JWTConfig{
			SecretKey:            "secret-key-two-32-characters-here",
			AccessTokenDuration:  1 * time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts2 := createTestTokenServiceWithConfig(cfg2)

		userID := uuid.New()
		tokenPair, err := ts1.GenerateTokenPair(ctx, userID, "test@example.com", nil, nil, nil)
		require.NoError(t, err)

		// Try to validate with different secret
		claims, err := ts2.ValidateToken(ctx, tokenPair.AccessToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for expired token", func(t *testing.T) {
		// Create token with very short expiry
		cfg := config.JWTConfig{
			SecretKey:            "test-secret-key-32-characters-long",
			AccessTokenDuration:  1 * time.Millisecond,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts := createTestTokenServiceWithConfig(cfg)

		userID := uuid.New()
		tokenPair, err := ts.GenerateTokenPair(ctx, userID, "test@example.com", nil, nil, nil)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		claims, err := ts.ValidateToken(ctx, tokenPair.AccessToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrExpiredToken)
	})

	t.Run("should return error for empty token", func(t *testing.T) {
		ts := createTestTokenService()

		claims, err := ts.ValidateToken(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})
}

func TestTokenService_ValidateRefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("should validate a valid refresh token", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "test@example.com"
		roles := []string{"admin"}
		roleIDs := []uuid.UUID{uuid.New()}
		permissions := []string{"task:read"}

		tokenPair, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		require.NoError(t, err)

		resultUserID, err := ts.ValidateRefreshToken(ctx, tokenPair.RefreshToken)

		require.NoError(t, err)
		assert.Equal(t, userID, resultUserID)
	})

	t.Run("should return error for invalid refresh token", func(t *testing.T) {
		ts := createTestTokenService()

		userID, err := ts.ValidateRefreshToken(ctx, "invalid-refresh-token")

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for malformed refresh token", func(t *testing.T) {
		ts := createTestTokenService()

		userID, err := ts.ValidateRefreshToken(ctx, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature")

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for refresh token signed with different secret", func(t *testing.T) {
		// Create token with one secret
		cfg1 := config.JWTConfig{
			SecretKey:            "secret-key-one-32-characters-here",
			AccessTokenDuration:  1 * time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts1 := createTestTokenServiceWithConfig(cfg1)

		// Create another service with different secret
		cfg2 := config.JWTConfig{
			SecretKey:            "secret-key-two-32-characters-here",
			AccessTokenDuration:  1 * time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:               "test-issuer",
		}
		ts2 := createTestTokenServiceWithConfig(cfg2)

		userID := uuid.New()
		tokenPair, err := ts1.GenerateTokenPair(ctx, userID, "test@example.com", nil, nil, nil)
		require.NoError(t, err)

		// Try to validate with different secret
		resultUserID, err := ts2.ValidateRefreshToken(ctx, tokenPair.RefreshToken)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, resultUserID)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})

	t.Run("should return error for expired refresh token", func(t *testing.T) {
		// Create token with very short expiry
		cfg := config.JWTConfig{
			SecretKey:            "test-secret-key-32-characters-long",
			AccessTokenDuration:  1 * time.Hour,
			RefreshTokenDuration: 1 * time.Millisecond,
			Issuer:               "test-issuer",
		}
		ts := createTestTokenServiceWithConfig(cfg)

		userID := uuid.New()
		tokenPair, err := ts.GenerateTokenPair(ctx, userID, "test@example.com", nil, nil, nil)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		resultUserID, err := ts.ValidateRefreshToken(ctx, tokenPair.RefreshToken)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, resultUserID)
		assert.ErrorIs(t, err, jwt.ErrExpiredToken)
	})

	t.Run("should return error for empty refresh token", func(t *testing.T) {
		ts := createTestTokenService()

		userID, err := ts.ValidateRefreshToken(ctx, "")

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
		assert.ErrorIs(t, err, jwt.ErrInvalidToken)
	})
}

func TestTokenService_RoundTrip(t *testing.T) {
	ctx := context.Background()

	t.Run("should successfully roundtrip access token with all claims", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "roundtrip@example.com"
		roles := []string{"admin", "manager", "user"}
		roleIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		permissions := []string{"task:read", "task:write", "task:delete", "user:read", "user:write"}

		// Generate token pair
		tokenPair, err := ts.GenerateTokenPair(ctx, userID, email, roles, roleIDs, permissions)
		require.NoError(t, err)

		// Validate access token and verify claims
		claims, err := ts.ValidateToken(ctx, tokenPair.AccessToken)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.ElementsMatch(t, roles, claims.Roles)
		assert.ElementsMatch(t, roleIDs, claims.RoleIDs)
		assert.ElementsMatch(t, permissions, claims.Permissions)
	})

	t.Run("should successfully roundtrip refresh token", func(t *testing.T) {
		ts := createTestTokenService()
		userID := uuid.New()
		email := "roundtrip@example.com"

		// Generate token pair
		tokenPair, err := ts.GenerateTokenPair(ctx, userID, email, nil, nil, nil)
		require.NoError(t, err)

		// Validate refresh token
		resultUserID, err := ts.ValidateRefreshToken(ctx, tokenPair.RefreshToken)
		require.NoError(t, err)

		assert.Equal(t, userID, resultUserID)
	})
}
