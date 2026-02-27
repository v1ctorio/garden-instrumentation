package internal

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	SlackID   string            `json:"slack_id"`
	EventName string            `json:"event_name"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata"`
}

type User struct {
	SlackID      string     `json:"slack_id"`
	JoinDate     *time.Time `json:"timestamp,omitempty"`
	Timezone     *string    `json:"timezone"`
	JoinOrigin   *string    `json:"join_origin,omitempty"`
	IsRestricted bool       `json:"is_restricted"`
}

func LoadAllowedEvents(ctx context.Context, db *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := db.Query(ctx, `
	SELECT event_name
	FROM allowed_events`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make(map[string]struct{})

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		events[name] = struct{}{}
	}

	return events, rows.Err()
}

func InsertUser(ctx context.Context, db *pgxpool.Pool, u User) error {
	var old_jo string
	err := db.QueryRow(ctx, `
	SELECT join_origin from users where slack_id=$1
	`, u.SlackID).Scan(&old_jo)
	ts := time.Now().UTC()
	jo := "unknown"
	tz := "unknown/unknown"

	if u.JoinDate != nil {
		ts = *u.JoinDate
	}
	if u.JoinOrigin != nil {
		jo = *u.JoinOrigin
	}

	if u.Timezone != nil {
		tz = *u.Timezone
	}

	if old_jo != "" && jo == "unknown" {
		return fmt.Errorf("User already registered")
	}

	if old_jo != "unknown" && jo != "unknown" {
		return fmt.Errorf("User already registered")
	}

	if old_jo == "unknown" && jo != "unknown" {
		_, err = db.Exec(ctx, `
		UPDATE users
		SET join_origin=$1
		WHERE slack_id=$2`,
			jo, u.SlackID)
		return err
	}

	_, err = db.Exec(ctx, `
	INSERT INTO users (slack_id, join_date, timezone, join_origin, is_restricted)
	values ($1, $2, $3, $4, $5)
	`, u.SlackID, ts, tz, jo, u.IsRestricted)

	return err
}

func RecordEvent(ctx context.Context, db *pgxpool.Pool, e Event) error {
	ts := time.Now().UTC()
	if e.Timestamp != nil {
		ts = *e.Timestamp
	}
	var table string

	switch {
	case slices.Contains(ValidMultiEntryEvents, e.EventName):
		table = "events"
	case slices.Contains(ValidSingleEntryEvents, e.EventName):
		table = "single_entry_events"
	default:
		return fmt.Errorf("invalid event name provided")
	}

	_, err := db.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (event_time, slack_id, event_name, metadata)
		VALUES ($1, $2, $3, $4)
	`, table), ts, e.SlackID, e.EventName, e.Metadata)
	return err
}
