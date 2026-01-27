CREATE TABLE IF NOT EXISTS users (
    slack_id TEXT PRIMARY KEY,
    join_date TIMESTAMPTZ NOT NULL,
    timezone TEXT,
    join_origin TEXT
);

CREATE TABLE IF NOT EXISTS  events (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMPTZ NOT NULL,
    slack_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);


CREATE TABLE IF NOT EXISTS allowed_events (
    event_name TEXT PRIMARY KEY
);

INSERT INTO allowed_events VALUES 
    ('wthc_message_sent'),
    ('introduction_sent');


ALTER TABLE events
ADD CONSTRAINT fk_events_event_name
FOREIGN KEY (event_name)
REFERENCES allowed_events (event_name);

CREATE INDEX IF NOT EXISTS idx_events_user_time
  ON events (slack_id, event_time);

CREATE INDEX IF NOT EXISTS idx_events_event_name
  ON events (event_name);

CREATE INDEX IF NOT EXISTS idx_events_time
  ON events (event_time);

CREATE INDEX IF NOT EXISTS idx_events_metadata
  ON events USING GIN (metadata);
