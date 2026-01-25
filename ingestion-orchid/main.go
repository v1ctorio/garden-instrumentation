package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	SlackID   string            `json:"slack_id"`
	EventName string            `json:"event_name"`
	Program   *string           `json:"program,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata"`
}

func main() {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	apiKeys := loadAPIKeys()
	allowedEvents, err := loadAllowedEvents(ctx, db)

	r := chi.NewRouter()
	r.Use(apiKeyAuth(apiKeys))

	r.Post("/instrumentation/event", eventHandler(db, allowedEvents))

	fmt.Println("Hello chat, listening on :8400")

	http.ListenAndServe(":8400", r)
}

func insertEvent(ctx context.Context, db *pgxpool.Pool, e Event) error {
	ts := time.Now().UTC()
	if e.Timestamp != nil {
		ts = *e.Timestamp
	}

	_, err := db.Exec(ctx, `
		INSERT INTO events (event_time, user_id, event_name, program, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, ts, e.SlackID, e.EventName, e.Program, e.Metadata)
	return err
}

func eventHandler(db *pgxpool.Pool, allowed map[string]struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e Event

		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if _, ok := allowed[e.EventName]; !ok {
			http.Error(w, "unknwon event_name", http.StatusBadRequest)
		}

		if e.SlackID == "" || e.EventName == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
		}

		if e.Metadata == nil {
			e.Metadata = map[string]string{}
		}

		if err := insertEvent(r.Context(), db, e); err != nil {
			http.Error(w, "failed to store event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func apiKeyAuth(validKeys map[string]struct{}) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}

			if _, present := validKeys[key]; !present {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}

			h.ServeHTTP(w, r)

		})
	}
}

func loadAPIKeys() map[string]struct{} {
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

func loadAllowedEvents(ctx context.Context, db *pgxpool.Pool) (map[string]struct{}, error) {
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
