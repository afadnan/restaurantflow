-- name: GetRestaurantByID :one
SELECT
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at
FROM restaurants
WHERE tenant_id = $1
  AND id = $2;


-- name: GetRestaurantBySlug :one
SELECT
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at
FROM restaurants
WHERE tenant_id = $1
  AND slug = $2;


-- name: CreateRestaurant :one
INSERT INTO restaurants (
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16
)
RETURNING
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at;


-- name: UpdateRestaurant :one
UPDATE restaurants
SET
    name = $3,
    slug = $4,
    description = $5,
    phone = $6,
    email = $7,
    address_line_1 = $8,
    address_line_2 = $9,
    city = $10,
    state = $11,
    postal_code = $12,
    country = $13,
    latitude = $14,
    longitude = $15,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at;


-- name: ActivateRestaurant :one
UPDATE restaurants
SET
    status = 'active',
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at;


-- name: DeactivateRestaurant :one
UPDATE restaurants
SET
    status = 'inactive',
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at;


-- name: SuspendRestaurant :one
UPDATE restaurants
SET
    status = 'suspended',
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at;

-- name: ListRestaurants :many
SELECT
    id,
    tenant_id,
    name,
    slug,
    description,
    phone,
    email,
    address_line_1,
    address_line_2,
    city,
    state,
    postal_code,
    country,
    latitude,
    longitude,
    status,
    created_at,
    updated_at
FROM restaurants
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;