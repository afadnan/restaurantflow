-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000001
-- Purpose: PostgreSQL extensions and common functions
-- ============================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE EXTENSION IF NOT EXISTS citext;


-- ------------------------------------------------------------
-- updated_at trigger function
-- ------------------------------------------------------------

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


-- ------------------------------------------------------------
-- Tenant context helper
--
-- Application sets:
--
-- SET LOCAL app.tenant_id = 'tenant-uuid';
--
-- RLS policies use this value.
-- ------------------------------------------------------------

CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')::UUID;
$$;