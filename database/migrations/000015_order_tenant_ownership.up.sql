-- ============================================================
-- Migration 000015: Order and KDS ownership hardening
-- ============================================================
-- Enforce tenant and restaurant ownership across:
--   restaurant_tables
--   orders
--   order_items
--   order_item_addons
--   kds_tickets
--   kds_ticket_items
--   kds_status_history
--
-- Existing ownership mismatches must cause this migration
-- to fail. No records are automatically deleted or repaired.
--
-- PostgreSQL 18 supports column-specific ON DELETE SET NULL.
-- ============================================================


-- ============================================================
-- 1. RESTAURANT TABLE OWNERSHIP
-- ============================================================

ALTER TABLE restaurant_tables
    ADD CONSTRAINT uq_restaurant_tables_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);

ALTER TABLE restaurant_tables
    DROP CONSTRAINT restaurant_tables_restaurant_id_fkey;

ALTER TABLE restaurant_tables
    ADD CONSTRAINT fk_restaurant_tables_tenant_restaurant
    FOREIGN KEY (tenant_id, restaurant_id)
    REFERENCES restaurants (tenant_id, id)
    ON DELETE CASCADE;


-- ============================================================
-- 2. ORDER OWNERSHIP
-- ============================================================

ALTER TABLE orders
    ADD CONSTRAINT uq_orders_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);

ALTER TABLE orders
    DROP CONSTRAINT orders_restaurant_id_fkey;

ALTER TABLE orders
    DROP CONSTRAINT orders_table_id_fkey;

ALTER TABLE orders
    DROP CONSTRAINT orders_customer_id_fkey;

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_tenant_restaurant
    FOREIGN KEY (tenant_id, restaurant_id)
    REFERENCES restaurants (tenant_id, id)
    ON DELETE RESTRICT;

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_tenant_restaurant_table
    FOREIGN KEY (tenant_id, restaurant_id, table_id)
    REFERENCES restaurant_tables (tenant_id, restaurant_id, id)
    ON DELETE SET NULL (table_id);

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_tenant_customer
    FOREIGN KEY (tenant_id, customer_id)
    REFERENCES users (tenant_id, id)
    ON DELETE SET NULL (customer_id);


-- ============================================================
-- 3. ORDER ITEM OWNERSHIP
-- ============================================================

ALTER TABLE order_items
    ADD COLUMN restaurant_id UUID;

UPDATE order_items oi
SET restaurant_id = o.restaurant_id
FROM orders o
WHERE o.tenant_id = oi.tenant_id
  AND o.id = oi.order_id;

ALTER TABLE order_items
    ALTER COLUMN restaurant_id SET NOT NULL;

ALTER TABLE order_items
    ADD CONSTRAINT uq_order_items_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);

ALTER TABLE order_items
    ADD CONSTRAINT uq_order_items_tenant_restaurant_order_item
    UNIQUE (tenant_id, restaurant_id, order_id, id);

ALTER TABLE order_items
    DROP CONSTRAINT order_items_order_id_fkey;

ALTER TABLE order_items
    DROP CONSTRAINT order_items_menu_item_id_fkey;

ALTER TABLE order_items
    ADD CONSTRAINT fk_order_items_tenant_restaurant_order
    FOREIGN KEY (tenant_id, restaurant_id, order_id)
    REFERENCES orders (tenant_id, restaurant_id, id)
    ON DELETE CASCADE;

ALTER TABLE order_items
    ADD CONSTRAINT fk_order_items_tenant_restaurant_menu_item
    FOREIGN KEY (tenant_id, restaurant_id, menu_item_id)
    REFERENCES menu_items (tenant_id, restaurant_id, id)
    ON DELETE RESTRICT;


-- ============================================================
-- 4. ORDER ITEM ADD-ON OWNERSHIP
-- ============================================================

ALTER TABLE order_item_addons
    ADD COLUMN restaurant_id UUID;

UPDATE order_item_addons a
SET restaurant_id = oi.restaurant_id
FROM order_items oi
WHERE oi.tenant_id = a.tenant_id
  AND oi.id = a.order_item_id;

ALTER TABLE order_item_addons
    ALTER COLUMN restaurant_id SET NOT NULL;

ALTER TABLE order_item_addons
    DROP CONSTRAINT order_item_addons_order_item_id_fkey;

ALTER TABLE order_item_addons
    DROP CONSTRAINT order_item_addons_addon_id_fkey;

ALTER TABLE order_item_addons
    ADD CONSTRAINT fk_order_item_addons_tenant_restaurant_item
    FOREIGN KEY (tenant_id, restaurant_id, order_item_id)
    REFERENCES order_items (tenant_id, restaurant_id, id)
    ON DELETE CASCADE;

ALTER TABLE order_item_addons
    ADD CONSTRAINT fk_order_item_addons_tenant_restaurant_addon
    FOREIGN KEY (tenant_id, restaurant_id, addon_id)
    REFERENCES menu_addons (tenant_id, restaurant_id, id)
    ON DELETE SET NULL (addon_id);


