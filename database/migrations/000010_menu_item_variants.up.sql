
-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000010
-- Purpose: Menu item variants
-- ============================================================

-- Ensure a variant's tenant and restaurant match its parent item.
ALTER TABLE menu_items
    ADD CONSTRAINT menu_items_tenant_restaurant_id_key
    UNIQUE (tenant_id, restaurant_id, id);

CREATE TABLE menu_item_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    menu_item_id UUID NOT NULL,

    name TEXT NOT NULL
        CHECK (
            length(btrim(name)) BETWEEN 1 AND 200
        ),

    -- Signed adjustment relative to the base menu item price.
    -- Negative values are valid for cheaper variants.
    price_delta NUMERIC(12, 2) NOT NULL DEFAULT 0,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT menu_item_variants_item_fk
        FOREIGN KEY (tenant_id, restaurant_id, menu_item_id)
        REFERENCES menu_items (tenant_id, restaurant_id, id)
        ON DELETE CASCADE,

    CONSTRAINT menu_item_variants_item_name_key
        UNIQUE (menu_item_id, name)
);

CREATE INDEX idx_menu_item_variants_tenant
    ON menu_item_variants (tenant_id);

CREATE INDEX idx_menu_item_variants_restaurant_item
    ON menu_item_variants (
        tenant_id,
        restaurant_id,
        menu_item_id,
        is_active
    );

CREATE TRIGGER menu_item_variants_set_updated_at
BEFORE UPDATE ON menu_item_variants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE menu_item_variants ENABLE ROW LEVEL SECURITY;

CREATE POLICY menu_item_variants_tenant_isolation
ON menu_item_variants
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());
