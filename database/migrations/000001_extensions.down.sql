-- ============================================================
-- Rollback Migration: 000001
-- ============================================================

DROP FUNCTION IF EXISTS current_tenant_id();

DROP FUNCTION IF EXISTS set_updated_at();

DROP EXTENSION IF EXISTS citext;

DROP EXTENSION IF EXISTS pgcrypto;