-- Uhh, rebuilding the schema again.

CREATE TABLE IF NOT EXISTS users (
    slack_id TEXT PRIMARY KEY,
    join_date TIMESTAMPTZ NOT NULL,
    timezone TEXT,
    join_origin TEXT,
    is_restricted BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMPTZ NOT NULL,
    slack_id VARCHAR(18) NOT NULL,
    event_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS single_entry_events (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMPTZ NOT NULL,
    slack_id VARCHAR(18) NOT NULL,
    event_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
    UNIQUE (slack_id, event_name)
)



CREATE INDEX IF NOT EXISTS idx_events_user_time
  ON events (slack_id, event_time);

CREATE INDEX IF NOT EXISTS idx_events_event_name
  ON events (event_name);

CREATE INDEX IF NOT EXISTS idx_events_time
  ON events (event_time);

CREATE INDEX IF NOT EXISTS idx_events_metadata
  ON events USING GIN (metadata);
