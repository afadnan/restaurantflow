-- name: CreateMenuCategory :one
-- Create a category only when the restaurant belongs to the tenant.
INSERT INTO menu_categories (
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order
)
SELECT
    sqlc.arg(category_id)::uuid,
    r.tenant_id,
    r.id,
    sqlc.arg(name)::text,
    sqlc.narg(description)::text,
    sqlc.arg(sort_order)::integer
FROM restaurants r
WHERE r.tenant_id = sqlc.arg(tenant_id)::uuid
  AND r.id = sqlc.arg(restaurant_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order,
    is_active,
    created_at,
    updated_at;


-- name: GetMenuCategoryByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order,
    is_active,
    created_at,
    updated_at
FROM menu_categories
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(category_id)::uuid;


-- name: ListMenuCategories :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order,
    is_active,
    created_at,
    updated_at
FROM menu_categories
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
ORDER BY sort_order ASC, created_at ASC, id ASC
LIMIT sqlc.arg(query_limit)::integer
OFFSET sqlc.arg(query_offset)::integer;


-- name: UpdateMenuCategory :one
UPDATE menu_categories
SET
    name = sqlc.arg(name)::text,
    description = sqlc.narg(description)::text,
    sort_order = sqlc.arg(sort_order)::integer
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(category_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order,
    is_active,
    created_at,
    updated_at;


-- name: DeactivateMenuCategory :one
UPDATE menu_categories
SET is_active = false
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(category_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    sort_order,
    is_active,
    created_at,
    updated_at;
