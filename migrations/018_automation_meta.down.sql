ALTER TABLE automations DROP COLUMN IF EXISTS conditions;
ALTER TABLE automations DROP COLUMN IF EXISTS run_count;
ALTER TABLE automations DROP COLUMN IF EXISTS last_run_at;
ALTER TABLE automations DROP COLUMN IF EXISTS description;
