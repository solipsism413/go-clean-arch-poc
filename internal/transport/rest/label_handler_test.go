package rest_test

import (
	"net/http"
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelHandler_CRUD(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	reader := app.CreateTestUser(t, "label-reader@example.com", "password123")
	writer := app.CreateTestUser(t, "label-writer@example.com", "password123")
	writer.AccessToken = createAuthorizedToken(t, app, writer.ID, writer.Email, []string{entity.RoleManager}, nil)

	var createdLabelID string

	t.Run("authenticated reader can list labels", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/labels", nil, reader.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("writer can create label", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/labels", map[string]string{
			"name":  "Bug",
			"color": "#FF0000",
		}, writer.AccessToken)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		createdLabelID = result["data"].(map[string]any)["id"].(string)
		assert.Equal(t, "Bug", result["data"].(map[string]any)["name"])
	})

	t.Run("reader can get label", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/labels/"+createdLabelID, nil, reader.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, createdLabelID, result["data"].(map[string]any)["id"])
	})

	t.Run("duplicate create with different case conflicts", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/labels", map[string]string{
			"name":  "bug",
			"color": "#00FF00",
		}, writer.AccessToken)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("writer can update label", func(t *testing.T) {
		resp := app.DoRequest(t, "PUT", "/api/v1/labels/"+createdLabelID, map[string]string{
			"name":  "Backend",
			"color": "#0000FF",
		}, writer.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, "Backend", result["data"].(map[string]any)["name"])
	})

	t.Run("same record casing-only rename succeeds", func(t *testing.T) {
		resp := app.DoRequest(t, "PUT", "/api/v1/labels/"+createdLabelID, map[string]string{
			"name": "BACKEND",
		}, writer.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, "BACKEND", result["data"].(map[string]any)["name"])
	})

	t.Run("duplicate update with different case conflicts", func(t *testing.T) {
		otherResp := app.DoRequest(t, "POST", "/api/v1/labels", map[string]string{
			"name":  "Feature",
			"color": "#123456",
		}, writer.AccessToken)
		require.Equal(t, http.StatusCreated, otherResp.StatusCode)
		other := ParseResponse[map[string]any](t, otherResp)
		otherLabelID := other["data"].(map[string]any)["id"].(string)

		resp := app.DoRequest(t, "PUT", "/api/v1/labels/"+otherLabelID, map[string]string{
			"name": "backend",
		}, writer.AccessToken)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("reader cannot create label", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/labels", map[string]string{
			"name":  "Reader Write Attempt",
			"color": "#654321",
		}, reader.AccessToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("writer can delete label", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/labels/"+createdLabelID, nil, writer.AccessToken)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("unauthenticated list is rejected", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/labels", nil, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
