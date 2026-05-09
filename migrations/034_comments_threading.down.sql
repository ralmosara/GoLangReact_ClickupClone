-- 034_comments_threading.down.sql — drops threading + reactions.
DROP TABLE IF EXISTS comment_reactions;
DROP INDEX IF EXISTS idx_comments_parent;
ALTER TABLE comments
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS edited_at,
    DROP COLUMN IF EXISTS parent_comment_id;
