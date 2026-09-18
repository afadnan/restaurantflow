-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000004
-- Purpose: Menu and ingredient inventory
-- ============================================================


-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE menu_item_status AS ENUM (
    'active',
    'inactive',
    'sold_out'
);


CREATE TYPE inventory_unit AS ENUM (
    'piece',
    'gram',
    'kilogram',
    'milliliter',
    'liter',
    'ounce',
    'pound'
);


-- ============================================================
-- MENU CATEGORIES
-- ============================================================

CREATE TABLE menu_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL,

    description TEXT,

    sort_order INTEGER NOT NULL DEFAULT 0,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (restaurant_id, name)
);


CREATE INDEX idx_menu_categories_tenant
    ON menu_categories(tenant_id);

CREATE INDEX idx_menu_categories_restaurant
    ON menu_categories(restaurant_id);

CREATE INDEX idx_menu_categories_sort
    ON menu_categories(restaurant_id, sort_order);


CREATE TRIGGER menu_categories_set_updated_at
BEFORE UPDATE ON menu_categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- MENU ITEMS
-- ============================================================

CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    category_id UUID NOT NULL
        REFERENCES menu_categories(id)
        ON DELETE RESTRICT,

    name TEXT NOT NULL,

    description TEXT,

    sku TEXT,

    price NUMERIC(12, 2) NOT NULL
        CHECK (price >= 0),

    status menu_item_status NOT NULL DEFAULT 'active',

    image_url TEXT,

    sort_order INTEGER NOT NULL DEFAULT 0,

    is_vegetarian BOOLEAN NOT NULL DEFAULT FALSE,

    is_vegan BOOLEAN NOT NULL DEFAULT FALSE,

    is_available BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (restaurant_id, sku)
);


CREATE INDEX idx_menu_items_tenant
    ON menu_items(tenant_id);

CREATE INDEX idx_menu_items_restaurant
    ON menu_items(restaurant_id);

CREATE INDEX idx_menu_items_category
    ON menu_items(category_id);

CREATE INDEX idx_menu_items_status
    ON menu_items(restaurant_id, status);


CREATE TRIGGER menu_items_set_updated_at
BEFORE UPDATE ON menu_items
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- MENU ITEM ADD-ONS
-- ============================================================

CREATE TABLE menu_addons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL,

    description TEXT,

    price NUMERIC(12, 2) NOT NULL DEFAULT 0
        CHECK (price >= 0),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_menu_addons_tenant
    ON menu_addons(tenant_id);

CREATE INDEX idx_menu_addons_restaurant
    ON menu_addons(restaurant_id);


CREATE TRIGGER menu_addons_set_updated_at
BEFORE UPDATE ON menu_addons
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- MENU ITEM ↔ ADD-ON
-- ============================================================

CREATE TABLE menu_item_addons (
    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    menu_item_id UUID NOT NULL
        REFERENCES menu_items(id)
        ON DELETE CASCADE,

    addon_id UUID NOT NULL
        REFERENCES menu_addons(id)
        ON DELETE CASCADE,

    is_required BOOLEAN NOT NULL DEFAULT FALSE,

    max_quantity INTEGER NOT NULL DEFAULT 1
        CHECK (max_quantity > 0),

    PRIMARY KEY (menu_item_id, addon_id)
);


CREATE INDEX idx_menu_item_addons_tenant
    ON menu_item_addons(tenant_id);


-- ============================================================
-- INGREDIENTS
-- ============================================================

CREATE TABLE ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL,

    sku TEXT,

    unit inventory_unit NOT NULL,

    description TEXT,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (restaurant_id, name),

    UNIQUE (restaurant_id, sku)
);


CREATE INDEX idx_ingredients_tenant
    ON ingredients(tenant_id);

CREATE INDEX idx_ingredients_restaurant
    ON ingredients(restaurant_id);


CREATE TRIGGER ingredients_set_updated_at
BEFORE UPDATE ON ingredients
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- INVENTORY
-- ============================================================

CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    ingredient_id UUID NOT NULL
        REFERENCES ingredients(id)
        ON DELETE CASCADE,

    quantity NUMERIC(14, 3) NOT NULL DEFAULT 0
        CHECK (quantity >= 0),

    low_stock_threshold NUMERIC(14, 3) NOT NULL DEFAULT 0
        CHECK (low_stock_threshold >= 0),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (restaurant_id, ingredient_id)
);


CREATE INDEX idx_inventory_tenant
    ON inventory(tenant_id);

CREATE INDEX idx_inventory_restaurant
    ON inventory(restaurant_id);

CREATE INDEX idx_inventory_low_stock
    ON inventory(restaurant_id, quantity, low_stock_threshold);


-- ============================================================
-- MENU ITEM ↔ INGREDIENT RECIPE
-- ============================================================

CREATE TABLE menu_item_ingredients (
    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    menu_item_id UUID NOT NULL
        REFERENCES menu_items(id)
        ON DELETE CASCADE,

    ingredient_id UUID NOT NULL
        REFERENCES ingredients(id)
        ON DELETE RESTRICT,

    quantity_required NUMERIC(14, 3) NOT NULL
        CHECK (quantity_required > 0),

    PRIMARY KEY (menu_item_id, ingredient_id)
);


CREATE INDEX idx_menu_item_ingredients_tenant
    ON menu_item_ingredients(tenant_id);


-- ============================================================
-- RLS
-- ============================================================

ALTER TABLE menu_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE menu_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE menu_addons ENABLE ROW LEVEL SECURITY;
ALTER TABLE menu_item_addons ENABLE ROW LEVEL SECURITY;
ALTER TABLE ingredients ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory ENABLE ROW LEVEL SECURITY;
ALTER TABLE menu_item_ingredients ENABLE ROW LEVEL SECURITY;


CREATE POLICY menu_categories_tenant_isolation
ON menu_categories
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY menu_items_tenant_isolation
ON menu_items
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY menu_addons_tenant_isolation
ON menu_addons
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY menu_item_addons_tenant_isolation
ON menu_item_addons
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY ingredients_tenant_isolation
ON ingredients
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY inventory_tenant_isolation
ON inventory
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY menu_item_ingredients_tenant_isolation
ON menu_item_ingredients
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());