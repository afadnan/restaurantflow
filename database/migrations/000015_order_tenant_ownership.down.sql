-- ============================================================
-- Migration 000015: Rollback order and KDS ownership hardening
-- ============================================================


-- ============================================================
-- 1. DROP SUPPORTING INDEXES
-- ============================================================

DROP INDEX IF EXISTS idx_order_items_tenant_restaurant_order;
DROP INDEX IF EXISTS idx_order_item_addons_tenant_restaurant_item;
DROP INDEX IF EXISTS idx_kds_ticket_items_exact_order;
DROP INDEX IF EXISTS idx_kds_status_history_tenant_restaurant_ticket;


-- ============================================================
-- 2. DROP NEW KDS STATUS HISTORY FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_status_history
    DROP CONSTRAINT fk_kds_status_history_tenant_restaurant_ticket;

ALTER TABLE kds_status_history
    DROP CONSTRAINT fk_kds_status_history_tenant_actor;


-- ============================================================
-- 3. DROP NEW KDS TICKET ITEM FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_ticket_items
    DROP CONSTRAINT fk_kds_ticket_items_exact_order_ticket;

ALTER TABLE kds_ticket_items
    DROP CONSTRAINT fk_kds_ticket_items_exact_order_item;


-- ============================================================
-- 4. DROP NEW ORDER ITEM ADD-ON FOREIGN KEYS
-- ============================================================

ALTER TABLE order_item_addons
    DROP CONSTRAINT fk_order_item_addons_tenant_restaurant_item;

ALTER TABLE order_item_addons
    DROP CONSTRAINT fk_order_item_addons_tenant_restaurant_addon;


-- ============================================================
-- 5. DROP NEW ORDER ITEM FOREIGN KEYS
-- ============================================================

ALTER TABLE order_items
    DROP CONSTRAINT fk_order_items_tenant_restaurant_order;

ALTER TABLE order_items
    DROP CONSTRAINT fk_order_items_tenant_restaurant_menu_item;


-- ============================================================
-- 6. DROP NEW KDS TICKET FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_tickets
    DROP CONSTRAINT fk_kds_tickets_tenant_restaurant_order;

ALTER TABLE kds_tickets
    DROP CONSTRAINT fk_kds_tickets_tenant_restaurant;


-- ============================================================
-- 7. DROP NEW ORDER FOREIGN KEYS
-- ============================================================

ALTER TABLE orders
    DROP CONSTRAINT fk_orders_tenant_restaurant_table;

ALTER TABLE orders
    DROP CONSTRAINT fk_orders_tenant_customer;

ALTER TABLE orders
    DROP CONSTRAINT fk_orders_tenant_restaurant;


-- ============================================================
-- 8. DROP NEW RESTAURANT TABLE FOREIGN KEY
-- ============================================================

ALTER TABLE restaurant_tables
    DROP CONSTRAINT fk_restaurant_tables_tenant_restaurant;


-- ============================================================
-- 9. DROP NEW COMPOSITE UNIQUE CONSTRAINTS
-- ============================================================

ALTER TABLE kds_tickets
    DROP CONSTRAINT uq_kds_tickets_tenant_restaurant_order;

ALTER TABLE kds_tickets
    DROP CONSTRAINT uq_kds_tickets_tenant_restaurant_id;

ALTER TABLE order_items
    DROP CONSTRAINT uq_order_items_tenant_restaurant_order_item;

ALTER TABLE order_items
    DROP CONSTRAINT uq_order_items_tenant_restaurant_id;

ALTER TABLE orders
    DROP CONSTRAINT uq_orders_tenant_restaurant_id;

ALTER TABLE restaurant_tables
    DROP CONSTRAINT uq_restaurant_tables_tenant_restaurant_id;


-- ============================================================
-- 10. RESTORE ORIGINAL RESTAURANT TABLE FOREIGN KEY
-- ============================================================

ALTER TABLE restaurant_tables
    ADD CONSTRAINT restaurant_tables_restaurant_id_fkey
    FOREIGN KEY (restaurant_id)
    REFERENCES restaurants(id)
    ON DELETE CASCADE;


