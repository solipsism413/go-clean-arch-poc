-- name: CreateUser :one
INSERT INTO users (
    id, email, password_hash, name, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET 
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    password_hash = COALESCE($4, password_hash),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: SearchUsers :many
SELECT * FROM users
WHERE 
    (name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
