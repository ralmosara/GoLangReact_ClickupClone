DROP INDEX IF EXISTS idx_notifications_entity;
ALTER TABLE notifications DROP COLUMN IF EXISTS entity_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS entity_type;
ALTER TABLE notifications DROP COLUMN IF EXISTS actor_id;
