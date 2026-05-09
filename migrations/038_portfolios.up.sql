-- 038_portfolios.up.sql
--
-- Portfolios are workspace-scoped collections of N spaces, used to roll
-- up status counts / due-date burndown / goal progress across multiple
-- projects. Executives use this to answer "how is the Mobile push
-- doing?" without drilling into every space.
--
-- Implementation is intentionally thin: a portfolio holds a name + a
-- description and is linked to spaces via an N:M join. All the rollup
-- math happens server-side at read-time off existing tables — there's
-- no denormalised counter to maintain.

CREATE TABLE IF NOT EXISTS portfolios (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    color         TEXT,
    owner_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_portfolios_workspace
    ON portfolios (workspace_id, created_at DESC);

CREATE TABLE IF NOT EXISTS portfolio_spaces (
    portfolio_id  UUID        NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    space_id      UUID        NOT NULL REFERENCES spaces(id)     ON DELETE CASCADE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (portfolio_id, space_id)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_spaces_space
    ON portfolio_spaces (space_id);
