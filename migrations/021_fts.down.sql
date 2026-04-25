DROP INDEX IF EXISTS idx_messages_search;
ALTER TABLE messages DROP COLUMN IF EXISTS search_tsv;

DROP INDEX IF EXISTS idx_comments_search;
ALTER TABLE comments DROP COLUMN IF EXISTS search_tsv;

DROP INDEX IF EXISTS idx_docs_search;
ALTER TABLE docs DROP COLUMN IF EXISTS search_tsv;

DROP INDEX IF EXISTS idx_tasks_search;
ALTER TABLE tasks DROP COLUMN IF EXISTS search_tsv;
