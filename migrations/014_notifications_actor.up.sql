ALTER TABLE notifications ADD COLUMN actor_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE notifications ADD COLUMN entity_type TEXT NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN entity_id   UUID;

CREATE INDEX idx_notifications_entity ON notifications(entity_type, entity_id);
