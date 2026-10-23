
-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000010 rollback
-- Purpose: Remove menu item variants
-- ============================================================

DROP TABLE IF EXISTS menu_item_variants;

ALTER TABLE menu_items
    DROP CONSTRAINT IF EXISTS menu_items_tenant_restaurant_id_key;
