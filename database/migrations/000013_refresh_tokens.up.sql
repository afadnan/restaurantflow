
-- ============================================================
-- RestaurantFlow.fun
-- Migration: 000013
-- Purpose: Secure refresh-token session storage
-- ============================================================


-- ------------------------------------------------------------
-- Ensure refresh-token users belong to the same tenant.
-- ------------------------------------------------------------

ALTER TABLE public.users
ADD CONSTRAINT users_tenant_id_id_key
UNIQUE (tenant_id, id);


-- ------------------------------------------------------------
-- REFRESH TOKENS
-- ------------------------------------------------------------

CREATE TABLE public.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL,

    user_id UUID NOT NULL,

    -- SHA-256 digest of the opaque refresh token.
    -- Raw refresh tokens must never be stored.
    token_hash BYTEA NOT NULL UNIQUE,

    -- Groups tokens belonging to one device/session.
    family_id UUID NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,

    -- Set when rotated, logged out, or revoked.
    revoked_at TIMESTAMPTZ,

    -- Points to the replacement token after rotation.
    replaced_by_token_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Supports same-tenant replacement-token references.
    CONSTRAINT refresh_tokens_tenant_id_id_key
        UNIQUE (tenant_id, id),

    CONSTRAINT refresh_tokens_hash_length_check
        CHECK (octet_length(token_hash) = 32),

    CONSTRAINT refresh_tokens_expiry_check
        CHECK (expires_at > created_at),

    CONSTRAINT refresh_tokens_not_self_referencing
        CHECK (
            replaced_by_token_id IS NULL
            OR replaced_by_token_id <> id
        ),

    -- A token must belong to a real user in the same tenant.
    CONSTRAINT refresh_tokens_user_tenant_fk
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES public.users (tenant_id, id)
        ON DELETE CASCADE,

    -- Replacement tokens must belong to the same tenant.
    CONSTRAINT refresh_tokens_replacement_fk
        FOREIGN KEY (tenant_id, replaced_by_token_id)
        REFERENCES public.refresh_tokens (tenant_id, id)
);


-- ------------------------------------------------------------
-- INDEXES
-- ------------------------------------------------------------

CREATE INDEX idx_refresh_tokens_tenant_user
    ON public.refresh_tokens (tenant_id, user_id);

CREATE INDEX idx_refresh_tokens_tenant_family
    ON public.refresh_tokens (tenant_id, family_id);

CREATE INDEX idx_refresh_tokens_expiry
    ON public.refresh_tokens (expires_at);


-- ------------------------------------------------------------
-- ROW LEVEL SECURITY
-- ------------------------------------------------------------

ALTER TABLE public.refresh_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY refresh_tokens_tenant_isolation
ON public.refresh_tokens
USING (
    tenant_id = current_tenant_id()
)
WITH CHECK (
    tenant_id = current_tenant_id()
);


-- ------------------------------------------------------------
-- APPLICATION PERMISSIONS
-- ------------------------------------------------------------

GRANT SELECT, INSERT, UPDATE
ON TABLE public.refresh_tokens
TO restaurantflow_app;
