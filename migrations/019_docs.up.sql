CREATE TABLE docs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    space_id        UUID REFERENCES spaces(id) ON DELETE SET NULL,
    parent_id       UUID REFERENCES docs(id) ON DELETE CASCADE,
    creator_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    title           TEXT NOT NULL DEFAULT 'Untitled',
    icon            TEXT,
    content         JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_text    TEXT NOT NULL DEFAULT '',
    version         INT  NOT NULL DEFAULT 1,
    archived        BOOLEAN NOT NULL DEFAULT FALSE,
    order_index     INT  NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_docs_workspace ON docs(workspace_id);
CREATE INDEX idx_docs_parent    ON docs(parent_id);
CREATE INDEX idx_docs_space     ON docs(space_id);
