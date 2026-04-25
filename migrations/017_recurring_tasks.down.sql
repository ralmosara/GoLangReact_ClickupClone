DROP INDEX IF EXISTS idx_tasks_recurring;
DROP INDEX IF EXISTS idx_tasks_recurring_parent;
ALTER TABLE tasks DROP COLUMN IF EXISTS recurring_parent_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS recurring_rule;
