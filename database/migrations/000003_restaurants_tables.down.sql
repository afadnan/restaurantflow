-- ============================================================
-- Rollback Migration: 000003
-- ============================================================

DROP TABLE IF EXISTS restaurant_tables;

DROP TABLE IF EXISTS restaurants;

DROP TYPE IF EXISTS restaurant_table_status;

DROP TYPE IF EXISTS restaurant_status;