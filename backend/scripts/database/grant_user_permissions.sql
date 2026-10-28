-- Application role permissions for user authentication.
-- Run as the database owner or an authorized administrator.
-- The restaurantflow_app role must already exist.

GRANT SELECT, INSERT, UPDATE
ON TABLE public.users
TO restaurantflow_app;