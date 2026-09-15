-- name: GetOrderByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    customer_id,
    status,
    total_amount,
    currency,
    created_at,
    updated_at
FROM orders
WHERE tenant_id = $1
  AND id = $2;