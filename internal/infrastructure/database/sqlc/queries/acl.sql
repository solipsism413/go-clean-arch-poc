-- name: CreateACLEntry :one
INSERT INTO acl_entries (
    id, resource_type, resource_id, subject_type, subject_id, permission, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetACLEntry :one
SELECT * FROM acl_entries
WHERE id = $1;

-- name: GetACLEntriesByResource :many
SELECT * FROM acl_entries
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at;

-- name: GetACLEntriesBySubject :many
SELECT * FROM acl_entries
WHERE subject_type = $1 AND subject_id = $2
ORDER BY created_at;

-- name: DeleteACLEntry :exec
DELETE FROM acl_entries
WHERE id = $1;

-- name: DeleteACLEntriesByResource :exec
DELETE FROM acl_entries
WHERE resource_type = $1 AND resource_id = $2;

-- name: HasPermission :one
SELECT EXISTS(
    SELECT 1 FROM acl_entries
    WHERE resource_type = $1 
      AND resource_id = $2 
      AND subject_type = $3 
      AND subject_id = $4 
      AND (permission = $5 OR permission = 'admin')
);
