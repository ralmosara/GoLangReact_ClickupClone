-- 033_sessions.up.sql
--
-- Session tracking for "active devices" + "sign out all other devices".
-- The auth model stays JWT-bearer (stateless on the wire), but every
-- issued token carries a `jti` (JWT ID) which is checked against this
-- table on each authenticated request. Deleting a row revokes the
-- corresponding token immediately.
--
-- Rows are inserted by /auth/login and /auth/mfa/verify and pruned by:
--   - explicit revoke via DELETE /me/sessions/{jti}
--   - "sign out all other devices" via /me/sessions/revoke-others
--   - the auth middleware, when expires_at has passed (lazy GC)
--
-- The `last_seen_at` column is updated by the middleware on every
-- request, debounced to once per minute — keeps the table hot writes
-- bounded even under sustained traffic.

CREATE TABLE IF NOT EXISTS user_sessions (
    -- jti — the JWT ID stamped into the token at issue time. Stored as
    -- UUID rather than text so the existing pgx UUID codec can scan it
    -- back without a string round-trip.
    id            UUID        PRIMARY KEY,
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Context for the "Active sessions" UI. Browser shows
    -- "Chrome on macOS — Bangalore — 2 minutes ago" using these.
    user_agent    TEXT        NOT NULL DEFAULT '',
    ip            INET,                                           -- nullable; behind some LBs we won't have one

    -- The label is the user-visible name. Defaults to a UA-derived
    -- string at issue time but the user could rename it later.
    label         TEXT        NOT NULL DEFAULT '',

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- expires_at mirrors the JWT exp claim. Lets the middleware GC
    -- expired sessions during normal traffic without a separate cron.
    expires_at    TIMESTAMPTZ NOT NULL
);

-- Per-user lookup is the dominant access pattern (the Active Sessions
-- UI lists only the current user's rows). The composite index orders
-- by last_seen DESC so the most-active devices surface first.
CREATE INDEX IF NOT EXISTS user_sessions_user_id_seen_idx
    ON user_sessions (user_id, last_seen_at DESC);

-- Used by the GC sweep — find all rows past their exp claim.
CREATE INDEX IF NOT EXISTS user_sessions_expires_idx
    ON user_sessions (expires_at);
