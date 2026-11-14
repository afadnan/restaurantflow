
-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000013
-- Purpose: Rollback refresh-token session storage
-- ============================================================


-- ------------------------------------------------------------
-- APPLICATION PERMISSIONS
-- ------------------------------------------------------------

REVOKE SELECT, INSERT, UPDATE
ON TABLE public.refresh_tokens
FROM restaurantflow_app;


-- ------------------------------------------------------------
-- DROP REFRESH TOKENS
-- ------------------------------------------------------------

DROP TABLE IF EXISTS public.refresh_tokens;


-- ------------------------------------------------------------
-- REMOVE COMPOSITE USER CONSTRAINT
-- ------------------------------------------------------------

ALTER TABLE public.users
DROP CONSTRAINT IF EXISTS users_tenant_id_id_key;
