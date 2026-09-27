-- name: CreateOutboxEvent :exec

INSERT INTO outbox_events (
    id,
    tenant_id,
    aggregate_id,
    event_type,
    payload,
    occurred_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
);
