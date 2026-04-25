CREATE TABLE statuses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id     UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       TEXT NOT NULL DEFAULT '#94a3b8',
    category    TEXT NOT NULL DEFAULT 'active',
    order_index INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (list_id, name)
);

CREATE INDEX idx_statuses_list ON statuses(list_id, order_index);

ALTER TABLE tasks ADD COLUMN status_id UUID REFERENCES statuses(id) ON DELETE SET NULL;
CREATE INDEX idx_tasks_status_id ON tasks(status_id);
