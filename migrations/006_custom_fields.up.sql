CREATE TABLE custom_fields (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE task_custom_values (
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    field_id        UUID NOT NULL REFERENCES custom_fields(id) ON DELETE CASCADE,
    value           JSONB NOT NULL DEFAULT 'null'::jsonb,
    PRIMARY KEY (task_id, field_id)
);

CREATE TABLE views (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id         UUID REFERENCES lists(id) ON DELETE CASCADE,
    space_id        UUID REFERENCES spaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'board',
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_views_list  ON views(list_id);
CREATE INDEX idx_views_space ON views(space_id);
