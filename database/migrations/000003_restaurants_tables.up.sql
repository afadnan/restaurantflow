-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000003
-- Purpose: Restaurants and dine-in tables
-- ============================================================


-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE restaurant_status AS ENUM (
    'active',
    'inactive',
    'suspended'
);


CREATE TYPE restaurant_table_status AS ENUM (
    'available',
    'occupied',
    'reserved',
    'inactive'
);


-- ============================================================
-- RESTAURANTS
-- ============================================================

CREATE TABLE restaurants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL,

    slug CITEXT NOT NULL,

    description TEXT,

    phone TEXT,

    email CITEXT,

    address_line_1 TEXT,

    address_line_2 TEXT,

    city TEXT,

    state TEXT,

    postal_code TEXT,

    country TEXT NOT NULL DEFAULT 'India',

    latitude NUMERIC(10, 7),

    longitude NUMERIC(10, 7),

    status restaurant_status NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, slug)
);


CREATE INDEX idx_restaurants_tenant_id
    ON restaurants(tenant_id);

CREATE INDEX idx_restaurants_tenant_status
    ON restaurants(tenant_id, status);


CREATE TRIGGER restaurants_set_updated_at
BEFORE UPDATE ON restaurants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- RESTAURANT TABLES
-- ============================================================

CREATE TABLE restaurant_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    restaurant_id UUID NOT NULL
        REFERENCES restaurants(id)
        ON DELETE CASCADE,

    table_number TEXT NOT NULL,

    display_name TEXT,

    capacity INTEGER NOT NULL DEFAULT 2
        CHECK (capacity > 0),

    status restaurant_table_status NOT NULL DEFAULT 'available',

    qr_token TEXT NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (restaurant_id, table_number)
);


CREATE INDEX idx_restaurant_tables_tenant_id
    ON restaurant_tables(tenant_id);

CREATE INDEX idx_restaurant_tables_restaurant_id
    ON restaurant_tables(restaurant_id);

CREATE INDEX idx_restaurant_tables_status
    ON restaurant_tables(tenant_id, restaurant_id, status);


CREATE TRIGGER restaurant_tables_set_updated_at
BEFORE UPDATE ON restaurant_tables
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- ROW LEVEL SECURITY
-- ============================================================

ALTER TABLE restaurants ENABLE ROW LEVEL SECURITY;

ALTER TABLE restaurant_tables ENABLE ROW LEVEL SECURITY;


CREATE POLICY restaurants_tenant_isolation
ON restaurants
USING (
    tenant_id = current_tenant_id()
)
WITH CHECK (
    tenant_id = current_tenant_id()
);


CREATE POLICY restaurant_tables_tenant_isolation
ON restaurant_tables
USING (
    tenant_id = current_tenant_id()
)
WITH CHECK (
    tenant_id = current_tenant_id()
);