
-- ============================================================
-- Migration 000011: Rollback Menu Add-on Ownership Hardening
-- ============================================================
-- Removes constraints and columns introduced by migration
-- 000011 only.
--
-- Warning:
-- Rolling back removes database-level tenant and restaurant
-- ownership enforcement introduced by this migration.
-- ============================================================


-- ============================================================
-- 1. REMOVE ASSOCIATION INDEX
-- ============================================================

DROP INDEX IF EXISTS idx_menu_item_addons_scope;


-- ============================================================
-- 2. REMOVE ASSOCIATION FOREIGN KEYS
-- ============================================================

ALTER TABLE menu_item_addons
    DROP CONSTRAINT IF EXISTS fk_menu_item_addons_addon_scope;

ALTER TABLE menu_item_addons
    DROP CONSTRAINT IF EXISTS fk_menu_item_addons_item_scope;


-- ============================================================
-- 3. REMOVE ASSOCIATION RESTAURANT COLUMN
-- ============================================================

ALTER TABLE menu_item_addons
    DROP COLUMN IF EXISTS restaurant_id;


-- ============================================================
-- 4. REMOVE RESTAURANT OWNERSHIP FOREIGN KEYS
-- ============================================================

ALTER TABLE menu_addons
    DROP CONSTRAINT IF EXISTS fk_menu_addons_tenant_restaurant;

ALTER TABLE menu_items
    DROP CONSTRAINT IF EXISTS fk_menu_items_tenant_restaurant;


-- ============================================================
-- 5. REMOVE COMPOSITE UNIQUE CONSTRAINTS
-- ============================================================

ALTER TABLE menu_addons
    DROP CONSTRAINT IF EXISTS uq_menu_addons_tenant_restaurant_id;

ALTER TABLE menu_items
    DROP CONSTRAINT IF EXISTS uq_menu_items_tenant_restaurant_id;

ALTER TABLE restaurants
    DROP CONSTRAINT IF EXISTS uq_restaurants_tenant_id_id;
