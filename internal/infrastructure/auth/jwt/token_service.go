// Package jwt provides JWT token generation and validation.
package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenService handles JWT token operations.
type TokenService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// NewTokenService creates a new TokenService.
func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{
		secret:          []byte(cfg.SecretKey),
		accessTokenTTL:  cfg.AccessTokenDuration,
		refreshTokenTTL: cfg.RefreshTokenDuration,
		issuer:          cfg.Issuer,
	}
}

// Claims represents the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID      uuid.UUID `json:"userId"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}

// GenerateTokenPair generates an access token and refresh token pair.
func (s *TokenService) GenerateTokenPair(ctx context.Context, userID uuid.UUID, email string, roles, permissions []string) (*dto.AuthOutput, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessTokenTTL)
	refreshExpiry := now.Add(s.refreshTokenTTL)

	// Generate access token
	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token (minimal claims)
	refreshClaims := &jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(refreshExpiry),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &dto.AuthOutput{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// ValidateToken validates an access token and returns its claims.
func (s *TokenService) ValidateToken(ctx context.Context, tokenString string) (*dto.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return &dto.TokenClaims{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time,
	}, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID.
func (s *TokenService) ValidateRefreshToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpiredToken
		}
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}
