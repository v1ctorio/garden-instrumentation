CREATE TABLE IF NOT EXISTS allowed_events (
  event_name TEXT PRIMARY KEY,
  unique_per_user BOOLEAN NOT NULL DEFAULT false
);


INSERT INTO allowed_events (event_name, unique_per_user)
VALUES
  ('wthc_message_sent', true),
  ('introduction_sent', true)
ON CONFLICT (event_name)
DO UPDATE SET
  unique_per_user = EXCLUDED.unique_per_user;

DO $$
DECLARE
  events text;
BEGIN
  SELECT string_agg(quote_literal(event_name), ', ')
  INTO events
  FROM allowed_events
  WHERE unique_per_user;

  DROP INDEX IF EXISTS unique_once_events;
  
  EXECUTE format(
    'CREATE UNIQUE INDEX IF NOT EXISTS unique_once_events
     ON events (slack_id, event_name)
     WHERE event_name IN (%s)',
    events
  );
END $$;