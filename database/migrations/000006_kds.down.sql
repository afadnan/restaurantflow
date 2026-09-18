-- ============================================================
-- Rollback Migration: 000006
-- ============================================================

DROP TABLE IF EXISTS kds_status_history;

DROP TABLE IF EXISTS kds_ticket_items;

DROP TABLE IF EXISTS kds_tickets;

DROP TYPE IF EXISTS kds_status;