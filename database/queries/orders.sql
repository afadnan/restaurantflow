-- name: GetOrderByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at
FROM orders
WHERE tenant_id = $1
  AND id = $2;


-- name: CreateOrder :one
INSERT INTO orders (
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
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
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16,
    $17
)
RETURNING
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at;


-- name: UpdateOrderStatus :one
UPDATE orders
SET
    status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at;


-- name: UpdateOrderPaymentStatus :one
UPDATE orders
SET
    payment_status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at;


-- name: CompleteOrder :one
UPDATE orders
SET
    status = 'completed',
    completed_at = NOW(),
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at;


-- name: CancelOrder :one
UPDATE orders
SET
    status = 'cancelled',
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at;


-- name: ListOrders :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    table_id,
    customer_id,
    order_number,
    order_type,
    status,
    payment_status,
    subtotal,
    tax_amount,
    discount_amount,
    service_fee,
    total_amount,
    currency,
    customer_name,
    customer_phone,
    notes,
    created_at,
    updated_at,
    completed_at,
    cancelled_at
FROM orders
WHERE tenant_id = $1
  AND restaurant_id = $2
ORDER BY created_at DESC
LIMIT $3
OFFSET $4;