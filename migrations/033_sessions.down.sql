-- 033_sessions.down.sql — drops the active-sessions table.
DROP INDEX IF EXISTS user_sessions_expires_idx;
DROP INDEX IF EXISTS user_sessions_user_id_seen_idx;
DROP TABLE IF EXISTS user_sessions;
