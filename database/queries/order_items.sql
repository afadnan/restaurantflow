-- name: CreateOrderItem :one
INSERT INTO order_items (
    tenant_id,
    order_id,
    menu_item_id,
    item_name,
    unit_price,
    quantity,
    subtotal,
    notes
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING
    id,
    tenant_id,
    order_id,
    menu_item_id,
    item_name,
    unit_price,
    quantity,
    subtotal,
    notes,
    created_at;


-- name: GetOrderItemByID :one
SELECT
    id,
    tenant_id,
    order_id,
    menu_item_id,
    item_name,
    unit_price,
    quantity,
    subtotal,
    notes,
    created_at
FROM order_items
WHERE tenant_id = $1
  AND id = $2;


-- name: ListOrderItems :many
SELECT
    id,
    tenant_id,
    order_id,
    menu_item_id,
    item_name,
    unit_price,
    quantity,
    subtotal,
    notes,
    created_at
FROM order_items
WHERE tenant_id = $1
  AND order_id = $2
ORDER BY created_at ASC;