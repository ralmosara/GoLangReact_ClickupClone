ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_time_estimate_sane;
ALTER TABLE tasks DROP COLUMN IF EXISTS time_estimate_s;
