-- The application writes outbox events inside business transactions.
-- Cross-tenant claiming and updates remain restricted to restaurantflow_worker.

GRANT INSERT
ON TABLE public.outbox_events
TO restaurantflow_app;
