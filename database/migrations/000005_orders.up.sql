-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000005
-- Purpose: Orders and order items
-- ============================================================


-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE order_status AS ENUM (
    'pending',
    'confirmed',
    'preparing',
    'ready',
    'completed',
    'cancelled'
);


CREATE TYPE order_type AS ENUM (
    'dine_in',
    'takeaway',
    'delivery'
);


CREATE TYPE payment_status AS ENUM (
    'pending',
    'paid',
    'failed',
    'refunded',
    'not_required'
);


-- ============================================================
-- ORDERS
-- ============================================================

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE RESTRICT,

    table_id UUID
        REFERENCES restaurant_tables(id)
        ON DELETE SET NULL,

    customer_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    order_number BIGSERIAL,

    order_type order_type NOT NULL DEFAULT 'dine_in',

    status order_status NOT NULL DEFAULT 'pending',

    payment_status payment_status NOT NULL DEFAULT 'pending',

    subtotal NUMERIC(12, 2) NOT NULL DEFAULT 0
        CHECK (subtotal >= 0),

    tax_amount NUMERIC(12, 2) NOT NULL DEFAULT 0
        CHECK (tax_amount >= 0),

    discount_amount NUMERIC(12, 2) NOT NULL DEFAULT 0
        CHECK (discount_amount >= 0),

    service_fee NUMERIC(12, 2) NOT NULL DEFAULT 0
        CHECK (service_fee >= 0),

    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0
    CHECK (total_amount >= 0),

    currency CHAR(3) NOT NULL DEFAULT 'INR'
    CHECK (currency ~ '^[A-Z]{3}$'),

    customer_name TEXT,

    customer_phone TEXT,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    completed_at TIMESTAMPTZ,

    cancelled_at TIMESTAMPTZ
);


CREATE INDEX idx_orders_tenant
    ON orders(tenant_id);

CREATE INDEX idx_orders_restaurant
    ON orders(restaurant_id);

CREATE INDEX idx_orders_table
    ON orders(table_id);

CREATE INDEX idx_orders_customer
    ON orders(customer_id);

CREATE INDEX idx_orders_status
    ON orders(tenant_id, restaurant_id, status);

CREATE INDEX idx_orders_created_at
    ON orders(tenant_id, restaurant_id, created_at DESC);


CREATE TRIGGER orders_set_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- ORDER ITEMS
-- ============================================================

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    order_id UUID NOT NULL
        REFERENCES orders(id)
        ON DELETE CASCADE,

    menu_item_id UUID NOT NULL
        REFERENCES menu_items(id)
        ON DELETE RESTRICT,

    item_name TEXT NOT NULL,

    unit_price NUMERIC(12, 2) NOT NULL
        CHECK (unit_price >= 0),

    quantity INTEGER NOT NULL
        CHECK (quantity > 0),

    subtotal NUMERIC(12, 2) NOT NULL
        CHECK (subtotal >= 0),

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_order_items_tenant
    ON order_items(tenant_id);

CREATE INDEX idx_order_items_order
    ON order_items(order_id);

CREATE INDEX idx_order_items_menu_item
    ON order_items(menu_item_id);


-- ============================================================
-- ORDER ITEM ADD-ONS
-- ============================================================

CREATE TABLE order_item_addons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    order_item_id UUID NOT NULL
        REFERENCES order_items(id)
        ON DELETE CASCADE,

    addon_id UUID
        REFERENCES menu_addons(id)
        ON DELETE SET NULL,

    addon_name TEXT NOT NULL,

    unit_price NUMERIC(12, 2) NOT NULL
        CHECK (unit_price >= 0),

    quantity INTEGER NOT NULL DEFAULT 1
        CHECK (quantity > 0),

    subtotal NUMERIC(12, 2) NOT NULL
        CHECK (subtotal >= 0)
);


CREATE INDEX idx_order_item_addons_tenant
    ON order_item_addons(tenant_id);

CREATE INDEX idx_order_item_addons_order_item
    ON order_item_addons(order_item_id);


-- ============================================================
-- RLS
-- ============================================================

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

ALTER TABLE order_items ENABLE ROW LEVEL SECURITY;

ALTER TABLE order_item_addons ENABLE ROW LEVEL SECURITY;


CREATE POLICY orders_tenant_isolation
ON orders
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY order_items_tenant_isolation
ON order_items
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());


CREATE POLICY order_item_addons_tenant_isolation
ON order_item_addons
USING (tenant_id = current_tenant_id())
WITH CHECK (tenant_id = current_tenant_id());