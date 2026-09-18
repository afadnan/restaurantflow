-- ============================================================
-- Rollback Migration: 000002
-- ============================================================

DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS tenants;

DROP TYPE IF EXISTS user_status;

DROP TYPE IF EXISTS user_role;