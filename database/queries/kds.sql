-- name: CreateKDSTicket :one
INSERT INTO kds_tickets (
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority
)
VALUES (
    $1,
    $2,
    $3,
    'pending',
    $4
)
RETURNING
    id,
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority,
    started_at,
    ready_at,
    completed_at,
    created_at,
    updated_at;


-- name: GetKDSTicketByID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority,
    started_at,
    ready_at,
    completed_at,
    created_at,
    updated_at
FROM kds_tickets
WHERE tenant_id = $1
  AND id = $2;


-- name: GetKDSTicketByOrderID :one
SELECT
    id,
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority,
    started_at,
    ready_at,
    completed_at,
    created_at,
    updated_at
FROM kds_tickets
WHERE tenant_id = $1
  AND order_id = $2;


-- name: UpdateKDSStatus :one
UPDATE kds_tickets
SET
    status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING
    id,
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority,
    started_at,
    ready_at,
    completed_at,
    created_at,
    updated_at;


-- name: ListKDSQueue :many
SELECT
    id,
    tenant_id,
    restaurant_id,
    order_id,
    status,
    priority,
    started_at,
    ready_at,
    completed_at,
    created_at,
    updated_at
FROM kds_tickets
WHERE tenant_id = $1
  AND restaurant_id = $2
  AND status IN (
      'pending',
      'preparing',
      'ready'
  )
ORDER BY
    priority DESC,
    created_at ASC;