-- ============================================================
-- 5. KDS TICKET OWNERSHIP
-- ============================================================

ALTER TABLE kds_tickets
    ADD CONSTRAINT uq_kds_tickets_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);

ALTER TABLE kds_tickets
    ADD CONSTRAINT uq_kds_tickets_tenant_restaurant_order
    UNIQUE (tenant_id, restaurant_id, order_id, id);

ALTER TABLE kds_tickets
    DROP CONSTRAINT kds_tickets_restaurant_id_fkey;

ALTER TABLE kds_tickets
    DROP CONSTRAINT kds_tickets_order_id_fkey;

ALTER TABLE kds_tickets
    ADD CONSTRAINT fk_kds_tickets_tenant_restaurant
    FOREIGN KEY (tenant_id, restaurant_id)
    REFERENCES restaurants (tenant_id, id)
    ON DELETE RESTRICT;

ALTER TABLE kds_tickets
    ADD CONSTRAINT fk_kds_tickets_tenant_restaurant_order
    FOREIGN KEY (tenant_id, restaurant_id, order_id)
    REFERENCES orders (tenant_id, restaurant_id, id)
    ON DELETE CASCADE;


-- ============================================================
-- 6. KDS TICKET ITEM OWNERSHIP
-- ============================================================

ALTER TABLE kds_ticket_items
    ADD COLUMN restaurant_id UUID;

ALTER TABLE kds_ticket_items
    ADD COLUMN order_id UUID;

UPDATE kds_ticket_items ki
SET
    restaurant_id = kt.restaurant_id,
    order_id = kt.order_id
FROM kds_tickets kt
WHERE kt.tenant_id = ki.tenant_id
  AND kt.id = ki.kds_ticket_id;

ALTER TABLE kds_ticket_items
    ALTER COLUMN restaurant_id SET NOT NULL;

ALTER TABLE kds_ticket_items
    ALTER COLUMN order_id SET NOT NULL;

ALTER TABLE kds_ticket_items
    DROP CONSTRAINT kds_ticket_items_kds_ticket_id_fkey;

ALTER TABLE kds_ticket_items
    DROP CONSTRAINT kds_ticket_items_order_item_id_fkey;

ALTER TABLE kds_ticket_items
    ADD CONSTRAINT fk_kds_ticket_items_exact_order_ticket
    FOREIGN KEY (tenant_id, restaurant_id, order_id, kds_ticket_id)
    REFERENCES kds_tickets (tenant_id, restaurant_id, order_id, id)
    ON DELETE CASCADE;

ALTER TABLE kds_ticket_items
    ADD CONSTRAINT fk_kds_ticket_items_exact_order_item
    FOREIGN KEY (tenant_id, restaurant_id, order_id, order_item_id)
    REFERENCES order_items (tenant_id, restaurant_id, order_id, id)
    ON DELETE CASCADE;


-- ============================================================
-- 7. KDS STATUS HISTORY OWNERSHIP
-- ============================================================

ALTER TABLE kds_status_history
    ADD COLUMN restaurant_id UUID;

UPDATE kds_status_history h
SET restaurant_id = kt.restaurant_id
FROM kds_tickets kt
WHERE kt.tenant_id = h.tenant_id
  AND kt.id = h.kds_ticket_id;

ALTER TABLE kds_status_history
    ALTER COLUMN restaurant_id SET NOT NULL;

ALTER TABLE kds_status_history
    DROP CONSTRAINT kds_status_history_kds_ticket_id_fkey;

ALTER TABLE kds_status_history
    DROP CONSTRAINT kds_status_history_changed_by_fkey;

ALTER TABLE kds_status_history
    ADD CONSTRAINT fk_kds_status_history_tenant_restaurant_ticket
    FOREIGN KEY (tenant_id, restaurant_id, kds_ticket_id)
    REFERENCES kds_tickets (tenant_id, restaurant_id, id)
    ON DELETE CASCADE;

ALTER TABLE kds_status_history
    ADD CONSTRAINT fk_kds_status_history_tenant_actor
    FOREIGN KEY (tenant_id, changed_by)
    REFERENCES users (tenant_id, id)
    ON DELETE SET NULL (changed_by);


-- ============================================================
-- 8. SUPPORTING INDEXES
-- ============================================================

CREATE INDEX idx_order_items_tenant_restaurant_order
    ON order_items (tenant_id, restaurant_id, order_id);

CREATE INDEX idx_order_item_addons_tenant_restaurant_item
    ON order_item_addons (tenant_id, restaurant_id, order_item_id);

CREATE INDEX idx_kds_ticket_items_exact_order
    ON kds_ticket_items (
        tenant_id,
        restaurant_id,
        order_id,
        kds_ticket_id,
        order_item_id
    );

CREATE INDEX idx_kds_status_history_tenant_restaurant_ticket
    ON kds_status_history (tenant_id, restaurant_id, kds_ticket_id);
