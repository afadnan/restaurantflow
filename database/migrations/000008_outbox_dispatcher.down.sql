-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000008
-- Purpose: Remove outbox dispatcher claims and leases
-- ============================================================

DROP INDEX IF EXISTS idx_outbox_events_dispatchable;

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS claim_token,
    DROP COLUMN IF EXISTS claimed_at;