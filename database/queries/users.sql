
-- name: CreateUser :one
INSERT INTO users (
    id,
    tenant_id,
    email,
    phone,
    password_hash,
    first_name,
    last_name,
    role,
    status
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
)
RETURNING *;


-- name: GetUserByID :one
SELECT *
FROM users
WHERE tenant_id = $1
  AND id = $2;


-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE tenant_id = $1
  AND email = $2;


-- name: UpdateUserLastLogin :one
UPDATE users
SET last_login_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;
