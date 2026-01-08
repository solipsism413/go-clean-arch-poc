package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Login(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	// Create test user
	email := "login@example.com"
	password := "password123"
	app.CreateTestUser(t, email, password)

	t.Run("valid credentials", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, "")

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]any
		result = ParseResponse[map[string]any](t, resp)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		assert.NotEmpty(t, data["accessToken"])
		assert.NotEmpty(t, data["refreshToken"])
	})

	t.Run("invalid password", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": "wrongpassword",
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid email", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/login", map[string]string{
			"email":    "nonexistent@example.com",
			"password": password,
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid body", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/login", "invalid json body", "")

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	// Create user and get tokens via login
	email := "refresh@example.com"
	password := "password123"
	app.CreateTestUser(t, email, password)

	// Login to get refresh token
	loginResp := app.DoRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	loginResult := ParseResponse[map[string]any](t, loginResp)
	refreshToken := loginResult["data"].(map[string]any)["refreshToken"].(string)

	t.Run("valid refresh token", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/refresh", map[string]string{
			"refreshToken": refreshToken,
		}, "")

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		assert.NotEmpty(t, data["accessToken"])
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/refresh", map[string]string{
			"refreshToken": "invalid-token",
		}, "")

		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "logout@example.com", "password123")

	t.Run("authenticated logout", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/logout", nil, user.AccessToken)

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/logout", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	email := "changepwd@example.com"
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"
	user := app.CreateTestUser(t, email, oldPassword)

	t.Run("valid password change", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/change-password", map[string]string{
			"oldPassword": oldPassword,
			"newPassword": newPassword,
		}, user.AccessToken)

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify can login with new password
		loginResp := app.DoRequest(t, "POST", "/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": newPassword,
		}, "")
		assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	})

	t.Run("wrong old password", func(t *testing.T) {
		// Create another user for this test
		user2 := app.CreateTestUser(t, "changepwd2@example.com", "password123")

		resp := app.DoRequest(t, "POST", "/api/v1/auth/change-password", map[string]string{
			"oldPassword": "wrongpassword",
			"newPassword": "newpassword",
		}, user2.AccessToken)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/auth/change-password", map[string]string{
			"oldPassword": oldPassword,
			"newPassword": newPassword,
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestHealthEndpoint(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	resp := app.DoRequest(t, "GET", "/health", nil, "")

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	result := ParseResponse[map[string]string](t, resp)
	assert.Equal(t, "ok", result["status"])
}
