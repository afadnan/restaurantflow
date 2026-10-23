-- name: CreateMenuItemVariant :one
-- Create a variant only when its parent item belongs to the tenant and restaurant.
INSERT INTO menu_item_variants (
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta
)
SELECT
    sqlc.arg(variant_id)::uuid,
    mi.tenant_id,
    mi.restaurant_id,
    mi.id,
    sqlc.arg(name)::text,
    sqlc.arg(price_delta)::text::numeric(12, 2)
FROM menu_items AS mi
WHERE mi.tenant_id = sqlc.arg(tenant_id)::uuid
  AND mi.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND mi.id = sqlc.arg(item_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta::text AS price_delta,
    is_active,
    created_at,
    updated_at;

-- name: GetMenuItemVariantByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta::text AS price_delta,
    is_active,
    created_at,
    updated_at
FROM menu_item_variants
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
  AND id = sqlc.arg(variant_id)::uuid;

-- name: ListMenuItemVariants :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta::text AS price_delta,
    is_active,
    created_at,
    updated_at
FROM menu_item_variants
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg(query_limit)::integer
OFFSET sqlc.arg(query_offset)::integer;

-- name: UpdateMenuItemVariant :one
UPDATE menu_item_variants
SET
    name = sqlc.arg(name)::text,
    price_delta = sqlc.arg(price_delta)::text::numeric(12, 2),
    updated_at = sqlc.arg(updated_at)::timestamptz
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
  AND id = sqlc.arg(variant_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta::text AS price_delta,
    is_active,
    created_at,
    updated_at;

-- name: DeactivateMenuItemVariant :one
UPDATE menu_item_variants
SET
    is_active = FALSE,
    updated_at = sqlc.arg(updated_at)::timestamptz
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
  AND id = sqlc.arg(variant_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    menu_item_id,
    name,
    price_delta::text AS price_delta,
    is_active,
    created_at,
    updated_at;
