DROP INDEX IF EXISTS idx_time_entries_running;
ALTER TABLE time_entries DROP COLUMN IF EXISTS updated_at;
ALTER TABLE time_entries DROP COLUMN IF EXISTS billable;
