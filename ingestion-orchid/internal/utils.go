package internal

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadAPIKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	raw := os.Getenv("INGEST_API_KEYS")

	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = struct{}{}
		}
	}

	return keys
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
