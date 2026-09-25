-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000007
-- Purpose: Remove transactional outbox
-- ============================================================

DROP TABLE IF EXISTS outbox_events;