CREATE TABLE whiteboards (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    space_id     UUID REFERENCES spaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
    version      INT  NOT NULL DEFAULT 1,
    creator_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_whiteboards_workspace ON whiteboards(workspace_id);
CREATE INDEX idx_whiteboards_space     ON whiteboards(space_id);
