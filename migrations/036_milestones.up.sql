-- 036_milestones.up.sql
--
-- Milestones are date-anchored deliverables ("v2.0 ships Aug 15") and
-- live at the workspace level rather than the list level. They differ
-- from sprints in two ways:
--
--   * Sprints are time-boxed iterations with a goal and a burndown.
--     Milestones are a single date the team is shipping toward.
--
--   * Sprints belong to one list. Milestones can pull tasks from any
--     list in the workspace — release planning is rarely confined to
--     a single backlog.
--
-- A task can belong to multiple milestones (a "v2.0" milestone and a
-- "Mobile 1.0" milestone might both ship the same payment-flow task),
-- so the relationship is N:M.

CREATE TABLE IF NOT EXISTS milestones (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    -- ISO-3166 hex for the colored chip in the UI. Optional — UI falls
    -- back to a derived color from the milestone id when null.
    color         TEXT,
    -- The single date the team is shipping toward. Required — a
    -- milestone without a date is a goal, not a milestone.
    due_at        DATE        NOT NULL,
    -- Lifecycle: 'planned' (default) → 'in_progress' → 'shipped' /
    -- 'cancelled'. The shipped/cancelled rows are preserved (not
    -- deleted) so the historical roadmap stays intact.
    status        TEXT        NOT NULL DEFAULT 'planned'
                              CHECK (status IN ('planned', 'in_progress', 'shipped', 'cancelled')),
    owner_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Roadmap timeline view orders by due_at; the index covers it. Filter
-- on workspace_id is cardinal so it leads.
CREATE INDEX IF NOT EXISTS idx_milestones_workspace_due
    ON milestones (workspace_id, due_at);

-- N:M task ↔ milestone link. ON DELETE CASCADE on both sides keeps
-- the join rows in sync without separate cleanup paths.
CREATE TABLE IF NOT EXISTS milestone_tasks (
    milestone_id  UUID        NOT NULL REFERENCES milestones(id) ON DELETE CASCADE,
    task_id       UUID        NOT NULL REFERENCES tasks(id)      ON DELETE CASCADE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (milestone_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_milestone_tasks_task
    ON milestone_tasks (task_id);
