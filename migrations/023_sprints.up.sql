CREATE TABLE sprints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id      UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    starts_at    TIMESTAMPTZ NOT NULL,
    ends_at      TIMESTAMPTZ NOT NULL,
    goal_points  INT  NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'planned', -- planned | active | closed
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sprints_list ON sprints(list_id);

ALTER TABLE tasks ADD COLUMN sprint_id UUID REFERENCES sprints(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN points    INT;
CREATE INDEX idx_tasks_sprint ON tasks(sprint_id);
