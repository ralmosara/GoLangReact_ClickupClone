ALTER TABLE time_entries ADD COLUMN billable BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE time_entries ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Partial index over running timers (stopped_at IS NULL) so we can cheaply
-- find all active timers per user.
CREATE INDEX idx_time_entries_running ON time_entries(user_id) WHERE stopped_at IS NULL;
