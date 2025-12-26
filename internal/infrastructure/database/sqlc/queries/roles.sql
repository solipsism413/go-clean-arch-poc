-- name: CreateRole :one
INSERT INTO roles (
    id, name, description, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpsertRole :one
INSERT INTO roles (
    id, name, description, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (name) DO UPDATE
SET 
    description = EXCLUDED.description,
    updated_at = EXCLUDED.updated_at
RETURNING *;


-- name: GetRole :one
SELECT * FROM roles
WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM roles
WHERE name = $1;

-- name: ListRoles :many
SELECT * FROM roles
ORDER BY name;

-- name: UpdateRole :one
UPDATE roles
SET 
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = $1;

-- name: DeleteRoleByName :exec
DELETE FROM roles
WHERE name = $1;

-- name: DeleteRolesByNames :exec
DELETE FROM roles
WHERE name = ANY($1::text[]);



-- name: RoleExists :one
SELECT EXISTS(SELECT 1 FROM roles WHERE id = $1);

-- name: RoleExistsByName :one
SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1);

-- name: CreatePermission :one
INSERT INTO permissions (
    id, name, resource, action, created_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpsertPermission :one
INSERT INTO permissions (
    id, name, resource, action, created_at
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (name) DO UPDATE
SET 
    resource = EXCLUDED.resource,
    action = EXCLUDED.action
RETURNING *;


-- name: GetPermission :one
SELECT * FROM permissions
WHERE id = $1;

-- name: GetPermissionByName :one
SELECT * FROM permissions
WHERE name = $1;

-- name: ListPermissions :many
SELECT * FROM permissions
ORDER BY resource, action;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = $1;

-- name: PermissionExists :one
SELECT EXISTS(SELECT 1 FROM permissions WHERE id = $1);

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRoles :many
SELECT r.* FROM roles r
JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;

-- name: RemoveAllPermissionsFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1;


-- name: GetRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1;

-- name: GetPermissionsByRoleIDs :many
SELECT rp.role_id, p.* FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = ANY($1::uuid[]);

