-- All of the previous migrations bundled on something not as shitty

CREATE TABLE IF NOT EXISTS users (
    slack_id TEXT PRIMARY KEY,
    join_date TIMESTAMPTZ NOT NULL,
    timezone TEXT,
    join_origin TEXT,
    is_restricted BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS allowed_events (
    event_name TEXT PRIMARY KEY,
    unique_per_user BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMPTZ NOT NULL,
    slack_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);


ALTER TABLE events
  DROP CONSTRAINT IF EXISTS fk_events_event_name; -- Safety check if re-running

ALTER TABLE events
  ADD CONSTRAINT fk_events_event_name
  FOREIGN KEY (event_name)
  REFERENCES allowed_events (event_name);

ALTER TABLE events
ADD CONSTRAINT fk_events_slack_user
FOREIGN KEY (slack_id)
REFERENCES users (slack_id)
ON DELETE CASCADE;


INSERT INTO allowed_events (event_name, unique_per_user)
VALUES
  ('wthc_message_sent', true),
  ('introduction_sent', true)
ON CONFLICT (event_name)
DO UPDATE SET
  unique_per_user = EXCLUDED.unique_per_user;


CREATE INDEX IF NOT EXISTS idx_events_user_time
  ON events (slack_id, event_time);

CREATE INDEX IF NOT EXISTS idx_events_event_name
  ON events (event_name);

CREATE INDEX IF NOT EXISTS idx_events_time
  ON events (event_time);

CREATE INDEX IF NOT EXISTS idx_events_metadata
  ON events USING GIN (metadata);


DO $$
DECLARE
  events_list text;
BEGIN
  SELECT string_agg(quote_literal(event_name), ', ')
  INTO events_list
  FROM allowed_events
  WHERE unique_per_user;

  IF events_list IS NOT NULL THEN
    
    DROP INDEX IF EXISTS unique_once_events;
  
    EXECUTE format(
      'CREATE UNIQUE INDEX unique_once_events
       ON events (slack_id, event_name)
       WHERE event_name IN (%s)',
      events_list
    );
    
  END IF;
END $$;