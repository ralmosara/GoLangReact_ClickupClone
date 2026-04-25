CREATE TABLE channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    space_id        UUID REFERENCES spaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    topic           TEXT NOT NULL DEFAULT '',
    kind            TEXT NOT NULL DEFAULT 'channel',   -- channel | dm
    is_private      BOOLEAN NOT NULL DEFAULT FALSE,
    creator_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_channels_workspace ON channels(workspace_id);
CREATE INDEX idx_channels_space     ON channels(space_id);

CREATE TABLE channel_members (
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_at    TIMESTAMPTZ,
    PRIMARY KEY (channel_id, user_id)
);
CREATE INDEX idx_channel_members_user ON channel_members(user_id);

CREATE TABLE messages (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id         UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_message_id  UUID REFERENCES messages(id) ON DELETE CASCADE,
    body               TEXT NOT NULL,
    edited_at          TIMESTAMPTZ,
    deleted_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_channel ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_parent  ON messages(parent_message_id);
