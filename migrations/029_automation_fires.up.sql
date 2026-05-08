-- Per-(automation, task) dedup table for the task.due_soon trigger.
--
-- The due-soon scanner runs on a timer and would otherwise re-fire on every
-- tick for the same task. We insert (automation_id, task_id) on first match
-- with ON CONFLICT DO NOTHING; only successful inserts dispatch the event.
--
-- The row is removed if the task changes due_at (or completes), so re-arming
-- the trigger is implicit.
CREATE TABLE automation_fires (
    automation_id   UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    task_id         UUID NOT NULL REFERENCES tasks(id)        ON DELETE CASCADE,
    fired_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (automation_id, task_id)
);

CREATE INDEX idx_automation_fires_task ON automation_fires(task_id);
