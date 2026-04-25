ALTER TABLE custom_fields ADD COLUMN list_id     UUID REFERENCES lists(id) ON DELETE CASCADE;
ALTER TABLE custom_fields ADD COLUMN required    BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE custom_fields ADD COLUMN order_index INT     NOT NULL DEFAULT 0;
ALTER TABLE custom_fields ADD COLUMN updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX idx_custom_fields_list      ON custom_fields(list_id);
CREATE INDEX idx_custom_fields_workspace ON custom_fields(workspace_id);
