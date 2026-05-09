-- 034_comments_threading.up.sql
--
-- Adds threading, edit history, and emoji reactions to comments.
--
--   * parent_comment_id: NULL for top-level comments, otherwise points
--     at the comment this one replies to. Enforced single-level on the
--     application side — replies can't reply (nested 1 deep, like Slack
--     threads). The schema permits deeper trees so a future "deeper
--     thread" UX is a code-only change.
--
--   * edited_at: set every time the body changes via Update. NULL means
--     "never edited" — the UI uses that to decide whether to show the
--     "(edited)" suffix.
--
--   * deleted_at + deleted_by: soft-delete instead of DELETE so a
--     "[Comment removed]" placeholder can render in the thread without
--     orphaning replies. The repo treats deleted rows as opaque.
--
--   * comment_reactions: one row per (comment, user, emoji) — uniqueness
--     enforced so the same user can't double-react with the same emoji.
--     Aggregate counts are computed on read; no denormalised counter
--     column to keep right at insert time.

ALTER TABLE comments
    ADD COLUMN IF NOT EXISTS parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS edited_at         TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_at        TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_by        UUID REFERENCES users(id) ON DELETE SET NULL;

-- Reply-list lookup hits this composite ("get all children of comment X
-- ordered by creation"). Top-level lookups still use idx_comments_task.
CREATE INDEX IF NOT EXISTS idx_comments_parent
    ON comments(parent_comment_id, created_at)
    WHERE parent_comment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS comment_reactions (
    comment_id  UUID        NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    emoji       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (comment_id, user_id, emoji)
);

-- Listing reactions for a comment is the dominant read pattern; the
-- (comment_id, emoji) prefix is enough to drive the GROUP BY emoji
-- aggregation efficiently.
CREATE INDEX IF NOT EXISTS idx_comment_reactions_comment
    ON comment_reactions(comment_id, emoji);
