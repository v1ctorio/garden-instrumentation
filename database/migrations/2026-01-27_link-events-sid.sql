ALTER TABLE events
ADD CONSTRAINT fk_events_slack_user
FOREIGN KEY (slack_id)
REFERENCES users (slack_id)
ON DELETE CASCADE;