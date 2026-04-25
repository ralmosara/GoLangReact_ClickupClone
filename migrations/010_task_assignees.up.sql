CREATE TABLE task_assignees (
    task_id     UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_assignees_user ON task_assignees(user_id);

-- Backfill from legacy single-assignee column; keep legacy column populated for back-compat.
INSERT INTO task_assignees (task_id, user_id, assigned_at)
SELECT id, assignee_id, COALESCE(created_at, NOW()) FROM tasks
WHERE assignee_id IS NOT NULL
ON CONFLICT DO NOTHING;
