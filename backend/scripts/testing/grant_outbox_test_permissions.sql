-- RestaurantFlow.fun
-- Dedicated Outbox integration database permissions.
-- Run after applying the database migrations.
-- Run using the database owner or an authorized administrator.

DO $$
BEGIN
    IF current_database() <> 'restaurantflow_outbox_test' THEN
        RAISE EXCEPTION
            'Refusing to configure Outbox test permissions on database %',
            current_database();
    END IF;

    IF to_regclass('public.outbox_events') IS NULL THEN
        RAISE EXCEPTION
            'Outbox schema is missing. Apply migrations first.';
    END IF;
END
$$;

BEGIN;

GRANT USAGE ON SCHEMA public
TO restaurantflow_app;

GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA public
TO restaurantflow_app;

GRANT USAGE, SELECT
ON ALL SEQUENCES IN SCHEMA public
TO restaurantflow_app;

GRANT USAGE ON SCHEMA public
TO restaurantflow_worker;

GRANT SELECT, UPDATE
ON TABLE public.outbox_events
TO restaurantflow_worker;

COMMIT;