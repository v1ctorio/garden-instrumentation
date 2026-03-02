-- Uhh, rebuilding the schema again.

CREATE TABLE IF NOT EXISTS users (
    slack_id TEXT PRIMARY KEY,
    join_timestamp TIMESTAMPTZ NOT NULL,
    join_reason TEXT,
    timezone TEXT,
    is_restricted BOOLEAN DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'
);


CREATE TABLE IF NOT EXISTS multi_entry_events (
    id BIGSERIAL PRIMARY KEY,
    event_timestamp TIMESTAMPTZ NOT NULL,
    slack_id TEXT NOT NULL REFERENCES users(slack_id) ON DELETE CASCADE,
    event_kind TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS single_entry_events (
    id BIGSERIAL PRIMARY KEY,
    event_timestamp TIMESTAMPTZ NOT NULL,
    slack_id TEXT NOT NULL REFERENCES users(slack_id) ON DELETE CASCADE,
    event_kind TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    UNIQUE (slack_id, event_kind)
);

-- Indexes for multi_entry_events
CREATE INDEX IF NOT EXISTS idx_multi_events_user_time 
    ON multi_entry_events (slack_id, event_timestamp);

CREATE INDEX IF NOT EXISTS idx_multi_events_kind 
    ON multi_entry_events (event_kind);

CREATE INDEX IF NOT EXISTS idx_multi_events_time 
    ON multi_entry_events (event_timestamp);

CREATE INDEX IF NOT EXISTS idx_multi_events_metadata 
    ON multi_entry_events USING GIN (metadata);

-- Indexes for single_entry_events
CREATE INDEX IF NOT EXISTS idx_single_events_user_time 
    ON single_entry_events (slack_id, event_timestamp);

CREATE INDEX IF NOT EXISTS idx_single_events_kind 
    ON single_entry_events (event_kind);

CREATE INDEX IF NOT EXISTS idx_single_events_time 
    ON single_entry_events (event_timestamp);

CREATE INDEX IF NOT EXISTS idx_single_events_metadata 
    ON single_entry_events USING GIN (metadata);