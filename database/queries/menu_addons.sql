
-- name: CreateMenuAddon :one
INSERT INTO menu_addons (
    id,
    tenant_id,
    restaurant_id,
    name,
    description,
    price
)
SELECT
    sqlc.arg(addon_id)::uuid,
    r.tenant_id,
    r.id,
    sqlc.arg(name)::text,
    NULLIF(sqlc.arg(description)::text, ''),
    sqlc.arg(price)::text::numeric(12, 2)
FROM restaurants AS r
WHERE r.tenant_id = sqlc.arg(tenant_id)::uuid
  AND r.id = sqlc.arg(restaurant_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    COALESCE(description, '') AS description,
    price::text AS price,
    is_active,
    created_at,
    updated_at;


-- name: GetMenuAddonByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    name,
    COALESCE(description, '') AS description,
    price::text AS price,
    is_active,
    created_at,
    updated_at
FROM menu_addons
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(addon_id)::uuid;


-- name: ListMenuAddons :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    name,
    COALESCE(description, '') AS description,
    price::text AS price,
    is_active,
    created_at,
    updated_at
FROM menu_addons
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg(query_limit)::integer
OFFSET sqlc.arg(query_offset)::integer;


-- name: UpdateMenuAddon :one
UPDATE menu_addons
SET
    name = sqlc.arg(name)::text,
    description = NULLIF(sqlc.arg(description)::text, ''),
    price = sqlc.arg(price)::text::numeric(12, 2),
    updated_at = sqlc.arg(updated_at)::timestamptz
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(addon_id)::uuid
  AND is_active = TRUE
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    COALESCE(description, '') AS description,
    price::text AS price,
    is_active,
    created_at,
    updated_at;


-- name: DeactivateMenuAddon :one
UPDATE menu_addons
SET
    is_active = FALSE,
    updated_at = sqlc.arg(updated_at)::timestamptz
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(addon_id)::uuid
  AND is_active = TRUE
RETURNING
    id,
    tenant_id,
    restaurant_id,
    name,
    COALESCE(description, '') AS description,
    price::text AS price,
    is_active,
    created_at,
    updated_at;
