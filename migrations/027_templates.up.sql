CREATE TABLE templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL,                    -- list | doc | task
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
    creator_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_templates_workspace ON templates(workspace_id);
CREATE INDEX idx_templates_kind      ON templates(workspace_id, kind);
