-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000006
-- Purpose: Kitchen Display System
-- ============================================================


-- ============================================================
-- ENUM
-- ============================================================

CREATE TYPE kds_status AS ENUM (
    'pending',
    'preparing',
    'ready',
    'completed'
);


-- ============================================================
-- KDS TICKETS
-- ============================================================

CREATE TABLE kds_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE RESTRICT,

    order_id UUID NOT NULL
        REFERENCES orders(id)
        ON DELETE CASCADE,

    status kds_status NOT NULL DEFAULT 'pending',

    priority INTEGER NOT NULL DEFAULT 0,

    started_at TIMESTAMPTZ,

    ready_at TIMESTAMPTZ,

    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (order_id)
);


CREATE INDEX idx_kds_tickets_tenant
    ON kds_tickets(tenant_id);

CREATE INDEX idx_kds_tickets_restaurant
    ON kds_tickets(restaurant_id);

CREATE INDEX idx_kds_tickets_status
    ON kds_tickets(tenant_id, restaurant_id, status);

CREATE INDEX idx_kds_tickets_queue
    ON kds_tickets(
        tenant_id,
        restaurant_id,
        status,
        priority DESC,
        created_at ASC
    );


CREATE TRIGGER kds_tickets_set_updated_at
BEFORE UPDATE ON kds_tickets
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- KDS TICKET ITEMS
-- ============================================================

CREATE TABLE kds_ticket_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    kds_ticket_id UUID NOT NULL
        REFERENCES kds_tickets(id)
        ON DELETE CASCADE,

    order_item_id UUID NOT NULL
        REFERENCES order_items(id)
        ON DELETE CASCADE,

    item_name TEXT NOT NULL,

    quantity INTEGER NOT NULL
        CHECK (quantity > 0),

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_kds_ticket_items_tenant
    ON kds_ticket_items(tenant_id);

CREATE INDEX idx_kds_ticket_items_ticket
    ON kds_ticket_items(kds_ticket_id);


-- ============================================================
-- KDS STATUS HISTORY
-- ============================================================

CREATE TABLE kds_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    kds_ticket_id UUID NOT NULL
        REFERENCES kds_tickets(id)
        ON DELETE CASCADE,

    from_status kds_status,

    to_status kds_status NOT NULL,

    changed_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_kds_status_history_tenant
    ON kds_status_history(tenant_id);

CREATE INDEX idx_kds_status_history_ticket
    ON kds_status_history(kds_ticket_id);

CREATE INDEX idx_kds_status_history_changed_at
    ON kds_status_history(kds_ticket_id, changed_at DESC);


-- ============================================================
-- RLS
-- ============================================================

ALTER TABLE kds_tickets ENABLE ROW LEVEL SECURITY;

ALTER TABLE kds_ticket_items ENABLE ROW LEVEL SECURITY;

ALTER TABLE kds_status_history ENABLE ROW LEVEL SECURITY;


CREATE POLICY kds_tickets_tenant_isolation
ON kds_tickets
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY kds_ticket_items_tenant_isolation
ON kds_ticket_items
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY kds_status_history_tenant_isolation
ON kds_status_history
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());