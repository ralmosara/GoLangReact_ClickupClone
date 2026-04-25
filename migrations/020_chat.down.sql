DROP INDEX IF EXISTS idx_messages_parent;
DROP INDEX IF EXISTS idx_messages_channel;
DROP TABLE IF EXISTS messages;

DROP INDEX IF EXISTS idx_channel_members_user;
DROP TABLE IF EXISTS channel_members;

DROP INDEX IF EXISTS idx_channels_space;
DROP INDEX IF EXISTS idx_channels_workspace;
DROP TABLE IF EXISTS channels;
