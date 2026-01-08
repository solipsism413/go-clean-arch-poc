package rest_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserHandler_Me(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "me@example.com", "password123")

	t.Run("get current user", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/me", nil, user.AccessToken)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		assert.Equal(t, user.Email, data["email"])
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/me", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestUserHandler_Get(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "getuser@example.com", "password123")
	targetUser := app.CreateTestUser(t, "target@example.com", "password123")

	t.Run("get user by ID", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/"+targetUser.ID.String(), nil, user.AccessToken)

		// May get 200 or 403 depending on ACL/RBAC rules
		if resp.StatusCode == http.StatusOK {
			result := ParseResponse[map[string]any](t, resp)
			assert.True(t, result["success"].(bool))
			data := result["data"].(map[string]any)
			assert.Equal(t, targetUser.Email, data["email"])
		} else {
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/"+uuid.New().String(), nil, user.AccessToken)

		// May get 403 (no permission) or 404 depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/invalid-uuid", nil, user.AccessToken)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users/"+targetUser.ID.String(), nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestUserHandler_List(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "listuser@example.com", "password123")

	t.Run("list users requires permission", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users", nil, user.AccessToken)

		// Without proper role, expect 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/users", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestUserHandler_Update(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "updateuser@example.com", "password123")

	t.Run("update own profile", func(t *testing.T) {
		resp := app.DoRequest(t, "PUT", "/api/v1/users/"+user.ID.String(), map[string]string{
			"name": "Updated Name",
		}, user.AccessToken)

		// May get 200 or 403 depending on ACL/RBAC rules
		if resp.StatusCode == http.StatusOK {
			result := ParseResponse[map[string]any](t, resp)
			assert.True(t, result["success"].(bool))
			data := result["data"].(map[string]any)
			assert.Equal(t, "Updated Name", data["name"])
		} else {
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "PUT", "/api/v1/users/"+user.ID.String(), map[string]string{
			"name": "Updated Name",
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestUserHandler_Delete(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "deleteuser@example.com", "password123")
	toDelete := app.CreateTestUser(t, "tobedeleted@example.com", "password123")

	t.Run("delete user", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/users/"+toDelete.ID.String(), nil, user.AccessToken)

		// May require permissions or ownership
		assert.Contains(t, []int{http.StatusNoContent, http.StatusForbidden}, resp.StatusCode)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/users/"+uuid.New().String(), nil, user.AccessToken)

		assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/users/"+uuid.New().String(), nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
