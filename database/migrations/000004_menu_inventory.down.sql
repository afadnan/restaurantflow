-- ============================================================
-- Rollback Migration: 000004
-- ============================================================

DROP TABLE IF EXISTS menu_item_ingredients;

DROP TABLE IF EXISTS inventory;

DROP TABLE IF EXISTS ingredients;

DROP TABLE IF EXISTS menu_item_addons;

DROP TABLE IF EXISTS menu_addons;

DROP TABLE IF EXISTS menu_items;

DROP TABLE IF EXISTS menu_categories;

DROP TYPE IF EXISTS inventory_unit;

DROP TYPE IF EXISTS menu_item_status;