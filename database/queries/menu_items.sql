
-- name: CreateMenuItem :one
INSERT INTO menu_items (
    tenant_id,
    restaurant_id,
    category_id,
    name,
    description,
    sku,
    price,
    status,
    image_url,
    sort_order,
    is_vegetarian,
    is_vegan,
    is_available
)
SELECT
    sqlc.arg(tenant_id)::uuid,
    sqlc.arg(restaurant_id)::uuid,
    c.id,
    sqlc.arg(name)::text,
    NULLIF(sqlc.arg(description)::text, ''),
    NULLIF(sqlc.arg(sku)::text, ''),
    sqlc.arg(price)::text::numeric(12, 2),
    sqlc.arg(status)::text::menu_item_status,
    NULLIF(sqlc.arg(image_url)::text, ''),
    sqlc.arg(sort_order)::integer,
    sqlc.arg(is_vegetarian)::boolean,
    sqlc.arg(is_vegan)::boolean,
    sqlc.arg(is_available)::boolean
FROM menu_categories AS c
WHERE c.id = sqlc.arg(category_id)::uuid
  AND c.tenant_id = sqlc.arg(tenant_id)::uuid
  AND c.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND c.is_active = TRUE
RETURNING
    id,
    tenant_id,
    restaurant_id,
    category_id,
    name,
    COALESCE(description, '') AS description,
    COALESCE(sku, '') AS sku,
    price::text AS price,
    status::text AS status,
    COALESCE(image_url, '') AS image_url,
    sort_order,
    is_vegetarian,
    is_vegan,
    is_available,
    created_at,
    updated_at;


-- name: GetMenuItem :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    category_id,
    name,
    COALESCE(description, '') AS description,
    COALESCE(sku, '') AS sku,
    price::text AS price,
    status::text AS status,
    COALESCE(image_url, '') AS image_url,
    sort_order,
    is_vegetarian,
    is_vegan,
    is_available,
    created_at,
    updated_at
FROM menu_items
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(item_id)::uuid;


-- name: ListMenuItems :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    category_id,
    name,
    COALESCE(description, '') AS description,
    COALESCE(sku, '') AS sku,
    price::text AS price,
    status::text AS status,
    COALESCE(image_url, '') AS image_url,
    sort_order,
    is_vegetarian,
    is_vegan,
    is_available,
    created_at,
    updated_at
FROM menu_items
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
ORDER BY sort_order, created_at, id
LIMIT sqlc.arg(limit_count)::integer
OFFSET sqlc.arg(offset_count)::integer;


-- name: UpdateMenuItem :one
UPDATE menu_items AS mi
SET
    category_id = c.id,
    name = sqlc.arg(name)::text,
    description = NULLIF(sqlc.arg(description)::text, ''),
    sku = NULLIF(sqlc.arg(sku)::text, ''),
    price = sqlc.arg(price)::text::numeric(12, 2),
    status = sqlc.arg(status)::text::menu_item_status,
    image_url = NULLIF(sqlc.arg(image_url)::text, ''),
    sort_order = sqlc.arg(sort_order)::integer,
    is_vegetarian = sqlc.arg(is_vegetarian)::boolean,
    is_vegan = sqlc.arg(is_vegan)::boolean,
    is_available = sqlc.arg(is_available)::boolean
FROM menu_categories AS c
WHERE mi.tenant_id = sqlc.arg(tenant_id)::uuid
  AND mi.restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND mi.id = sqlc.arg(item_id)::uuid
  AND c.id = sqlc.arg(category_id)::uuid
  AND c.tenant_id = mi.tenant_id
  AND c.restaurant_id = mi.restaurant_id
  AND c.is_active = TRUE
RETURNING
    mi.id,
    mi.tenant_id,
    mi.restaurant_id,
    mi.category_id,
    mi.name,
    COALESCE(mi.description, '') AS description,
    COALESCE(mi.sku, '') AS sku,
    mi.price::text AS price,
    mi.status::text AS status,
    COALESCE(mi.image_url, '') AS image_url,
    mi.sort_order,
    mi.is_vegetarian,
    mi.is_vegan,
    mi.is_available,
    mi.created_at,
    mi.updated_at;


-- name: DeactivateMenuItem :one
UPDATE menu_items
SET
    status = 'inactive',
    is_available = FALSE
WHERE tenant_id = sqlc.arg(tenant_id)::uuid
  AND restaurant_id = sqlc.arg(restaurant_id)::uuid
  AND id = sqlc.arg(item_id)::uuid
RETURNING
    id,
    tenant_id,
    restaurant_id,
    category_id,
    name,
    COALESCE(description, '') AS description,
    COALESCE(sku, '') AS sku,
    price::text AS price,
    status::text AS status,
    COALESCE(image_url, '') AS image_url,
    sort_order,
    is_vegetarian,
    is_vegan,
    is_available,
    created_at,
    updated_at;