-- ============================================================
-- 11. RESTORE ORIGINAL ORDER FOREIGN KEYS
-- ============================================================

ALTER TABLE orders
    ADD CONSTRAINT orders_restaurant_id_fkey
    FOREIGN KEY (restaurant_id)
    REFERENCES restaurants(id)
    ON DELETE RESTRICT;

ALTER TABLE orders
    ADD CONSTRAINT orders_table_id_fkey
    FOREIGN KEY (table_id)
    REFERENCES restaurant_tables(id)
    ON DELETE SET NULL;

ALTER TABLE orders
    ADD CONSTRAINT orders_customer_id_fkey
    FOREIGN KEY (customer_id)
    REFERENCES users(id)
    ON DELETE SET NULL;


-- ============================================================
-- 12. RESTORE ORIGINAL ORDER ITEM FOREIGN KEYS
-- ============================================================

ALTER TABLE order_items
    ADD CONSTRAINT order_items_order_id_fkey
    FOREIGN KEY (order_id)
    REFERENCES orders(id)
    ON DELETE CASCADE;

ALTER TABLE order_items
    ADD CONSTRAINT order_items_menu_item_id_fkey
    FOREIGN KEY (menu_item_id)
    REFERENCES menu_items(id)
    ON DELETE RESTRICT;


-- ============================================================
-- 13. RESTORE ORIGINAL ORDER ITEM ADD-ON FOREIGN KEYS
-- ============================================================

ALTER TABLE order_item_addons
    ADD CONSTRAINT order_item_addons_order_item_id_fkey
    FOREIGN KEY (order_item_id)
    REFERENCES order_items(id)
    ON DELETE CASCADE;

ALTER TABLE order_item_addons
    ADD CONSTRAINT order_item_addons_addon_id_fkey
    FOREIGN KEY (addon_id)
    REFERENCES menu_addons(id)
    ON DELETE SET NULL;


-- ============================================================
-- 14. RESTORE ORIGINAL KDS TICKET FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_tickets
    ADD CONSTRAINT kds_tickets_restaurant_id_fkey
    FOREIGN KEY (restaurant_id)
    REFERENCES restaurants(id)
    ON DELETE RESTRICT;

ALTER TABLE kds_tickets
    ADD CONSTRAINT kds_tickets_order_id_fkey
    FOREIGN KEY (order_id)
    REFERENCES orders(id)
    ON DELETE CASCADE;


-- ============================================================
-- 15. RESTORE ORIGINAL KDS TICKET ITEM FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_ticket_items
    ADD CONSTRAINT kds_ticket_items_kds_ticket_id_fkey
    FOREIGN KEY (kds_ticket_id)
    REFERENCES kds_tickets(id)
    ON DELETE CASCADE;

ALTER TABLE kds_ticket_items
    ADD CONSTRAINT kds_ticket_items_order_item_id_fkey
    FOREIGN KEY (order_item_id)
    REFERENCES order_items(id)
    ON DELETE CASCADE;


-- ============================================================
-- 16. RESTORE ORIGINAL KDS STATUS HISTORY FOREIGN KEYS
-- ============================================================

ALTER TABLE kds_status_history
    ADD CONSTRAINT kds_status_history_kds_ticket_id_fkey
    FOREIGN KEY (kds_ticket_id)
    REFERENCES kds_tickets(id)
    ON DELETE CASCADE;

ALTER TABLE kds_status_history
    ADD CONSTRAINT kds_status_history_changed_by_fkey
    FOREIGN KEY (changed_by)
    REFERENCES users(id)
    ON DELETE SET NULL;


-- ============================================================
-- 17. REMOVE MIGRATION-ADDED COLUMNS
-- ============================================================

ALTER TABLE kds_ticket_items
    DROP COLUMN order_id,
    DROP COLUMN restaurant_id;

ALTER TABLE kds_status_history
    DROP COLUMN restaurant_id;

ALTER TABLE order_item_addons
    DROP COLUMN restaurant_id;

ALTER TABLE order_items
    DROP COLUMN restaurant_id;
