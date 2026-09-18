-- ============================================================
-- Rollback Migration: 000005
-- ============================================================

DROP TABLE IF EXISTS order_item_addons;

DROP TABLE IF EXISTS order_items;

DROP TABLE IF EXISTS orders;

DROP TYPE IF EXISTS payment_status;

DROP TYPE IF EXISTS order_type;

DROP TYPE IF EXISTS order_status;