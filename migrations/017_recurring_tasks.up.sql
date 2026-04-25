ALTER TABLE tasks ADD COLUMN recurring_rule      TEXT;
ALTER TABLE tasks ADD COLUMN recurring_parent_id UUID REFERENCES tasks(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_recurring_parent ON tasks(recurring_parent_id) WHERE recurring_parent_id IS NOT NULL;
CREATE INDEX idx_tasks_recurring        ON tasks(id) WHERE recurring_rule IS NOT NULL;
