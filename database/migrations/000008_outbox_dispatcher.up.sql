-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000008
-- Purpose: Outbox dispatcher claims and leases
-- ============================================================

ALTER TABLE outbox_events
    ADD COLUMN claimed_at TIMESTAMPTZ,
    ADD COLUMN claim_token UUID;

CREATE INDEX idx_outbox_events_dispatchable
    ON outbox_events (created_at, id)
    WHERE published_at IS NULL;