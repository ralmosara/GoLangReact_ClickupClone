DROP INDEX IF EXISTS idx_tasks_status_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS status_id;
DROP INDEX IF EXISTS idx_statuses_list;
DROP TABLE IF EXISTS statuses;
