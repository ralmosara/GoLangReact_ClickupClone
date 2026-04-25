CREATE TABLE dashboards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    space_id      UUID REFERENCES spaces(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    creator_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dashboards_workspace ON dashboards(workspace_id);
CREATE INDEX idx_dashboards_space     ON dashboards(space_id);

CREATE TABLE dashboard_widgets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id  UUID NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,          -- burndown | velocity | task_count | time_per_user
    title         TEXT NOT NULL DEFAULT '',
    config        JSONB NOT NULL DEFAULT '{}'::jsonb,
    order_index   INT   NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_widgets_dashboard ON dashboard_widgets(dashboard_id);
