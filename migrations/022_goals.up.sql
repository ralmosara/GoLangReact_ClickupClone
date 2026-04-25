CREATE TABLE goals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_goal_id  UUID REFERENCES goals(id) ON DELETE CASCADE,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    start_at        TIMESTAMPTZ,
    due_at          TIMESTAMPTZ,
    archived        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_goals_workspace ON goals(workspace_id);
CREATE INDEX idx_goals_parent    ON goals(parent_goal_id);

-- Targets are the measurable units inside a goal. Kind determines which
-- *_target / *_current columns are meaningful:
--   number / currency → target_number + current_number
--   boolean           → target_boolean + current_boolean
--   task_completed    → task_list_id (progress = completed / total tasks in list)
CREATE TABLE goal_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id         UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL,       -- number | currency | boolean | task_completed
    target_number   DOUBLE PRECISION,
    current_number  DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency        TEXT,
    target_boolean  BOOLEAN,
    current_boolean BOOLEAN NOT NULL DEFAULT FALSE,
    task_list_id    UUID REFERENCES lists(id) ON DELETE SET NULL,
    order_index     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_goal_targets_goal ON goal_targets(goal_id);
