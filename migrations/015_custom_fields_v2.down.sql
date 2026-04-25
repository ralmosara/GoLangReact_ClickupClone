DROP INDEX IF EXISTS idx_custom_fields_workspace;
DROP INDEX IF EXISTS idx_custom_fields_list;
ALTER TABLE custom_fields DROP COLUMN IF EXISTS updated_at;
ALTER TABLE custom_fields DROP COLUMN IF EXISTS order_index;
ALTER TABLE custom_fields DROP COLUMN IF EXISTS required;
ALTER TABLE custom_fields DROP COLUMN IF EXISTS list_id;
