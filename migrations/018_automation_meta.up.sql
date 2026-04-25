ALTER TABLE automations ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE automations ADD COLUMN last_run_at TIMESTAMPTZ;
ALTER TABLE automations ADD COLUMN run_count   INT NOT NULL DEFAULT 0;
ALTER TABLE automations ADD COLUMN conditions  JSONB NOT NULL DEFAULT '[]'::jsonb;
