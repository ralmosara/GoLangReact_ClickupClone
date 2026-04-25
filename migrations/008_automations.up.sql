CREATE TABLE automations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    list_id         UUID REFERENCES lists(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    trigger         JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions         JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_automations_workspace ON automations(workspace_id);
CREATE INDEX idx_automations_list      ON automations(list_id);
