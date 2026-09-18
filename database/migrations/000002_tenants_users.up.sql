-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000002
-- Purpose: Tenants and users
-- ============================================================


-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE user_role AS ENUM (
    'owner',
    'admin',
    'manager',
    'staff',
    'kitchen',
    'cashier'
);


CREATE TYPE user_status AS ENUM (
    'active',
    'inactive',
    'suspended'
);


-- ============================================================
-- TENANTS
-- ============================================================

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,

    slug CITEXT NOT NULL UNIQUE,

    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'inactive')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_tenants_status
    ON tenants(status);


CREATE TRIGGER tenants_set_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- USERS
-- ============================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    email CITEXT NOT NULL,

    phone TEXT,

    password_hash TEXT NOT NULL,

    first_name TEXT NOT NULL,

    last_name TEXT,

    role user_role NOT NULL DEFAULT 'staff',

    status user_status NOT NULL DEFAULT 'active',

    last_login_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, email)
);


CREATE INDEX idx_users_tenant_id
    ON users(tenant_id);

CREATE INDEX idx_users_tenant_role
    ON users(tenant_id, role);

CREATE INDEX idx_users_tenant_status
    ON users(tenant_id, status);


CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- ROW LEVEL SECURITY
-- ============================================================

ALTER TABLE users ENABLE ROW LEVEL SECURITY;


CREATE POLICY users_tenant_isolation
ON users
USING (
    tenant_id = current_tenant_id()
)
WITH CHECK (
    tenant_id = current_tenant_id()
);