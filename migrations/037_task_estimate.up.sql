-- 037_task_estimate.up.sql
--
-- Adds time_estimate_s to tasks. This is the planned duration in
-- seconds the assignee has committed to — distinct from `points`,
-- which is the unitless sprint-velocity number. Both can co-exist:
-- a task can be 5 points AND estimated at 3h.
--
-- The workload/capacity view sums this column per assignee to render
-- "Alice has 47h booked this week, Bob has 12h." Without an estimate
-- the workload view falls back to counting tasks (less useful but
-- still a signal).
--
-- Stored as INTEGER seconds rather than INTERVAL so cross-DB tooling
-- (analytics dumps, CSV exports) keeps clean numeric columns.

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS time_estimate_s INTEGER;

-- Optional CHECK to catch obvious garbage (negative, > 1 year). The
-- 1y ceiling is generous — it's about catching a unit-mismatch bug
-- ("user typed 7200 thinking minutes; we stored seconds") without
-- restricting legitimate long-running estimates.
ALTER TABLE tasks
    ADD CONSTRAINT tasks_time_estimate_sane
    CHECK (time_estimate_s IS NULL OR (time_estimate_s >= 0 AND time_estimate_s <= 31536000));
