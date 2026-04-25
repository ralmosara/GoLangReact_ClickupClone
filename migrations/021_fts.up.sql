-- Full-text search: one generated tsvector column per searchable entity.
-- We use stored generated columns so a GIN index stays in sync automatically —
-- no triggers needed for the common cases.

ALTER TABLE tasks
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_tasks_search ON tasks USING GIN (search_tsv);

ALTER TABLE docs
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content_text, '')), 'B')
    ) STORED;
CREATE INDEX idx_docs_search ON docs USING GIN (search_tsv);

ALTER TABLE comments
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(body, ''))
    ) STORED;
CREATE INDEX idx_comments_search ON comments USING GIN (search_tsv);

ALTER TABLE messages
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(body, ''))
    ) STORED;
CREATE INDEX idx_messages_search ON messages USING GIN (search_tsv);
