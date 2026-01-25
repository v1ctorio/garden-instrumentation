package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	. "ingestion-orchid/internal"
)

type Event struct {
	SlackID   string            `json:"slack_id"`
	EventName string            `json:"event_name"`
	Program   *string           `json:"program,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata"`
}

type User struct {
	SlackID    string     `json:"slack_id"`
	JoinDate   *time.Time `json:"timestamp,omitempty"`
	Timezone   string     `json:"timezone,omitempty"`
	JoinOrigin string     `json:"join_origin"`
}

func main() {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	apiKeys := LoadAPIKeys()
	allowedEvents, err := LoadAllowedEvents(ctx, db)

	r := chi.NewRouter()
	r.Use(ApiKeyAuth(apiKeys))

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
