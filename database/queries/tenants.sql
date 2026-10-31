-- name: CreateTenant :one
INSERT INTO tenants (
    id,
    name,
    slug,
    status
)
VALUES (
    $1,
    $2,
    $3,
    'active'
)
RETURNING
    id,
    name,
    slug,
    status,
    created_at,
    updated_at;