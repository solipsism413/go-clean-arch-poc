package rest_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskHandler_Create(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-create@example.com", "password123")

	t.Run("create task with valid data", func(t *testing.T) {
		user.AccessToken = createAuthorizedToken(t, app, user.ID, user.Email, []string{entity.RoleAdmin}, []string{"tasks:*"})

		resp := app.DoRequest(t, "POST", "/api/v1/tasks", map[string]any{
			"title":       "Test Task",
			"description": "Test Description",
			"priority":    "HIGH",
		}, user.AccessToken)

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		assert.Equal(t, "Test Task", data["title"])
		assert.Equal(t, user.ID.String(), data["creatorId"])
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/tasks", map[string]any{
			"title":    "Test Task",
			"priority": "HIGH",
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_ChangeStatus(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-status@example.com", "password123")
	user.AccessToken = createAuthorizedToken(t, app, user.ID, user.Email, []string{entity.RoleAdmin}, []string{"tasks:*"})

	ctx := context.Background()
	taskID := uuid.New()
	_, err := app.Pool.Exec(ctx, `
		INSERT INTO tasks (id, title, description, status, priority, creator_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, taskID, "Archived Task", "Test Description", "ARCHIVED", "HIGH", user.ID)
	require.NoError(t, err)

	_, err = app.Pool.Exec(ctx, `
		INSERT INTO acl_entries (resource_type, resource_id, subject_type, subject_id, permission, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, "task", taskID, "user", user.ID, "admin")
	require.NoError(t, err)

	resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID.String()+"/status", map[string]any{
		"status": "IN_REVIEW",
	}, user.AccessToken)

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	result := ParseResponse[map[string]any](t, resp)
	assert.False(t, result["success"].(bool))
}

func createAuthorizedToken(t *testing.T, app *TestApp, userID uuid.UUID, email string, roles []string, permissions []string) string {
	t.Helper()

	authOutput, err := app.TokenService.GenerateTokenPair(context.Background(), userID, email, roles, nil, permissions)
	require.NoError(t, err)

	return authOutput.AccessToken
}

func TestTaskHandler_List(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-list@example.com", "password123")

	t.Run("list tasks authenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks", nil, user.AccessToken)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])
	})

	t.Run("list with pagination", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks?page=1&pageSize=10", nil, user.AccessToken)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_Get(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-get@example.com", "password123")

	t.Run("get non-existent task", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID, nil, user.AccessToken)

		// May get 403 (no permission) or 404 depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("invalid task ID", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/invalid-uuid", nil, user.AccessToken)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID, nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_Update(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-update@example.com", "password123")

	t.Run("update non-existent task", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "PUT", "/api/v1/tasks/"+taskID, map[string]any{
			"title": "Updated Title",
		}, user.AccessToken)

		// May get 403 (no permission) or 404 depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "PUT", "/api/v1/tasks/"+taskID, map[string]any{
			"title": "Updated Title",
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_Delete(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-delete@example.com", "password123")

	t.Run("delete non-existent task", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+taskID, nil, user.AccessToken)

		// May get 403 (no permission) or 404 depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+taskID, nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskQueryHandler_Search(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-search@example.com", "password123")

	t.Run("search tasks", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/search?q=test", nil, user.AccessToken)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/search?q=test", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskQueryHandler_Overdue(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-overdue@example.com", "password123")

	t.Run("get overdue tasks", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/overdue", nil, user.AccessToken)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/overdue", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_Complete(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-complete@example.com", "password123")

	t.Run("complete non-existent task", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID+"/complete", nil, user.AccessToken)

		require.NotNil(t, resp)
		// May get 403 (no permission) or 404 (not found) depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID+"/complete", nil, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_Assign(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-assign@example.com", "password123")
	assignee := app.CreateTestUser(t, "assignee@example.com", "password123")

	t.Run("assign non-existent task", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID+"/assign", map[string]string{
			"assigneeId": assignee.ID.String(),
		}, user.AccessToken)

		// May get 403 (no permission) or 404 (not found) depending on ACL check order
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskID := uuid.New().String()
		resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID+"/assign", map[string]string{
			"assigneeId": assignee.ID.String(),
		}, "")

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestTaskHandler_HappyPathFlow(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-flow@example.com", "password123")
	user.AccessToken = createAuthorizedToken(t, app, user.ID, user.Email, []string{entity.RoleAdmin}, []string{"tasks:*"})

	createResp := app.DoRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":       "Flow Task",
		"description": "Task flow description",
		"priority":    "HIGH",
	}, user.AccessToken)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	created := ParseResponse[map[string]any](t, createResp)
	taskID := created["data"].(map[string]any)["id"].(string)
	taskUUID, err := uuid.Parse(taskID)
	require.NoError(t, err)

	_, err = app.Pool.Exec(context.Background(), `
		INSERT INTO acl_entries (resource_type, resource_id, subject_type, subject_id, permission, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, "task", taskUUID, "user", user.ID, "admin")
	require.NoError(t, err)

	t.Run("get created task", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID, nil, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, "Flow Task", result["data"].(map[string]any)["title"])
	})

	t.Run("update created task", func(t *testing.T) {
		resp := app.DoRequest(t, "PUT", "/api/v1/tasks/"+taskID, map[string]any{
			"title": "Flow Task Updated",
		}, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, "Flow Task Updated", result["data"].(map[string]any)["title"])
	})

	t.Run("complete created task", func(t *testing.T) {
		resp := app.DoRequest(t, "POST", "/api/v1/tasks/"+taskID+"/complete", nil, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.Equal(t, "DONE", result["data"].(map[string]any)["status"])
	})

	t.Run("delete created task", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+taskID, nil, user.AccessToken)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestTaskHandler_InvalidPaginationParams(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-pagination@example.com", "password123")

	cases := []string{
		"/api/v1/tasks?page=abc",
		"/api/v1/tasks?page=0",
		"/api/v1/tasks?pageSize=-1",
		"/api/v1/tasks?pageSize=999",
		"/api/v1/tasks/search?q=test&sortDesc=not-bool",
		"/api/v1/tasks/overdue?page=abc",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			resp := app.DoRequest(t, "GET", path, nil, user.AccessToken)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestTaskHandler_Attachments(t *testing.T) {
	app := SetupTestApp(t)
	defer app.Cleanup(t)

	user := app.CreateTestUser(t, "task-attachments@example.com", "password123")
	user.AccessToken = createAuthorizedToken(t, app, user.ID, user.Email, []string{entity.RoleAdmin}, []string{"tasks:*"})

	// Create a task
	createResp := app.DoRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":       "Task with attachments",
		"description": "Test attachment flow",
		"priority":    "HIGH",
	}, user.AccessToken)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	created := ParseResponse[map[string]any](t, createResp)
	taskID := created["data"].(map[string]any)["id"].(string)
	taskUUID, err := uuid.Parse(taskID)
	require.NoError(t, err)

	_, err = app.Pool.Exec(context.Background(), `
		INSERT INTO acl_entries (resource_type, resource_id, subject_type, subject_id, permission, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, "task", taskUUID, "user", user.ID, "admin")
	require.NoError(t, err)

	var attachmentID string

	t.Run("upload attachment", func(t *testing.T) {
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("Hello, this is a test file!"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		resp := app.DoMultipartRequest(t, "POST", "/api/v1/tasks/"+taskID+"/attachments", &b, writer.FormDataContentType(), user.AccessToken)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		assert.Equal(t, "test.txt", data["filename"])
		assert.Equal(t, taskID, data["taskId"])
		attachmentID = data["id"].(string)
		assert.Len(t, app.FileStorage.files, 1)
	})

	t.Run("sanitize attachment filename", func(t *testing.T) {
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		part, err := writer.CreateFormFile("file", "../escape.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("escaped"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		resp := app.DoMultipartRequest(t, "POST", "/api/v1/tasks/"+taskID+"/attachments", &b, writer.FormDataContentType(), user.AccessToken)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		data := result["data"].(map[string]any)
		assert.Equal(t, "escape.txt", data["filename"])
	})

	t.Run("reject oversized attachment", func(t *testing.T) {
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		part, err := writer.CreateFormFile("file", "large.bin")
		require.NoError(t, err)
		_, err = part.Write(bytes.Repeat([]byte("a"), (32<<20)+1))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		resp := app.DoMultipartRequest(t, "POST", "/api/v1/tasks/"+taskID+"/attachments", &b, writer.FormDataContentType(), user.AccessToken)
		require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	})

	t.Run("list attachments", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID+"/attachments", nil, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		attachments := data["attachments"].([]any)
		assert.Len(t, attachments, 2)
		assert.Equal(t, "escape.txt", attachments[0].(map[string]any)["filename"])
		assert.Equal(t, "test.txt", attachments[1].(map[string]any)["filename"])
	})

	t.Run("download attachment", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID+"/attachments/"+attachmentID, nil, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, "Hello, this is a test file!", string(body))
	})

	t.Run("delete attachment", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+taskID+"/attachments/"+attachmentID, nil, user.AccessToken)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("list attachments after delete", func(t *testing.T) {
		resp := app.DoRequest(t, "GET", "/api/v1/tasks/"+taskID+"/attachments", nil, user.AccessToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		result := ParseResponse[map[string]any](t, resp)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]any)
		attachments := data["attachments"].([]any)
		assert.Len(t, attachments, 1)
		assert.Equal(t, "escape.txt", attachments[0].(map[string]any)["filename"])
	})

	t.Run("reject cross-task attachment access", func(t *testing.T) {
		otherResp := app.DoRequest(t, "POST", "/api/v1/tasks", map[string]any{
			"title":       "Other Task",
			"description": "Second task",
			"priority":    "HIGH",
		}, user.AccessToken)
		require.Equal(t, http.StatusCreated, otherResp.StatusCode)

		otherCreated := ParseResponse[map[string]any](t, otherResp)
		otherTaskID := otherCreated["data"].(map[string]any)["id"].(string)
		otherTaskUUID, err := uuid.Parse(otherTaskID)
		require.NoError(t, err)

		_, err = app.Pool.Exec(context.Background(), `
			INSERT INTO acl_entries (resource_type, resource_id, subject_type, subject_id, permission, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, "task", otherTaskUUID, "user", user.ID, "admin")
		require.NoError(t, err)

		downloadResp := app.DoRequest(t, "GET", "/api/v1/tasks/"+otherTaskID+"/attachments/"+attachmentID, nil, user.AccessToken)
		assert.Equal(t, http.StatusNotFound, downloadResp.StatusCode)

		deleteResp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+otherTaskID+"/attachments/"+attachmentID, nil, user.AccessToken)
		assert.Equal(t, http.StatusNotFound, deleteResp.StatusCode)
	})

	t.Run("delete task cleans attachment storage", func(t *testing.T) {
		resp := app.DoRequest(t, "DELETE", "/api/v1/tasks/"+taskID, nil, user.AccessToken)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, app.FileStorage.files)
	})
}
