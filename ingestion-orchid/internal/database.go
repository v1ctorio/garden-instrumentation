package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ingestion-orchid/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	SlackID   string            `json:"slack_id"`
	EventKind string            `json:"event_kind"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata"`
}

type User struct {
	SlackID       string            `json:"slack_id"`
	JoinTimestamp *time.Time        `json:"join_timestamp,omitempty"`
	JoinReason    *string           `json:"join_reason,omitempty"`
	Timezone      *string           `json:"timezone"`
	IsRestricted  bool              `json:"is_restricted"`
	Metadata      map[string]string `json:"metadata"`
}

func InsertUser(ctx context.Context, db *pgxpool.Pool, u User) error {
	ts := time.Now().UTC()
	tz := "unknown/unknown"

	if u.JoinTimestamp != nil {
		ts = *u.JoinTimestamp
	}
	if u.Timezone != nil {
		tz = *u.Timezone
	}
	metadata := u.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	_, err := db.Exec(ctx, `
		INSERT INTO users (slack_id, join_timestamp, join_reason, timezone, is_restricted, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slack_id) DO UPDATE
		SET join_reason = EXCLUDED.join_reason,
			timezone = EXCLUDED.timezone,
			is_restricted = EXCLUDED.is_restricted,
			metadata = EXCLUDED.metadata
	`, u.SlackID, ts, u.JoinReason, tz, u.IsRestricted, metadata)
	if err != nil {
		return err
	}

	slog.Info("Inserted user", "slack_id", u.SlackID, "metadata", u.Metadata)
	return nil

}

func RecordEvent(ctx context.Context, db *pgxpool.Pool, e Event) error {
	slog.Info("Received event", "event_kind", e.EventKind, "slack_id", e.SlackID)
	ts := time.Now().UTC()
	if e.Timestamp != nil {
		ts = *e.Timestamp
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to init transaction for event ack", "error", err)
		return err
	}
	defer tx.Rollback(ctx)

	var table string
	table = config.RecognizedEvents[e.EventKind]
	if table == "" {
		slog.Warn("Received unknown event", "event_kind", e.EventKind)

		return fmt.Errorf("Unknown event kind")
	}

	upsertUserSQL := `
			INSERT INTO users (slack_id, join_timestamp, metadata)
			VALUES ($1, NOW(), '{"auto_created": true}')
			ON CONFLICT (slack_id) DO NOTHING;
		`
	_, err = tx.Exec(ctx, upsertUserSQL, e.SlackID)
	if err != nil {
		slog.Error("Failed to upsert user on event ack", "error", err)
		return err
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (event_timestamp, slack_id, event_kind, metadata)
		VALUES ($1, $2, $3, $4)
	`, table), ts, e.SlackID, e.EventKind, e.Metadata)

	slog.Info("Recorded event", "event_kind", e.EventKind, "slack_id", e.SlackID, "table", table)
	return err
}
