-- name: CreateTask :one
INSERT INTO tasks (
    id, title, description, status, priority, due_date, assignee_id, creator_id, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTasksByStatus :many
SELECT * FROM tasks
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTasksByAssignee :many
SELECT * FROM tasks
WHERE assignee_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTasksByCreator :many
SELECT * FROM tasks
WHERE creator_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTask :one
UPDATE tasks
SET 
    title = COALESCE($2, title),
    description = COALESCE($3, description),
    status = COALESCE($4, status),
    priority = COALESCE($5, priority),
    due_date = $6,
    assignee_id = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks;

-- name: CountTasksByStatus :one
SELECT COUNT(*) FROM tasks
WHERE status = $1;

-- name: TaskExists :one
SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1);

-- name: SearchTasks :many
SELECT * FROM tasks
WHERE
    (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateTaskAttachment :one
INSERT INTO task_attachments (
    id, task_id, filename, s3_key, content_type, size_bytes, uploaded_by, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetTaskAttachment :one
SELECT * FROM task_attachments
WHERE id = $1;

-- name: ListTaskAttachments :many
SELECT * FROM task_attachments
WHERE task_id = $1
ORDER BY created_at DESC;

-- name: DeleteTaskAttachment :exec
DELETE FROM task_attachments
WHERE id = $1;

-- name: DeleteTaskAttachmentsByTask :exec
DELETE FROM task_attachments
WHERE task_id = $1;
