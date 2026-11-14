
-- ============================================================
-- RestaurantFlow.fun
-- Refresh Token Queries
-- ============================================================

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    id,
    tenant_id,
    user_id,
    token_hash,
    family_id,
    expires_at
)
VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;


-- name: GetRefreshTokenByHash :one
SELECT *
FROM refresh_tokens
WHERE tenant_id = $1
  AND token_hash = $2;


-- name: LockRefreshTokenByHash :one
SELECT *
FROM refresh_tokens
WHERE tenant_id = $1
  AND token_hash = $2
FOR UPDATE;


-- name: RotateRefreshToken :one
UPDATE refresh_tokens
SET
    revoked_at = NOW(),
    replaced_by_token_id = $3
WHERE tenant_id = $1
  AND id = $2
  AND revoked_at IS NULL
RETURNING *;


-- name: RevokeRefreshTokenFamily :exec
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE tenant_id = $1
  AND family_id = $2;
