-- Widen tasks.position so we can gap-insert on drag-drop without rewriting N rows.
ALTER TABLE tasks
    ALTER COLUMN position TYPE DOUBLE PRECISION USING position::double precision,
    ALTER COLUMN position SET DEFAULT 1000;

-- Seed existing rows to a spaced-out sequence per (list_id, status) grouping so initial order is preserved.
WITH ordered AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY list_id, status ORDER BY position, created_at) * 1000.0 AS new_pos
    FROM tasks
)
UPDATE tasks SET position = ordered.new_pos
FROM ordered WHERE tasks.id = ordered.id;
