
-- ============================================================
-- Migration 000011: Menu Add-on Ownership Hardening
-- ============================================================
-- Enforce tenant and restaurant ownership across:
--   1. restaurants
--   2. menu_items
--   3. menu_addons
--   4. menu_item_addons
--
-- Existing inconsistent data will cause this migration to fail.
-- Do not automatically repair or delete inconsistent records.
-- ============================================================


-- ============================================================
-- 1. COMPOSITE UNIQUE CONSTRAINTS
-- ============================================================

-- Required for composite foreign keys referencing restaurants.
ALTER TABLE restaurants
    ADD CONSTRAINT uq_restaurants_tenant_id_id
    UNIQUE (tenant_id, id);

-- Required for scoped item references.
ALTER TABLE menu_items
    ADD CONSTRAINT uq_menu_items_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);

-- Required for scoped add-on references.
ALTER TABLE menu_addons
    ADD CONSTRAINT uq_menu_addons_tenant_restaurant_id
    UNIQUE (tenant_id, restaurant_id, id);


-- ============================================================
-- 2. ENFORCE RESTAURANT OWNERSHIP
-- ============================================================

-- A menu item must belong to a restaurant in the same tenant.
ALTER TABLE menu_items
    ADD CONSTRAINT fk_menu_items_tenant_restaurant
    FOREIGN KEY (tenant_id, restaurant_id)
    REFERENCES restaurants (tenant_id, id);

-- An add-on must belong to a restaurant in the same tenant.
ALTER TABLE menu_addons
    ADD CONSTRAINT fk_menu_addons_tenant_restaurant
    FOREIGN KEY (tenant_id, restaurant_id)
    REFERENCES restaurants (tenant_id, id);


-- ============================================================
-- 3. EXTEND ITEM-ADDON ASSOCIATIONS
-- ============================================================

-- Existing association rows do not store restaurant identity.
ALTER TABLE menu_item_addons
    ADD COLUMN restaurant_id UUID;

-- Backfill restaurant identity from the associated menu item.
UPDATE menu_item_addons AS mia
SET restaurant_id = mi.restaurant_id
FROM menu_items AS mi
WHERE mi.id = mia.menu_item_id;

-- Every association must have a restaurant.
ALTER TABLE menu_item_addons
    ALTER COLUMN restaurant_id SET NOT NULL;


-- ============================================================
-- 4. ENFORCE ITEM-ADDON OWNERSHIP
-- ============================================================

-- The item must belong to the association tenant and restaurant.
ALTER TABLE menu_item_addons
    ADD CONSTRAINT fk_menu_item_addons_item_scope
    FOREIGN KEY (tenant_id, restaurant_id, menu_item_id)
    REFERENCES menu_items (tenant_id, restaurant_id, id);

-- The add-on must belong to the same tenant and restaurant.
ALTER TABLE menu_item_addons
    ADD CONSTRAINT fk_menu_item_addons_addon_scope
    FOREIGN KEY (tenant_id, restaurant_id, addon_id)
    REFERENCES menu_addons (tenant_id, restaurant_id, id);


-- ============================================================
-- 5. INDEXES
-- ============================================================

CREATE INDEX idx_menu_item_addons_scope
    ON menu_item_addons (
        tenant_id,
        restaurant_id,
        menu_item_id
    );
