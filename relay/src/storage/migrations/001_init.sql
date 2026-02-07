CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    pubkey TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    kind INTEGER NOT NULL,
    tags JSONB NOT NULL,
    content TEXT NOT NULL,
    sig TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_pubkey_kind_created_at
    ON events (pubkey, kind, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_events_created_at
    ON events (created_at DESC);

CREATE TABLE IF NOT EXISTS event_tags (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    tag_index INTEGER NOT NULL,
    tag_name TEXT NOT NULL,
    tag_value TEXT NOT NULL DEFAULT '',
    tag_array JSONB NOT NULL,
    UNIQUE (event_id, tag_index)
);

CREATE INDEX IF NOT EXISTS idx_event_tags_name_value
    ON event_tags (tag_name, tag_value);

CREATE TABLE IF NOT EXISTS deleted_events (
    event_id TEXT PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    deleted_at BIGINT NOT NULL,
    deleted_by TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    about TEXT NOT NULL DEFAULT '',
    picture TEXT NOT NULL DEFAULT '',
    geohash TEXT NOT NULL DEFAULT '',
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_restricted BOOLEAN NOT NULL DEFAULT FALSE,
    is_vetted BOOLEAN NOT NULL DEFAULT FALSE,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at BIGINT NOT NULL,
    created_by TEXT NOT NULL,
    updated_at BIGINT NOT NULL,
    updated_by TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_groups_updated_at
    ON groups (updated_at DESC);

CREATE TABLE IF NOT EXISTS group_roles (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    role_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    created_at BIGINT NOT NULL,
    created_by TEXT NOT NULL,
    updated_at BIGINT NOT NULL,
    updated_by TEXT NOT NULL,
    PRIMARY KEY (group_id, role_name)
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    pubkey TEXT NOT NULL,
    added_at BIGINT NOT NULL,
    added_by TEXT NOT NULL,
    role_name TEXT NOT NULL DEFAULT '',
    promoted_at BIGINT NOT NULL DEFAULT 0,
    promoted_by TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_id, pubkey)
);

CREATE TABLE IF NOT EXISTS group_bans (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    pubkey TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    banned_at BIGINT NOT NULL,
    banned_by TEXT NOT NULL,
    expires_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (group_id, pubkey)
);

CREATE TABLE IF NOT EXISTS group_invites (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    expires_at BIGINT NOT NULL DEFAULT 0,
    max_usage_count INTEGER NOT NULL DEFAULT 0,
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    created_by TEXT NOT NULL,
    PRIMARY KEY (group_id, code),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS group_join_requests (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    pubkey TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (group_id, pubkey)
);

CREATE TABLE IF NOT EXISTS group_events (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (group_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_group_events_event_id
    ON group_events (event_id);
