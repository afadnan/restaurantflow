-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000007
-- Purpose: Transactional outbox events
-- ============================================================


-- ============================================================
-- OUTBOX EVENTS
-- ============================================================

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    aggregate_id UUID NOT NULL,

    event_type TEXT NOT NULL,

    payload JSONB NOT NULL,

    occurred_at TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    published_at TIMESTAMPTZ,

    attempts INTEGER NOT NULL DEFAULT 0
        CHECK (attempts >= 0),

    last_error TEXT
);


-- ============================================================
-- INDEXES
-- ============================================================

CREATE INDEX idx_outbox_events_tenant
    ON outbox_events(tenant_id);

CREATE INDEX idx_outbox_events_aggregate
    ON outbox_events(tenant_id, aggregate_id);

CREATE INDEX idx_outbox_events_type
    ON outbox_events(tenant_id, event_type);

CREATE INDEX idx_outbox_events_created_at
    ON outbox_events(created_at);


-- Unpublished events are the events that the publisher needs
-- to process.
CREATE INDEX idx_outbox_events_unpublished
    ON outbox_events(created_at)
    WHERE published_at IS NULL;


-- ============================================================
-- ROW LEVEL SECURITY
-- ============================================================

ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;


CREATE POLICY outbox_events_tenant_isolation
ON outbox_events
USING (
    tenant_id = current_tenant_id()
)
WITH CHECK (
    tenant_id = current_tenant_id()
);