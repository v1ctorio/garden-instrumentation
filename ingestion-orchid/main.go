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
	Timezone   *string    `json:"timezone"`
	JoinOrigin *string    `json:"join_origin,omitempty"`
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

	r.Get("/health", healthcheckHandler(db))

	r.Route("/instrumentation", func(r chi.Router) {
		r.Use(ApiKeyAuth(apiKeys))
		r.Post("/user", userHandler(db))
		r.Post("/event", eventHandler(db, allowedEvents))

	})

	fmt.Println("Hello chat, listening on :8400")

	http.ListenAndServe(":8400", r)
}

func insertUser(ctx context.Context, db *pgxpool.Pool, u User) error {
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

	_, err := db.Exec(ctx, `
	INSERT INTO users (slack_id, join_date, timezone, join_origin)
	values ($1, $2, $3, $4)
	`, u.SlackID, ts, tz, jo)

	return err
}

func userHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if u.SlackID == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
			return
		}

		if err := insertUser(r.Context(), db, u); err != nil {
			http.Error(w, "failed to store user", http.StatusInternalServerError)
			return
		}

		LogToSlack("New user recorded")
		w.WriteHeader(http.StatusAccepted)
	}
}

func insertEvent(ctx context.Context, db *pgxpool.Pool, e Event) error {
	ts := time.Now().UTC()
	if e.Timestamp != nil {
		ts = *e.Timestamp
	}

	_, err := db.Exec(ctx, `
		INSERT INTO events (event_time, slack_id, event_name, program, metadata)
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

		LogToSlack("New event recorded")
		w.WriteHeader(http.StatusAccepted)
	}
}

func healthcheckHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.Ping(r.Context())

		if err != nil {
			http.Error(w, "bad", http.StatusInternalServerError)
			fmt.Printf("Healthcheck failed: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
