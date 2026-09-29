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


-- name: ClaimOutboxEvents :many

WITH candidates AS (
    SELECT id
    FROM outbox_events
    WHERE published_at IS NULL
      AND (
          claimed_at IS NULL
          OR claimed_at < NOW()
              - (
                  sqlc.arg(lease_seconds)::integer
                  * INTERVAL '1 second'
              )
      )
    ORDER BY created_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE outbox_events AS e
SET
    claimed_at = NOW(),
    claim_token = sqlc.arg(claim_token)
FROM candidates
WHERE e.id = candidates.id
RETURNING
    e.id,
    e.tenant_id,
    e.aggregate_id,
    e.event_type,
    e.payload,
    e.occurred_at,
    e.created_at,
    e.published_at,
    e.attempts,
    e.last_error;


-- name: MarkOutboxEventPublished :exec

UPDATE outbox_events
SET
    published_at = NOW(),
    claimed_at = NULL,
    claim_token = NULL,
    last_error = NULL
WHERE id = $1
  AND claim_token = $2
  AND published_at IS NULL;


-- name: MarkOutboxEventFailed :exec

UPDATE outbox_events
SET
    attempts = attempts + 1,
    last_error = $3,
    claimed_at = NULL,
    claim_token = NULL
WHERE id = $1
  AND claim_token = $2
  AND published_at IS NULL;