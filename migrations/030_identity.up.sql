-- 030_identity.up.sql
-- Phase 2 enterprise-identity tables:
--   * mfa_secrets       — per-user TOTP secret + enrolled flag
--   * mfa_recovery_codes — single-use recovery codes (hashed)
--   * oauth_identities   — links a user to an external IdP subject (Google, etc.)
-- Together these support TOTP MFA on every login and JIT user creation via
-- OIDC SSO. The shape is provider-agnostic so adding Microsoft / Okta later
-- only requires a new `provider` value, not a new table.

CREATE TABLE IF NOT EXISTS mfa_secrets (
    user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret       TEXT NOT NULL,            -- base32-encoded TOTP shared secret
    enrolled_at  TIMESTAMPTZ,              -- NULL until the user verifies a code
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recovery codes are stored hashed (sha256). Code verification looks up by
-- (user_id, code_hash) and atomically marks consumed_at. Each code is
-- single-use; the UI must encourage the user to print the full set when
-- enrolling.
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   BYTEA NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_user
    ON mfa_recovery_codes(user_id) WHERE consumed_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_mfa_recovery_codes_user_hash
    ON mfa_recovery_codes(user_id, code_hash);

-- oauth_identities maps an external IdP subject (e.g. a Google `sub`) to a
-- user. Multiple identities per user are allowed (a user can link Google +
-- Microsoft to the same account); a single (provider, subject) pair maps to
-- exactly one user.
CREATE TABLE IF NOT EXISTS oauth_identities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,            -- 'google' | 'microsoft' | 'okta-oidc'
    subject     TEXT NOT NULL,            -- the IdP `sub` claim
    email       TEXT,
    raw_profile JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oauth_identities_user ON oauth_identities(user_id);
