ALTER TABLE tasks
    ALTER COLUMN position TYPE INT USING position::int,
    ALTER COLUMN position SET DEFAULT 0;
