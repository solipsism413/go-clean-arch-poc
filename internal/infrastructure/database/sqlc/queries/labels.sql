-- name: CreateLabel :one
INSERT INTO labels (
    id, name, color, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetLabel :one
SELECT * FROM labels
WHERE id = $1;

-- name: GetLabelByName :one
SELECT * FROM labels
WHERE lower(name) = lower($1);

-- name: ListLabels :many
SELECT * FROM labels
ORDER BY name;

-- name: UpdateLabel :one
UPDATE labels
SET 
    name = COALESCE($2, name),
    color = COALESCE($3, color),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteLabel :exec
DELETE FROM labels
WHERE id = $1;

-- name: LabelExists :one
SELECT EXISTS(SELECT 1 FROM labels WHERE id = $1);

-- name: AddLabelToTask :exec
INSERT INTO task_labels (task_id, label_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveLabelFromTask :exec
DELETE FROM task_labels
WHERE task_id = $1 AND label_id = $2;

-- name: GetTaskLabels :many
SELECT l.* FROM labels l
JOIN task_labels tl ON l.id = tl.label_id
WHERE tl.task_id = $1;

-- name: GetLabelTasks :many
SELECT t.* FROM tasks t
JOIN task_labels tl ON t.id = tl.task_id
WHERE tl.label_id = $1;
