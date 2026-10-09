CREATE POLICY outbox_events_worker_select
ON outbox_events
FOR SELECT
TO restaurantflow_worker
USING (true);

CREATE POLICY outbox_events_worker_update
ON outbox_events
FOR UPDATE
TO restaurantflow_worker
USING (true)
WITH CHECK (true);