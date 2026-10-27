
-- name: AttachMenuItemAddon :one
-- Only attach an active add-on belonging to the same tenant and restaurant.
INSERT INTO menu_item_addons (
    tenant_id,
    restaurant_id,
    menu_item_id,
    addon_id,
    is_required,
    max_quantity
)
SELECT
    mi.tenant_id,
    mi.restaurant_id,
    mi.id,
    ma.id,
    sqlc.arg(is_required)::boolean,
    sqlc.arg(max_quantity)::integer
FROM menu_items AS mi
JOIN menu_addons AS ma
  ON ma.tenant_id = mi.tenant_id
 AND ma.restaurant_id = mi.restaurant_id
WHERE mi.tenant_id = sqlc.arg(tenant_id)::uuid
  AND mi.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND mi.id = sqlc.arg(item_id)::uuid
  AND ma.id = sqlc.arg(addon_id)::uuid
  AND ma.is_active = TRUE
RETURNING
    tenant_id,
    restaurant_id,
    menu_item_id,
    addon_id,
    is_required,
    max_quantity;


-- name: GetMenuItemAddon :one
SELECT
    tenant_id,
    restaurant_id,
    menu_item_id,
    addon_id,
    is_required,
    max_quantity
FROM menu_item_addons
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
  AND addon_id = sqlc.arg(addon_id)::uuid;


-- name: ListMenuItemAddons :many
SELECT
    mia.tenant_id,
    mia.restaurant_id,
    mia.menu_item_id,
    mia.addon_id,
    mia.is_required,
    mia.max_quantity
FROM menu_item_addons AS mia
WHERE mia.tenant_id = sqlc.arg(tenant_id)::uuid
  AND mia.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND mia.menu_item_id = sqlc.arg(item_id)::uuid
ORDER BY mia.addon_id
LIMIT sqlc.arg(query_limit)::integer
OFFSET sqlc.arg(query_offset)::integer;


-- name: UpdateMenuItemAddon :one
UPDATE menu_item_addons AS mia
SET
    is_required = sqlc.arg(is_required)::boolean,
    max_quantity = sqlc.arg(max_quantity)::integer
FROM menu_addons AS ma
WHERE mia.tenant_id = sqlc.arg(tenant_id)::uuid
  AND mia.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND mia.menu_item_id = sqlc.arg(item_id)::uuid
  AND mia.addon_id = sqlc.arg(addon_id)::uuid
  AND ma.tenant_id = mia.tenant_id
  AND ma.restaurant_id = mia.restaurant_id
  AND ma.id = mia.addon_id
  AND ma.is_active = TRUE
RETURNING
    mia.tenant_id,
    mia.restaurant_id,
    mia.menu_item_id,
    mia.addon_id,
    mia.is_required,
    mia.max_quantity;


-- name: DetachMenuItemAddon :one
DELETE FROM menu_item_addons
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND menu_item_id = sqlc.arg(item_id)::uuid
  AND addon_id = sqlc.arg(addon_id)::uuid
RETURNING
    tenant_id,
    restaurant_id,
    menu_item_id,
    addon_id,
    is_required,
    max_quantity;
