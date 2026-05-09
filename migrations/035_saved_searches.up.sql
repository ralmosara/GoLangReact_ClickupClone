-- 035_saved_searches.up.sql
--
-- Per-user saved searches. The product treats this like browser bookmarks:
-- "my open tasks due this week" lives as a row a user can click to re-run
-- the query later. Sharing isn't in scope yet (no `is_shared` flag) — the
-- shape leaves room for it (a future migration can add `workspace_id`
-- nullable + `is_shared`).
--
-- The `query` JSONB holds the canonical, structured query the UI sends
-- back to the server. Stored as JSONB rather than a free-text string so
-- we can index/filter on it server-side later (e.g. "every saved search
-- that filters on assignee X"), and so the schema stays version-tolerant
-- as the FilterClause type grows new ops.

CREATE TABLE IF NOT EXISTS saved_searches (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    workspace_id  UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    -- entity_type narrows the query to one search facet (task/doc/comment/message)
    -- or NULL for "all entities". Mirrors the SearchHit.entity_type vocabulary.
    entity_type   TEXT,
    -- Free-text search term. Optional — saves like "all overdue tasks
    -- assigned to me" carry filters but no q.
    query_text    TEXT        NOT NULL DEFAULT '',
    -- Structured filter payload. Shape mirrors the FilterClause + SortField
    -- types the UI already speaks to view configs. JSON here so the server
    -- doesn't need a typed schema migration each time the UI gains a filter.
    filters       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    pinned        BOOLEAN     NOT NULL DEFAULT FALSE,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A user can't have two saved searches with the same name in the
    -- same workspace — keeps the picker dropdown sane and lets the UI
    -- treat name + workspace as a stable display key.
    UNIQUE (user_id, workspace_id, name)
);

CREATE INDEX IF NOT EXISTS saved_searches_user_workspace_idx
    ON saved_searches (user_id, workspace_id, pinned DESC, last_used_at DESC NULLS LAST);
